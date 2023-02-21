package main

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rComment = regexp.MustCompile(`^//.*?@(?i:go-enum-no-prefix?)\s*(.*)$`)
)

type ReplaceArea interface {
	GetStart() int
	GetEnd() int
	Replace(contents []byte, offset int) ([]byte, int)
}

type EnumValueIdentArea struct {
	Start       int
	End         int
	CurrentName string
	NewName     string
}

func (r *EnumValueIdentArea) GetStart() int {
	return r.Start
}

func (r *EnumValueIdentArea) GetEnd() int {
	return r.End
}

func (r *EnumValueIdentArea) Replace(contents []byte, offset int) ([]byte, int) {
	var injected []byte
	injected = append(injected, contents[:r.Start-1-offset]...)
	injected = append(injected, r.NewName...)
	injected = append(injected, contents[r.End-1-offset:]...)

	diff := len(r.CurrentName) - len(r.NewName)
	offset += diff

	return injected, offset
}

type CommentArea struct {
	Start    int
	End      int
	EnumType string
}

func (r *CommentArea) GetStart() int {
	return r.Start
}

func (r *CommentArea) GetEnd() int {
	return r.End
}

func (r *CommentArea) Replace(contents []byte, offset int) ([]byte, int) {
	var injected []byte

	start := r.Start - 1 - offset
	end := r.End - 1 - offset

	prevChar := string(contents[r.Start-2])
	if prevChar == "\n" {
		start--
	}

	injected = append(injected, contents[:start]...)
	injected = append(injected, contents[end:]...)

	offset += end - start

	return injected, offset
}

type SourceMap = map[string]SourceMapFile

type SourceMapFile struct {
	Offset int
	Decls  []ast.Decl
}

func simpleImporter(imports map[string]*ast.Object, path string) (*ast.Object, error) {
	pkg := imports[path]
	if pkg == nil {
		// note that strings.LastIndex returns -1 if there is no "/"
		pkg = ast.NewObj(ast.Pkg, path[strings.LastIndex(path, "/")+1:])
		pkg.Data = ast.NewScope(nil) // required by ast.NewPackage for dot-import
		imports[path] = pkg
	}

	return pkg, nil
}

func loadSources(inputPath string, sourceMap SourceMap) error {
	logf("parsing file %q for comments", inputPath)

	fset := token.NewFileSet()

	buildPkg, err := build.ImportDir(filepath.Dir(inputPath), build.ImportComment)
	if err != nil {
		return err
	}

	files := make(map[string]*ast.File)

	for _, file := range append(buildPkg.GoFiles, buildPkg.CgoFiles...) {
		fname := filepath.Join(buildPkg.Dir, file)
		// src, err := os.ReadFile(fname)
		// if err != nil {
		//	return err
		//}
		f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		files[fname] = f
	}

	astPkg, _ := ast.NewPackage(fset, files, simpleImporter, nil)

	for path, file := range astPkg.Files {
		fset2 := token.NewFileSet()

		decl2, err := parser.ParseFile(fset2, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		sourceMap[path] = SourceMapFile{
			Offset: int(file.Comments[0].Pos()) - int(decl2.Comments[0].Pos()),
			Decls:  file.Decls,
		}
	}

	return nil
}

func findEnumsToReplace(source SourceMapFile, enumsToReplace map[string]any) []ReplaceArea {
	var comments []ReplaceArea

	for _, decl := range source.Decls {
		// check if is generic declaration
		genDecl, isGen := decl.(*ast.GenDecl)

		if !isGen {
			continue
		}

		var typeSpec *ast.TypeSpec

		for _, spec := range genDecl.Specs {
			if ts, tsOK := spec.(*ast.TypeSpec); tsOK {
				typeSpec = ts

				break
			}
		}

		if typeSpec == nil {
			continue
		}

		ident, ok := typeSpec.Type.(*ast.Ident)
		if !ok {
			continue
		}

		if ident.Name != "int32" {
			continue
		}

		if genDecl.Doc == nil {
			continue
		}

		for _, comment := range genDecl.Doc.List {
			isTagged := isTaggedEnum(comment.Text)
			if isTagged {
				enumsToReplace[typeSpec.Name.Name] = true

				comments = append(comments, &CommentArea{
					Start:    int(comment.Pos()) - source.Offset,
					End:      int(comment.End()) - source.Offset,
					EnumType: typeSpec.Name.Name,
				})
			}
		}
	}

	return comments
}

func findIdents(source SourceMapFile, enumsToReplace map[string]any) []ReplaceArea {
	var idents []ReplaceArea

	for _, decl := range source.Decls {
		// check if is generic declaration
		genDecl, isGen := decl.(*ast.GenDecl)
		funcDecl, isFunc := decl.(*ast.FuncDecl)

		if !isGen && !isFunc {
			continue
		}

		if isGen {
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

					if _, ok = enumsToReplace[valTypeIdentSpec.Name]; !ok {
						continue
					}

					for _, valName := range valSpec.Names {
						newName := valName.Name[len(valTypeIdentSpec.Name)+1:]

						idents = append(idents, &EnumValueIdentArea{
							Start:       int(valName.Pos()) - source.Offset,
							End:         int(valName.End()) - source.Offset,
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
					if !ok || resIdentDecl.Obj == nil {
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

					if _, ok = enumsToReplace[resValueTypeDecl.Name]; !ok {
						continue
					}

					idents = append(idents, &EnumValueIdentArea{
						Start:       int(resDecl.Pos()) - source.Offset,
						End:         int(resDecl.End()) - source.Offset,
						CurrentName: resIdentDecl.Name,
						NewName:     resIdentDecl.Name[len(resValueTypeDecl.Name)+1:],
					})
				}
			}
		}
	}

	return idents
}

func writeFile(inputPath string, replaces []ReplaceArea) error {
	fileHandle, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	contents, err := io.ReadAll(fileHandle)
	if err != nil {
		return err
	}

	if err = fileHandle.Close(); err != nil {
		return err
	}

	offset := 0

	sort.Slice(replaces, func(i, j int) bool {
		return replaces[i].GetStart() < replaces[j].GetStart()
	})

	for _, replaceArea := range replaces {
		contents, offset = replaceArea.Replace(contents, offset)
	}

	if err = os.WriteFile(inputPath, contents, 0o644); err != nil {
		return err
	}

	if len(replaces) > 0 {
		logf("file %q has %d replaces", inputPath, replaces)
	}

	return nil
}
