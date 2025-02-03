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

// Package documentdb provides DocumentDB extension integration.
package documentdb

import "context"

// The only schema we should be using is documentdb_api.
// See also:
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/1221
//
// We use documentdb_api_catalog schema for `listDatabases` and `explain` commands.
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/26
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/143
//
// We use documentdb_api_internal schema for indexes and authentication:
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/1147
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/1221
//
//go:generate go run ./genwrap -debug -schemas=documentdb_api,documentdb_api_catalog,documentdb_api_internal,documentdb_core

// todoCtx should be used instead of [context.TODO] in this package.
// See https://github.com/jackc/pgx/issues/1726#issuecomment-1711612138.
var todoCtx = context.TODO()
