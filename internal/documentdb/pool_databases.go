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

package documentdb

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"

	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
)

// DatabaseInfo represents an information about a single database.
type DatabaseInfo struct {
	// TODO https://github.com/FerretDB/FerretDB/issues/4722
	// TODO https://github.com/microsoft/documentdb/issues/45
}

// ListDatabases returns a list of existing databases and their information.
//
// TODO https://github.com/FerretDB/FerretDB/issues/4722
// TODO https://github.com/microsoft/documentdb/issues/45
func (p *Pool) ListDatabases(ctx context.Context) (map[string]DatabaseInfo, error) {
	ctx, span := otel.Tracer("").Start(ctx, "pool.ListDatabases")
	defer span.End()

	rows, err := p.p.Query(ctx, "SELECT DISTINCT database_name FROM documentdb_api_catalog.collections")
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	res := map[string]DatabaseInfo{}

	var databaseName string
	scans := []any{&databaseName}

	_, err = pgx.ForEachRow(rows, scans, func() error {
		res[databaseName] = DatabaseInfo{}
		return nil
	})
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return res, nil
}
