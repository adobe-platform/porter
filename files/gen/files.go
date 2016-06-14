package main

//go:generate go run files.go

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

func main() {
	// printAST()
	// return

	workDir := os.Getenv("PWD") + "/files"

	fmt.Println("generating constants strings for files")

	constsAST := make([]ast.Decl, 0)

	infos, err := ioutil.ReadDir(workDir)
	if err != nil {
		panic(err)
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		switch path.Ext(info.Name()) {
		case ".go", ".md":
			continue
		default:
			// do nothing
		}

		nameParts := regexp.MustCompile(`\.|_|-`).Split(info.Name(), -1)
		constName := ""
		for _, namePart := range nameParts {
			namePart = strings.ToUpper(namePart[:1]) + namePart[1:]
			constName = constName + namePart
		}
		if constName == "" {
			panic(info.Name() + " couldn't be constantized")
		}

		constValueBytes, err := ioutil.ReadFile(workDir + "/" + info.Name())
		if err != nil {
			panic(err)
		}

		constValue := string(constValueBytes)
		constValue = fmt.Sprintf("`%s`", constValue)

		constAST := &ast.GenDecl{
			Tok: token.CONST,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{
						{
							Name: constName,
							Obj:  ast.NewObj(ast.Con, constName),
						},
					},
					Values: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: constValue,
						},
					},
				},
			},
		}

		constsAST = append(constsAST, constAST)
	}

	astRoot := &ast.File{
		Name: &ast.Ident{
			Name: "files",
		},
		Decls: constsAST,
	}

	genFile, err := os.Create(workDir + "/files_generated.go")
	if err != nil {
		panic(err)
	}
	defer genFile.Close()
	fmt.Println("writing generated consts to", genFile.Name())

	fset := token.NewFileSet()

	printer.Fprint(genFile, fset, astRoot)
	// printer.Fprint(os.Stdout, fset, astRoot)
}

func printAST() {
	src := "package main\n\nconst foo = `hi\nbye`"

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "foo.go", src, 0)
	if err != nil {
		panic(err)
	}

	// Print the AST.
	ast.Print(fset, f)
}
