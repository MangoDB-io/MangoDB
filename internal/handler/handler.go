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

// Package handler provides a universal handler implementation for all backends.
package handler

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/FerretDB/FerretDB/internal/backends"
	"github.com/FerretDB/FerretDB/internal/backends/decorators/oplog"
	"github.com/FerretDB/FerretDB/internal/clientconn/conninfo"
	"github.com/FerretDB/FerretDB/internal/clientconn/connmetrics"
	"github.com/FerretDB/FerretDB/internal/clientconn/cursor"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/state"
)

// Parts of Prometheus metric names.
const (
	namespace = "ferretdb"
	subsystem = "handler"
)

// Handler provides a set of methods to process clients' requests sent over wire protocol.
//
// MsgXXX methods handle OP_MSG commands.
// CmdQuery handles a limited subset of OP_QUERY messages.
//
// Handler instance is shared between all client connections.
type Handler struct {
	*NewOpts

	b backends.Backend

	cursors  *cursor.Registry
	commands map[string]command

	cappedCleanupStop             chan struct{}
	cleanupCappedCollectionsDocs  *prometheus.CounterVec
	cleanupCappedCollectionsBytes *prometheus.CounterVec
}

// NewOpts represents handler configuration.
//
//nolint:vet // for readability
type NewOpts struct {
	Backend backends.Backend

	L             *zap.Logger
	ConnMetrics   *connmetrics.ConnMetrics
	StateProvider *state.Provider

	// test options
	DisablePushdown         bool
	EnableOplog             bool
	CappedCleanupInterval   time.Duration
	CappedCleanupPercentage uint8
	EnableNewAuth           bool
}

// New returns a new handler.
func New(opts *NewOpts) (*Handler, error) {
	b := opts.Backend

	if opts.EnableOplog {
		b = oplog.NewBackend(b, opts.L.Named("oplog"))
	}

	if opts.CappedCleanupPercentage > 100 {
		opts.CappedCleanupPercentage = 100
	}

	h := &Handler{
		b:       b,
		NewOpts: opts,
		cursors: cursor.NewRegistry(opts.L.Named("cursors")),

		cappedCleanupStop: make(chan struct{}),
		cleanupCappedCollectionsDocs: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cleanup_capped_docs",
				Help:      "Total number of documents deleted in capped collections during cleanup.",
			},
			[]string{"db", "collection"},
		),
		cleanupCappedCollectionsBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cleanup_capped_bytes",
				Help:      "Total number of bytes freed in capped collections during cleanup.",
			},
			[]string{"db", "collection"},
		),
	}

	h.initCommands()

	if opts.EnableOplog {
		// FIXME: how to test
		// mongosh --port=27017 --eval="db.createCollection('coll', {capped: true, size: 1000, max: 100})" cleanup
		//
		// for in in (seq 100)
		//   mongosh --port=27017 --eval="db.coll.insert({data: 'document $i'})" cleanup
		// end
		//
		// fixme:
		// - Run cleanup in embedded?(flag)

		go func() {
			ticker := time.NewTicker(opts.CappedCleanupInterval)
			for {
				select {
				case <-ticker.C:
					if err := h.cleanupAllCappedCollections(context.Background()); err != nil {
						h.L.Error("Failed to cleanup capped collections", zap.Error(err))
					}

				case <-h.cappedCleanupStop:
					ticker.Stop()
					h.L.Debug("The routine to cleanup capped collections is stopped")
					return
				}
			}
		}()
	}

	return h, nil
}

// Close gracefully shutdowns handler.
// It should be called after listener closes all client connections and stops listening.
func (h *Handler) Close() {
	h.cursors.Close()
	h.cappedCleanupStop <- struct{}{}
}

// Describe implements prometheus.Collector interface.
func (h *Handler) Describe(ch chan<- *prometheus.Desc) {
	h.b.Describe(ch)
	h.cursors.Describe(ch)
	h.cleanupCappedCollectionsDocs.Describe(ch)
	h.cleanupCappedCollectionsBytes.Describe(ch)
}

// Collect implements prometheus.Collector interface.
func (h *Handler) Collect(ch chan<- prometheus.Metric) {
	h.b.Collect(ch)
	h.cursors.Collect(ch)
	h.cleanupCappedCollectionsDocs.Collect(ch)
	h.cleanupCappedCollectionsBytes.Collect(ch)
}

