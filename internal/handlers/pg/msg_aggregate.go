// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"

	"github.com/FerretDB/FerretDB/internal/handlers/common"
	"github.com/FerretDB/FerretDB/internal/handlers/common/aggregations"
	"github.com/FerretDB/FerretDB/internal/handlers/commonerrors"
	"github.com/FerretDB/FerretDB/internal/handlers/pg/pgdb"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/wire"
)

// MsgAggregate implements HandlerInterface.
func (h *Handler) MsgAggregate(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	dbPool, err := h.DBPool(ctx)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	// TODO https://github.com/FerretDB/FerretDB/issues/1892
	common.Ignored(document, h.L, "cursor", "lsid")

	if err = common.Unimplemented(document, "explain", "collation", "let"); err != nil {
		return nil, err
	}

	common.Ignored(
		document, h.L,
		"allowDiskUse", "maxTimeMS", "bypassDocumentValidation", "readConcern", "hint", "comment", "writeConcern",
	)

	var db string

	if db, err = common.GetRequiredParam[string](document, "$db"); err != nil {
		return nil, err
	}

	collectionParam, err := document.Get(document.Command())
	if err != nil {
		return nil, err
	}

	// TODO handle collection-agnostic pipelines ({aggregate: 1})
	// https://github.com/FerretDB/FerretDB/issues/1890
	var ok bool
	var collection string

	if collection, ok = collectionParam.(string); !ok {
		return nil, commonerrors.NewCommandErrorMsgWithArgument(
			commonerrors.ErrFailedToParse,
			"Invalid command format: the 'aggregate' field must specify a collection name or 1",
			document.Command(),
		)
	}

	pipeline, err := common.GetRequiredParam[*types.Array](document, "pipeline")
	if err != nil {
		return nil, commonerrors.NewCommandErrorMsgWithArgument(
			commonerrors.ErrTypeMismatch,
			"'pipeline' option must be specified as an array",
			document.Command(),
		)
	}

	stages := must.NotFail(iterator.ConsumeValues(pipeline.Iterator()))
	stagesDocuments := make([]aggregations.Stage, len(stages))
	stagesStats := make([]aggregations.Stage, len(stages))

	for _, d := range stages {
		d, ok := d.(*types.Document)
		if !ok {
			return nil, commonerrors.NewCommandErrorMsgWithArgument(
				commonerrors.ErrTypeMismatch,
				"Each element of the 'pipeline' array must be an object",
				document.Command(),
			)
		}

		var s aggregations.Stage

		if s, err = aggregations.NewStage(d); err != nil {
			return nil, err
		}

		switch s.Type() {
		case aggregations.StageTypeDocuments:
			stagesDocuments = append(stagesDocuments, s)
		case aggregations.StageTypeStats:
			stagesStats = append(stagesStats, s)
		default:
			panic(fmt.Sprintf("unknown stage type: %v", s.Type()))
		}
	}

	var resDocs []*types.Document

	if len(stagesDocuments) > 0 {
		qp := pgdb.QueryParams{
			DB:         db,
			Collection: collection,
		}

		qp.Filter = aggregations.GetPushdownQuery(stages)

		var err error
		if resDocs, err = processStagesDocuments(ctx, dbPool, &qp, stagesDocuments); err != nil {
			return nil, err
		}
	}

	if len(stagesStats) > 0 {
		statistics := aggregations.GetStatistics(stagesStats)

		if resDocs, err = processStagesStats(ctx, dbPool, statistics, db, collection, stagesStats); err != nil {
			return nil, err
		}
	}

	// TODO https://github.com/FerretDB/FerretDB/issues/1892
	firstBatch := types.MakeArray(len(resDocs))
	for _, doc := range resDocs {
		firstBatch.Append(doc)
	}

	var reply wire.OpMsg
	must.NoError(reply.SetSections(wire.OpMsgSection{
		Documents: []*types.Document{must.NotFail(types.NewDocument(
			"cursor", must.NotFail(types.NewDocument(
				"firstBatch", firstBatch,
				"id", int64(0),
				"ns", db+"."+collection,
			)),
			"ok", float64(1),
		))},
	}))

	return &reply, nil
}

func processStagesDocuments(ctx context.Context, dbPool *pgdb.Pool, qp *pgdb.QueryParams, stages []aggregations.Stage) ([]*types.Document, error) { //nolint:lll // for readability
	var docs []*types.Document

	if err := dbPool.InTransaction(ctx, func(tx pgx.Tx) error {
		iter, getErr := pgdb.QueryDocuments(ctx, tx, qp)
		if getErr != nil {
			return getErr
		}

		var err error
		docs, err = iterator.ConsumeValues(iterator.Interface[struct{}, *types.Document](iter))
		return err
	}); err != nil {
		return nil, err
	}

	for _, s := range stages {
		var err error
		if docs, err = s.Process(ctx, docs); err != nil {
			return nil, err
		}
	}

	return docs, nil
}

func processStagesStats(ctx context.Context, dbPool *pgdb.Pool, statistics map[aggregations.Statistic]struct{}, db, collection string, stages []aggregations.Stage) ([]*types.Document, error) {
	var docs []*types.Document

	_, hasCount := statistics[aggregations.StatisticCount]
	_, hasStorage := statistics[aggregations.StatisticStorage]

	var dbStats *pgdb.DBStats
	var err error

	if hasCount || hasStorage {
		dbStats, err = dbPool.Stats(ctx, db, collection)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}
	}

	for stat := range statistics {
		switch stat {
		case aggregations.StatisticCount:
			docs = append(docs, must.NotFail(types.NewDocument(
				"type", int32(aggregations.StatisticCount),
				"value", float64(dbStats.CountRows),
			)))
		case aggregations.StatisticStorage:
			docs = append(docs, must.NotFail(types.NewDocument(
				"type", int32(aggregations.StatisticStorage),
				"value", float64(dbStats.SizeTotal),
			)))
		default:
			panic(fmt.Sprintf("unknown statistic: %v", stat))
		}
	}

	var res []*types.Document
	for _, s := range stages {
		var err error
		if res, err = s.Process(ctx, docs); err != nil {
			return nil, err
		}
	}

	return res, nil
}
