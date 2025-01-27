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
	"io"
	"strings"
	"text/template"
)

// headerTemplate is used to generate header.
//
// "// Code generated" is intentionally not on the next line
// to prevent generate.go itself being marked as generated on GitHub.
var headerTemplate = template.Must(template.New("header").Parse(`// Code generated by "{{.Cmd}}"; DO NOT EDIT.

package {{.Package}}

import (
	"context"
	"log/slog"

	"github.com/FerretDB/wire/wirebson"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/FerretDB/FerretDB/v2/internal/mongoerrors"
)
`))

// headerData contains information needed for generating header.
type headerData struct {
	Cmd     string
	Package string
}

// funcTemplate is used to generate function.
var funcTemplate = template.Must(template.New("func").Funcs(funcMap).Parse(`
// {{.FuncName}} is a wrapper for
//
//	{{.Comment}}.
func {{.FuncName}}({{joinParameters .Params}}) ({{joinParameters .Returns}}) {
	ctx, span := otel.Tracer("").Start(ctx, "{{.SQLFuncName}}", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	defer span.End()

	row := conn.QueryRow({{join .QueryRowArgs ", "}})
	if err = row.Scan({{join .ScanArgs ", "}}); err != nil {
		err = mongoerrors.Make(ctx, err, "{{.SQLFuncName}}", l)
	}
	return
}
`))

// funcMap maps template to use go functions.
var funcMap = template.FuncMap{
	"join":           strings.Join,
	"joinParameters": joinParameters,
}

// joinParameters concatenates name and type with a space then
// produce a string by concatenating them with comma and space.
func joinParameters(params []convertedRoutineParam) string {
	s := make([]string, len(params))
	for i, p := range params {
		s[i] = fmt.Sprintf("%s %s", p.Name, p.Type)
	}

	return strings.Join(s, ", ")
}

// templateData contains information need for generating a function to run SQL query and scan the output.
type templateData struct {
	FuncName     string
	SQLFuncName  string
	Comment      string
	Params       []convertedRoutineParam
	Returns      []convertedRoutineParam
	QueryRowArgs []string
	ScanArgs     []string
}

// Generate uses schema and function definition to produce go function for
// querying DocumentDB API.
//
// The function is generated by using template and written to the writer.
func Generate(writer io.Writer, f *convertedRoutine) error {
	params := []convertedRoutineParam{
		{Name: "ctx", Type: "context.Context"},
		{Name: "conn", Type: "*pgx.Conn"},
		{Name: "l", Type: "*slog.Logger"},
	}
	params = append(params, f.GoParams...)

	scanArgs := make([]string, len(f.GoReturns))

	for i, r := range f.GoReturns {
		scanArgs[i] = fmt.Sprintf("&%s", r.Name)
	}

	returns := append(f.GoReturns, convertedRoutineParam{
		Name: "err",
		Type: "error",
	})

	q, sqlArgs := generateSQL(f)
	queryRowArgs := append([]string{"ctx", fmt.Sprintf(`"%s"`, q)}, sqlArgs...)

	data := templateData{
		FuncName:     pascalCase(f.Name),
		SQLFuncName:  f.SQLFuncName,
		Comment:      f.Comment,
		Params:       params,
		Returns:      returns,
		QueryRowArgs: queryRowArgs,
		ScanArgs:     scanArgs,
	}

	return funcTemplate.Execute(writer, &data)
}

// generateSQL builds SQL query and arguments for the given function definition.
func generateSQL(f *convertedRoutine) (string, []string) {
	args := make([]string, len(f.GoParams))
	for i, p := range append(f.GoParams) {
		args[i] = p.Name
	}

	if f.IsProcedure {
		q := fmt.Sprintf("CALL %s(%s)", f.SQLFuncName, f.QueryArgs)
		return q, args
	}

	q := fmt.Sprintf(
		"SELECT %s FROM %s(%s)",
		f.QueryReturns,
		f.SQLFuncName,
		f.QueryArgs,
	)

	return q, args
}
