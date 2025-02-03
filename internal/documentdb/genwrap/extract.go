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

package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
)

// Extract returns rows of routines and parameters for given schemas.
// Returned routines are grouped by full specific names; rows are sorted by parameter position.
func Extract(ctx context.Context, uri string, schemas []string) (map[string][]map[string]any, error) {
	conn, err := pgx.Connect(ctx, uri)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	defer conn.Close(ctx)

	slices.Sort(schemas)
	schemas = slices.Compact(schemas)

	placeholders := make([]string, len(schemas))
	args := make([]any, len(schemas))

	for i, schema := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = schema
	}

	// https://www.postgresql.org/docs/current/infoschema-routines.html
	// https://www.postgresql.org/docs/current/infoschema-parameters.html
	q := fmt.Sprintf(`
	SELECT
		specific_schema,
		specific_name,
		routine_name,
		routine_type,
		r.data_type AS routine_data_type,
		r.type_udt_schema AS routine_udt_schema,
		r.type_udt_name AS routine_udt_name,
		parameter_name,
		parameter_mode,
		parameter_default,
		p.data_type,
		p.udt_schema,
		p.udt_name
	FROM information_schema.routines AS r
		LEFT JOIN information_schema.parameters AS p USING (specific_schema, specific_name)
	WHERE specific_schema IN (%s)
	ORDER BY ordinal_position
	`,
		strings.Join(placeholders, ", "),
	)

	rows, err := conn.Query(ctx, q, args...)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	mappedRows, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	res := make(map[string][]map[string]any)

	for _, row := range mappedRows {
		fullName := row["specific_schema"].(string) + "." + row["specific_name"].(string)

		routine := res[fullName]
		if routine == nil {
			routine = []map[string]any{}
		}

		routine = append(routine, row)
		res[fullName] = routine
	}

	return res, nil
}
