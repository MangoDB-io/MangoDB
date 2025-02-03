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

// Package main contains linter for switches.
package main

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// bsonOrder is the preferred order of case elements in switch statements.
var bsonOrder = map[string]int{
	"AnyDocument":          1,
	"Document":             1,
	"RawDocument":          1,
	"TypeEmbeddedDocument": 1,

	"AnyArray":  2,
	"Array":     2,
	"RawArray":  2,
	"TypeArray": 2,

	"float64":    3,
	"TypeDouble": 3,

	"string":     4,
	"TypeString": 4,

	"Binary":     5,
	"TypeBinary": 5,

	"UndefinedType": 6,
	"TypeUndefined": 6,

	"ObjectID":     7,
	"TypeObjectID": 7,

	"bool":        8,
	"TypeBoolean": 8,

	"Time":         9,
	"TypeDateTime": 9,

	"NullType": 10,
	"TypeNull": 10,

	"Regex":     11,
	"TypeRegex": 11,

	"TypeDBPointer": 12,

	"TypeJavaScript": 13,

	"TypeSymbol": 14,

	"TypeCodeWithScope": 15,

	"int32":     16,
	"TypeInt32": 16,

	"Timestamp":     17,
	"TypeTimestamp": 17,

	"int64":     18,
	"TypeInt64": 18,

	"TypeDecimal128": 19,

	"TypeMinKey": 20,

	"TypeMaxKey": 21,

	"CString": 22,
}

var analyzer = &analysis.Analyzer{
	Name: "checkswitch",
	Doc:  "check the preferred order of types in the switch",
	Run:  run,
}

func main() {
	singlechecker.Main(analyzer)
}

// run is the function to be called by the driver to execute analysis on a single package.
//
// It analyzes the presence of types in 'case' in ascending order of indexes defined in 'bsonOrder'.
func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.TypeSwitchStmt:
				checkOrder(n.Body.List, pass, n.Pos())
			case *ast.SwitchStmt:
				checkOrder(n.Body.List, pass, n.Pos())
			}

			return true
		})
	}

	return nil, nil
}

// checkOrder checks the order of the case elements in switch statements.
func checkOrder(list []ast.Stmt, pass *analysis.Pass, pos token.Pos) {
	var order int
	var name string

	// outer loop checks the order of case clauses
	// case "int32":
	// case "int64":
	for _, stmt := range list {
		caseOrder, caseName := order, name

		// inner loop checks the order within a case statement
		// case "int32", "int64":
		for i, caseElem := range stmt.(*ast.CaseClause).List {
			var elemName string

			switch expr := caseElem.(type) {
			case *ast.StarExpr:
				switch starExpr := expr.X.(type) {
				case *ast.SelectorExpr:
					elemName = starExpr.Sel.Name
				case *ast.Ident:
					elemName = starExpr.Name
				default:
					// not `types` or `tags`
				}

			case *ast.SelectorExpr:
				elemName = expr.Sel.Name

			case *ast.Ident:
				elemName = expr.Name

			default:
				// not `types` or `tags`
			}

			elemOrder, ok := bsonOrder[elemName]
			if ok && (elemOrder < caseOrder) {
				pass.Reportf(pos, "%s should go before %s in the switch", elemName, caseName)
			}

			caseOrder, caseName = elemOrder, elemName

			if i == 0 {
				// i.e. "int64" is larger than "int32" but below is allowed
				// case "float64", "int64":
				// case "int32":
				order, name = elemOrder, elemName
			}
		}
	}
}
