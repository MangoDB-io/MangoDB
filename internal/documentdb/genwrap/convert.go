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
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"unicode"
)

// Convert takes rows containing parameters of routines.
// It returns a map of schemas and routines belonging to each schema by
// converting rows to Go formatted names and types.
//
// For an anonymous SQL parameter, it assigns a unique name.
// It also produces SQL query placeholders and return parameters in strings.
func Convert(routineParams map[string][]map[string]any, l *slog.Logger) map[string]map[string]templateData {
	c := &converter{
		l: l,
	}

	schemas := map[string]map[string]templateData{}

	for _, specificName := range slices.Sorted(maps.Keys(routineParams)) {
		params := routineParams[specificName]

		var goParams, goReturns, sqlParams, sqlReturns, comment, queryRowArgs, scanArgs, paramNames []string
		var placeholderCounter int

		for _, row := range params {
			name := "anonymous"

			if row["parameter_name"] != nil {
				name = row["parameter_name"].(string)
			}

			if row["parameter_mode"] == "IN" {
				name = c.uniqueName(paramNames, name)
			}

			paramNames = append(paramNames, name)

			if row["parameter_name"] == "p_transaction_id" {
				// skip p_transaction_id, transaction is not supported yet
				// TODO https://github.com/FerretDB/FerretDB/issues/8
				continue
			}

			if row["parameter_name"] == nil {
				// skip a row if the row does not contain a parameter such as BinaryExtendedVersion()
				continue
			}

			paramComment := name + " " + c.pgParameterType(row)
			if row["parameter_mode"] != "IN" {
				paramComment = row["parameter_mode"].(string) + " " + paramComment
			}

			if row["parameter_default"] != nil {
				d, _, _ := strings.Cut(row["parameter_default"].(string), "::")
				paramComment += " DEFAULT " + d
			}

			comment = append(comment, paramComment)

			dataType := c.pgParameterType(row)

			if row["parameter_mode"] == "IN" || row["parameter_mode"] == "INOUT" {
				placeholder := fmt.Sprintf("$%d", placeholderCounter+1)
				placeholderCounter++

				goName := c.parameterName(name)
				sqlParams = append(sqlParams, c.parameterCast(placeholder, dataType))
				goParams = append(goParams, fmt.Sprintf("%s %s", goName, c.parameterType(dataType)))
				queryRowArgs = append(queryRowArgs, goName)
			}

			if row["parameter_mode"] == "OUT" || row["parameter_mode"] == "INOUT" {
				goName := "out" + c.pascalCase(c.parameterName(name))
				sqlReturns = append(sqlReturns, c.parameterCast(name, dataType))
				goReturns = append(goReturns, fmt.Sprintf("%s %s", goName, c.parameterType(dataType)))
				scanArgs = append(scanArgs, fmt.Sprintf("&%s", goName))
			}
		}

		routineName := params[0]["routine_name"].(string)

		if len(goReturns) == 0 && params[0]["routine_type"] == "FUNCTION" {
			// function such as binary_extended_version() does not have
			// parameter data type, but it has routine data type for the return variable.
			goName := "out" + c.pascalCase(c.parameterName(routineName))
			dataType := c.routineDataType(params[0])
			sqlReturns = append(sqlReturns, c.parameterCast(routineName, dataType))
			goReturns = append(goReturns, fmt.Sprintf("%s %s", goName, c.parameterType(dataType)))
			scanArgs = append(scanArgs, fmt.Sprintf("&%s", goName))
			comment = append(comment, fmt.Sprintf("OUT %s %s", routineName, dataType))
		}

		schema := params[0]["specific_schema"].(string)

		if _, ok := schemas[schema]; !ok {
			schemas[schema] = map[string]templateData{}
		}

		// unique name is used to handle function overloading
		uniqueFunctionName := c.uniqueName(slices.Collect(maps.Keys(schemas[schema])), routineName)

		r := templateData{
			FuncName:     c.pascalCase(uniqueFunctionName),
			SQLFuncName:  fmt.Sprintf("%s.%s", schema, routineName),
			Comment:      fmt.Sprintf("%s.%s(%s)", schema, routineName, strings.Join(comment, ", ")),
			IsProcedure:  params[0]["routine_type"] == "PROCEDURE",
			SQLArgs:      strings.Join(sqlParams, ", "),
			SQLReturns:   strings.Join(sqlReturns, ", "),
			Params:       strings.Join(goParams, ", "),
			Returns:      strings.Join(goReturns, ", "),
			QueryRowArgs: strings.Join(queryRowArgs, ", "),
			ScanArgs:     strings.Join(scanArgs, ", "),
		}

		c.handleFunctionOverloading(&r)

		schemas[schema][uniqueFunctionName] = r
	}

	return schemas
}

