package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"regexp"
)

var (
	rComment = regexp.MustCompile(`^//.*?@(?i:go-enum-no-prefix?)\s*(.*)$`)
)

type EnumValueIdentArea struct {
	Start       int
	End         int
	CurrentName string
	NewName     string
}

type CommentArea struct {
	Start int
	End   int
}

func parseFile(inputPath string, src interface{}) (idents []EnumValueIdentArea, comments []CommentArea, err error) {
	logf("parsing file %q for inject tag comments", inputPath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, src, parser.ParseComments)
	if err != nil {
		return
	}

	enumTypesToReplace := map[string]any{}

	for _, decl := range f.Decls {
		// check if is generic declaration
		genDecl, isGen := decl.(*ast.GenDecl)
		funcDecl, isFunc := decl.(*ast.FuncDecl)

		if !isGen && !isFunc {
			continue
		}

		if isGen {
			var typeSpec *ast.TypeSpec
			for _, spec := range genDecl.Specs {
				if ts, tsOK := spec.(*ast.TypeSpec); tsOK {
					typeSpec = ts
					break
				}
			}

			// skip if can't get type spec
			if typeSpec != nil {
				ident, ok := typeSpec.Type.(*ast.Ident)
				if !ok {
					continue
				}

				if ident.Name != "int32" {
					continue
				}

				for _, comment := range genDecl.Doc.List {
					isTagged := isTaggedEnum(comment.Text)
					if isTagged {
						enumTypesToReplace[typeSpec.Name.Name] = true
						comments = append(comments, CommentArea{
							Start: int(comment.Pos()),
							End:   int(comment.End()),
						})
					}
				}

				continue
			}

			if genDecl.Tok == token.CONST {
				for _, val := range genDecl.Specs {
					valSpec, ok := val.(*ast.ValueSpec)
					if !ok {
						continue
					}

					valTypeIdentSpec, ok := valSpec.Type.(*ast.Ident)
					if !ok {
						continue
					}

					if _, ok = enumTypesToReplace[valTypeIdentSpec.Name]; !ok {
						continue
					}

					for _, valName := range valSpec.Names {
						newName := valName.Name[len(valTypeIdentSpec.Name)+1:]
						log.Printf("%s will be replaced with %s\n", valName.Name, newName)
						idents = append(idents, EnumValueIdentArea{
							Start:       int(valName.Pos()),
							End:         int(valName.End()),
							CurrentName: valName.Name,
							NewName:     newName,
						})
					}

				}
			}
		}

		if isFunc {
			for _, fDecl := range funcDecl.Body.List {
				retDecl, ok := fDecl.(*ast.ReturnStmt)
				if !ok {
					continue
				}

				for _, resDecl := range retDecl.Results {
					resIdentDecl, ok := resDecl.(*ast.Ident)
					if !ok {
						continue
					}

					if resIdentDecl.Obj.Kind != ast.Con {
						continue
					}

					resValueDecl, ok := resIdentDecl.Obj.Decl.(*ast.ValueSpec)
					if !ok {
						continue
					}

					resValueTypeDecl, ok := resValueDecl.Type.(*ast.Ident)
					if !ok {
						continue
					}

					if _, ok = enumTypesToReplace[resValueTypeDecl.Name]; !ok {
						continue
					}

					idents = append(idents, EnumValueIdentArea{
						Start:       int(resDecl.Pos()),
						End:         int(resDecl.End()),
						CurrentName: resIdentDecl.Name,
						NewName:     resIdentDecl.Name[len(resValueTypeDecl.Name)+1:],
					})
				}
			}
		}

	}
	logf("parsed file %q, number of fields to inject custom tags: %d", inputPath, len(idents))
	return
}

func writeFile(inputPath string, idents []EnumValueIdentArea, comments []CommentArea) (err error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return
	}

	contents, err := io.ReadAll(f)
	if err != nil {
		return
	}

	if err = f.Close(); err != nil {
		return
	}

	// Inject custom tags from tail of file first to preserve order
	for i := range idents {
		identArea := idents[len(idents)-i-1]
		logf("inject custom tag %q to expression %q", identArea.CurrentName, string(contents[identArea.Start-1:identArea.End-1]))
		contents = replaceEnumValueIdent(contents, identArea)
	}

	// Remove tags
	for i := range comments {
		commentArea := comments[len(comments)-i-1]
		contents = removeEnumComment(contents, commentArea)
	}
	if err = os.WriteFile(inputPath, contents, 0o644); err != nil {
		return
	}

	if len(idents) > 0 {
		logf("file %q is injected with custom tags", inputPath)
	}
	return
}
