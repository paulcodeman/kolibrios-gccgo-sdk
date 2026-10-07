// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package go2go

import (
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

// liftLocalTypes makes ordinary function-local types available to package-level
// specializations. Type checking has already enforced the original scopes. Keep
// that information: rechecking the lifted trees would change embedded field
// names to compiler identifiers and lose their original Go ownership.
// Generic function bodies require per-instantiation local type identities and
// are deliberately left for the instantiation pass.
func (imp *Importer) liftLocalTypes(fset *token.FileSet, pkg *types.Package, files []*ast.File) {
	baseCounts := make(map[string]int)
	for _, file := range files {
		baseCounts[filepath.Base(fset.PositionFor(file.Package, false).Filename)]++
	}
	type localDecl struct {
		file *ast.File
		spec *ast.TypeSpec
		obj  *types.TypeName
	}
	var decls []localDecl
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok {
				if obj := imp.info.Defs[fn.Name]; obj != nil {
					if sig, ok := obj.Type().(*types.Signature); ok &&
						(sig.TypeParams().Len() != 0 || sig.RecvTypeParams().Len() != 0) {
						return false
					}
				}
			}
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			obj, ok := imp.info.Defs[spec.Name].(*types.TypeName)
			if !ok || obj.Parent() == nil || obj.Parent() == pkg.Scope() {
				return true
			}
			pos := fset.PositionFor(obj.Pos(), false)
			filename := filepath.Base(pos.Filename)
			if baseCounts[filename] > 1 {
				filename = pos.Filename
			}
			identity := fmt.Sprintf("%s.local.%s:%d", pkg.Path(), filename, pos.Offset)
			parts := []string{identity, pkg.Path(), pkg.Name(), obj.Name()}
			for i, part := range parts {
				parts[i] = hex.EncodeToString([]byte(part))
			}
			imp.localTypes[obj] = "GccgoCompatType_" + strings.Join(parts, "_")
			decls = append(decls, localDecl{file, spec, obj})
			return true
		})
	}
	for _, decl := range decls {
		body := decl.spec.Type
		ast.Inspect(body, func(node ast.Node) bool {
			if array, ok := node.(*ast.ArrayType); ok && array.Len != nil {
				if value := imp.info.Types[array.Len].Value; value != nil {
					// Bounds may refer to constants declared in the enclosing block.
					if expr, err := parser.ParseExpr(value.ExactString()); err == nil {
						array.Len = expr
					}
				}
			}
			if id, ok := node.(*ast.Ident); ok {
				if obj, ok := imp.info.Uses[id].(*types.TypeName); ok {
					if name := imp.localTypes[obj]; name != "" {
						id.Name = name
					}
				}
			}
			return true
		})
		name := imp.localTypes[decl.obj]
		id := ast.NewIdent(name)
		imp.info.Defs[id] = decl.obj
		global := &ast.TypeSpec{Name: id, Assign: decl.spec.Assign, Type: body}
		decl.file.Decls = append(decl.file.Decls, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{global}})
		// The lexical spelling remains usable only where originally declared.
		decl.spec.Assign = token.Pos(1)
		decl.spec.Type = ast.NewIdent(name)
	}
}