// converter is used to group methods used by [Convert].
type converter struct {
	l *slog.Logger
}

// camelCase converts snake_case to to camelCase.
func (c *converter) camelCase(s string) string {
	var res []rune

	var u bool
	for _, ch := range s {
		if ch == '_' {
			u = true
			continue
		}

		if u {
			ch = unicode.ToUpper(ch)
			u = false
		}

		res = append(res, ch)
	}

	return string(res)
}

// pascalCase converts snake_case to PascalCase.
func (c *converter) pascalCase(s string) string {
	res := []rune(c.camelCase(s))
	res[0] = unicode.ToUpper(res[0])
	return string(res)
}

// parameterName converts PostgreSQL/DocumentDB routine parameter name
// to Go function/method parameter name.
func (c *converter) parameterName(name string) string {
	name = strings.TrimPrefix(name, "p_")

	switch name {
	case "dbname":
		return "database"

	case "getmorespec":
		return "getMoreSpec"
	case "letvariablespec":
		return "letVariableSpec"

	case "cursorid":
		return "cursorID"
	case "cursorpage":
		return "cursorPage"
	case "object_id":
		return "objectID"
	case "persistconnection":
		return "persistConnection"
	case "retval":
		return "retValue"
	}

	if name != "spec" && strings.HasSuffix(name, "spec") {
		name = strings.TrimSuffix(name, "spec") + "_spec"
	}

	return c.camelCase(name)
}

// parameterType converts PostgreSQL/DocumentDB routine parameter type
// to Go function/method parameter type.
func (c *converter) parameterType(pgType string) string {
	switch pgType {
	case "text":
		return "string"
	case "boolean":
		return "bool"
	case "bigint":
		return "int64"
	case "double precision":
		return "float64"
	case "uuid":
		return "[]byte"

	case "documentdb_core.bson":
		return "wirebson.RawDocument"
	case "documentdb_core.bsonsequence":
		return "[]byte"
	}

	c.l.Debug("Unhandled type", slog.String("type", pgType))
	return "struct{}"
}

// parameterCast adds a type cast (::type) to a parameter if needed.
func (c *converter) parameterCast(name string, typ string) string {
	switch typ {
	case "documentdb_core.bson", "documentdb_core.bsonsequence":
		return name + "::bytea"
	default:
		return name
	}
}

// funcName converts PostgreSQL/DocumentDB routine name
// to Go function/method name.
func (c *converter) funcName(name string) string {
	return c.pascalCase(c.parameterName(name))
}

// pgParameterType gets PostgreSQL/DocumentDB routine parameter type.
func (c *converter) pgParameterType(row map[string]any) string {
	if row["data_type"] == "USER-DEFINED" {
		return row["udt_schema"].(string) + "." + row["udt_name"].(string)
	}

	return row["data_type"].(string)
}

// pgParameterType gets PostgreSQL/DocumentDB routine result parameter type.
func (c *converter) pgResultType(row map[string]any) string {
	if row["routine_data_type"] == "USER-DEFINED" {
		return row["routine_udt_schema"].(string) + "." + row["routine_udt_name"].(string)
	}

	return row["routine_data_type"].(string)
}

func (c *converter) routine(routine []map[string]any) templateData {
	f := routine[0]
	res := templateData{
		FuncName:     c.funcName(f["routine_name"].(string)),
		SQLFuncName:  fmt.Sprintf("%s.%s", f["specific_schema"], f["routine_name"]),
		Comment:      "[Comment]",
		Params:       "[Params]",
		Returns:      "[Returns]",
		SQLArgs:      "[SQLArgs]",
		SQLReturns:   "[SQLReturns]",
		IsProcedure:  f["routine_type"] == "PROCEDURE",
		QueryRowArgs: "[QueryRowArgs]",
		ScanArgs:     "[ScanArgs]",
	}

	return res
}