// cleanupAllCappedCollections drops the given percent of documents from all capped collections.
func (h *Handler) cleanupAllCappedCollections(ctx context.Context) error {
	h.L.Debug("cleanupCappedCollections: started", zap.Uint8("percentage", h.CappedCleanupPercentage))
	defer h.L.Debug("cleanupCappedCollections: finished")

	connInfo := conninfo.New()
	connInfo.BypassAuth = true
	ctx = conninfo.Ctx(ctx, connInfo)

	dbList, err := h.b.ListDatabases(ctx, nil)
	if err != nil {
		return lazyerrors.Error(err)
	}

	for _, dbInfo := range dbList.Databases {
		db, err := h.b.Database(dbInfo.Name)
		if err != nil {
			return lazyerrors.Error(err)
		}

		if db == nil {
			continue
		}

		var cList *backends.ListCollectionsResult

		if cList, err = db.ListCollections(ctx, nil); err != nil {
			return lazyerrors.Error(err)
		}

		for _, cInfo := range cList.Collections {
			if !cInfo.Capped() {
				continue
			}

			deleted, bytesFreed, err := h.cleanupCappedCollection(ctx, db, &cInfo, false)
			if err != nil {
				if backends.ErrorCodeIs(err, backends.ErrorCodeCollectionDoesNotExist) ||
					backends.ErrorCodeIs(err, backends.ErrorCodeDatabaseDoesNotExist) {
					continue
				}

				return lazyerrors.Error(err)
			}

			h.cleanupCappedCollectionsDocs.WithLabelValues(dbInfo.Name, cInfo.Name).Add(float64(deleted))
			h.L.Debug("cleanupCappedCollection: documents deleted",
				zap.String("db", dbInfo.Name), zap.String("collection", cInfo.Name),
				zap.Int32("deleted", deleted),
			)

			h.cleanupCappedCollectionsBytes.WithLabelValues(dbInfo.Name, cInfo.Name).Add(float64(bytesFreed))
			h.L.Debug("cleanupCappedCollection: bytes freed",
				zap.String("db", dbInfo.Name), zap.String("collection", cInfo.Name),
				zap.Int64("bytes", bytesFreed),
			)
		}

		if err != nil {
			return lazyerrors.Error(err)
		}
	}

	return nil
}

// cleanupCappedCollection drops a percent of documents from the given capped collection and compacts it.
// If the collection is not capped, it does nothing.
func (h *Handler) cleanupCappedCollection(ctx context.Context, db backends.Database, cInfo *backends.CollectionInfo, force bool) (int32, int64, error) {
	if !cInfo.Capped() {
		return 0, 0, nil
	}

	collection, err := db.Collection(cInfo.Name)
	if err != nil {
		return 0, 0, lazyerrors.Error(err)
	}

	statsBefore, err := collection.Stats(ctx, &backends.CollectionStatsParams{Refresh: true})
	if err != nil {
		return 0, 0, lazyerrors.Error(err)
	}

	if statsBefore.SizeCollection < cInfo.CappedSize && statsBefore.CountDocuments < cInfo.CappedDocuments {
		return 0, 0, nil
	}

	params := backends.QueryParams{
		Limit:         int64(float64(statsBefore.CountDocuments) * float64(h.CappedCleanupPercentage) / 100),
		OnlyRecordIDs: true,
	}

	res, err := collection.Query(ctx, &params)
	if err != nil {
		return 0, 0, lazyerrors.Error(err)
	}

	var recordIDs []int64

	for {
		_, doc, err := res.Iter.Next()
		if errors.Is(err, iterator.ErrIteratorDone) {
			break
		}

		if err != nil {
			return 0, 0, lazyerrors.Error(err)
		}

		recordIDs = append(recordIDs, doc.RecordID())
	}

	deleted, err := collection.DeleteAll(ctx, &backends.DeleteAllParams{RecordIDs: recordIDs})
	if err != nil {
		return 0, 0, lazyerrors.Error(err)
	}

	if _, err := collection.Compact(ctx, &backends.CompactParams{Full: force}); err != nil {
		return 0, 0, lazyerrors.Error(err)
	}

	statsAfter, err := collection.Stats(ctx, &backends.CollectionStatsParams{Refresh: true})
	if err != nil {
		return 0, 0, lazyerrors.Error(err)
	}

	bytesFreed := statsBefore.SizeTotal - statsAfter.SizeTotal

	// There's a possibility that the size of a collection might be greater at the
	// end of a compact operation if the collection is being actively written to at
	// the time of compaction.
	if bytesFreed < 0 {
		bytesFreed = 0
	}

	return deleted.Deleted, bytesFreed, nil
}
