/*
 * (c) 2016-2018 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	typeToTypes     map[string][]string
	typeToShortHelp map[string]string
	typeToLongHelp  map[string]string
	typeToMethods   map[string][]string
	methodToComment map[string]string
)

func init() {
	typeToTypes = make(map[string][]string)
	typeToShortHelp = make(map[string]string)
	typeToLongHelp = make(map[string]string)
	typeToMethods = make(map[string][]string)
	methodToComment = make(map[string]string)
}

// Generate CLI command help from comments found on methods in the cfn_template
// package.
//
// This generates /commands/commands_generated.go with the following functions
// that commands.go references:
//   func GetCfnHelpCommands
func main() {

	pwd := os.Getenv("PWD")

	// PWD is where `go generate` is called from which is the root of the source
	// tree
	rootPath := pwd + "/cfn_template"

	fmt.Println("generating CLI commands for struct/methods in the cfn_template package")

	fset := token.NewFileSet()
	// Create the AST by parsing src.
	ps, err := parser.ParseDir(fset, rootPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	for name, pack := range ps {
		fmt.Println("processing package " + name)

		// Inspect the AST and print all identifiers and literals.
		ast.Inspect(pack, func(n ast.Node) (walk bool) {
			walk = true

			switch t := n.(type) {
			case *ast.GenDecl:
				if t.Tok != token.TYPE {
					return
				}

				for _, spec := range t.Specs {

					if tSpec, ok := spec.(*ast.TypeSpec); ok {
						typeName := fmt.Sprintf("%s", tSpec.Name)

						if tSpec.Doc != nil {
							comment := tSpec.Doc.Text()
							if strings.TrimSpace(comment) == "@ignore" {
								continue
							}

							typeToLongHelp[typeName] = comment
						}

						if structT, ok := tSpec.Type.(*ast.StructType); ok && structT.Fields != nil {

							for _, field := range structT.Fields.List {
								fieldName := field.Names[0].Name
								fieldType := fmt.Sprintf("%s", field.Type)
								if fieldName != fieldType {
									fmt.Println("WARNING: field name should match field type on struct", typeName)
									continue
								}

								if field.Doc != nil {
									typeToShortHelp[fieldName] = field.Doc.Text()
								}

								types, exists := typeToTypes[typeName]
								if !exists {
									types = []string{}
								}

								types = append(types, fieldType)
								typeToTypes[typeName] = types
							}
						}
					}
				}

			case *ast.FuncDecl:

				if t.Recv == nil { // this is a function, not a method
					return
				}

				if len(t.Recv.List) != 1 {
					return
				}

				if t.Doc == nil {
					fmt.Println("WARNING: missing help documentation for", t.Name)
					return
				}

				methodName := fmt.Sprintf("%s", t.Name)
				typeName := fmt.Sprintf("%s", t.Recv.List[0].Type)

				methods, exists := typeToMethods[typeName]
				if !exists {
					methods = []string{}
				}

				methods = append(methods, methodName)
				typeToMethods[typeName] = methods

				methodToComment[methodName] = t.Doc.Text()
			}
			return
		})
	}

	commandsAST := make([]ast.Expr, 0)

	for _, typeName := range typeToTypes["Context"] {
		shortHelp, _ := typeToShortHelp[typeName]
		longHelp, _ := typeToLongHelp[typeName]

		cmdAST := newCLICommand(typeName, shortHelp, longHelp, make([]ast.Expr, 0))
		walkContextTree(typeName, cmdAST)

		commandsAST = append(commandsAST, cmdAST)
	}

	astRoot := &ast.File{
		Name: &ast.Ident{
			Name: "commands",
		},
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Value: strconv.Quote("github.com/phylake/go-cli"),
						},
					},
				},
			},

			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Value: strconv.Quote("github.com/phylake/go-cli/cmd"),
						},
					},
				},
			},

			&ast.FuncDecl{
				Name: ast.NewIdent("GetCfnHelpCommands"),
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{
								Type: &ast.ArrayType{
									Elt: &ast.SelectorExpr{
										X:   ast.NewIdent("cli"),
										Sel: ast.NewIdent("Command"),
									},
								},
							},
						},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: []ast.Expr{
								&ast.CompositeLit{
									Type: &ast.ArrayType{
										Elt: &ast.SelectorExpr{
											X:   ast.NewIdent("cli"),
											Sel: ast.NewIdent("Command"),
										},
									},
									Elts: commandsAST,
								},
							},
						},
					},
				},
			},
		},
	}

	fset = token.NewFileSet()
	filepath.Walk(os.Getenv("GOPATH")+"/src/github.com/phylake/go-cli", func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) != ".go" {
			return nil
		}

		fset.AddFile(path, fset.Base(), int(info.Size()))
		return nil
	})

	genFile, err := os.Create(pwd + "/commands/commands_generated.go")
	if err != nil {
		panic(err)
	}
	defer genFile.Close()
	fmt.Println("writing generated commands to", genFile.Name())

	printer.Fprint(genFile, fset, astRoot)
	// printer.Fprint(os.Stdout, fset, astRoot)
}

func walkContextTree(typeName string, unary *ast.UnaryExpr) {

	compositeLit1, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		panic("unary.X.(*ast.CompositeLit)")
	}

	kve, ok := compositeLit1.Elts[3].(*ast.KeyValueExpr)
	if !ok {
		panic("compositeLit1.Elts[3].(*ast.KeyValueExpr)")
	}

	compositeLit2, ok := kve.Value.(*ast.CompositeLit)
	if !ok {
		panic("kve.Value.(*ast.CompositeLit)")
	}

	if ts, exists := typeToTypes[typeName]; exists {
		for _, t := range ts {
			shortHelp, _ := typeToShortHelp[t]
			longHelp, _ := typeToLongHelp[t]

			if shortHelp == "" {
				shortHelp = t + " help"
			}

			cmd := newCLICommand(t, shortHelp, longHelp, make([]ast.Expr, 0))
			compositeLit2.Elts = append(compositeLit2.Elts, cmd)
			walkContextTree(t, cmd)
		}
	}

	if ms, exists := typeToMethods[typeName]; exists {
		sort.Strings(ms)
		for _, method := range ms {
			if comment, exists := methodToComment[method]; exists {
				cmd := newCLICommand(method, method+" generator", comment, nil)
				compositeLit2.Elts = append(compositeLit2.Elts, cmd)
			}
		}
	}

}

func newCLICommand(nameStr, shortHelpStr, longHelpStr string, subs []ast.Expr) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		Op: token.AND,
		X: &ast.CompositeLit{
			Type: &ast.SelectorExpr{
				X:   ast.NewIdent("cmd"),
				Sel: ast.NewIdent("Default"),
			},
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key: ast.NewIdent("NameStr"),
					Value: &ast.BasicLit{
						Value: strconv.Quote(nameStr),
					},
				},
				&ast.KeyValueExpr{
					Key: ast.NewIdent("ShortHelpStr"),
					Value: &ast.BasicLit{
						Value: strconv.Quote(shortHelpStr),
					},
				},
				&ast.KeyValueExpr{
					Key: ast.NewIdent("LongHelpStr"),
					Value: &ast.BasicLit{
						Value: strconv.Quote(longHelpStr),
					},
				},
				&ast.KeyValueExpr{
					Key: ast.NewIdent("SubCommandList"),
					Value: &ast.CompositeLit{
						Type: &ast.ArrayType{
							Elt: &ast.SelectorExpr{
								X:   ast.NewIdent("cli"),
								Sel: ast.NewIdent("Command"),
							},
						},
						Elts: subs,
					},
				},
			},
		},
	}
}
