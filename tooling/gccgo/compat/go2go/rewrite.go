// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package go2go

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var config = printer.Config{
	Mode:     printer.UseSpaces | printer.TabIndent | printer.SourcePos,
	Tabwidth: 8,
}

// isParameterizedFuncDecl reports whether fd is a parameterized function.
func isParameterizedFuncDecl(fd *ast.FuncDecl, info *types.Info) bool {
	if fd.Type.TypeParams != nil && len(fd.Type.TypeParams.List) > 0 {
		return true
	}
	if fd.Recv != nil {
		rtyp := info.TypeOf(fd.Recv.List[0].Type)
		if rtyp == nil {
			// Already instantiated.
			return false
		}
		if p, ok := rtyp.(*types.Pointer); ok {
			rtyp = p.Elem()
		}
		if named, ok := rtyp.(*types.Named); ok {
			if typeParams(named) != nil {
				return true
			}
		}
	}
	return false
}

// isTranslatableType reports whether a type spec can be translated to Go1.
// This is false if the type spec relies on any features that use generics.
func isTranslatableType(s ast.Spec, info *types.Info) bool {
	if isParameterizedTypeDecl(s, info) {
		return false
	}
	if isTypeBound(s, info) {
		return false
	}
	if embedsComparable(s, info) {
		return false
	}
	return true
}

// isParameterizedTypeDecl reports whether s is a parameterized type.
func isParameterizedTypeDecl(s ast.Spec, info *types.Info) bool {
	ts := s.(*ast.TypeSpec)
	if ts.TypeParams != nil && len(ts.TypeParams.List) > 0 {
		return true
	}
	if ts.Assign == token.NoPos {
		return false
	}

	// This is a type alias. Try to resolve it.
	typ := info.TypeOf(ts.Type)
	if typ == nil {
		return false
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	return len(typeParams(named)) > 0 && len(typeArgsList(named)) == 0
}

// isTypeBound reports whether s is an interface type that includes a
// type bound or that embeds an interface that must be a type bound.
func isTypeBound(s ast.Spec, info *types.Info) bool {
	typ := info.TypeOf(s.(*ast.TypeSpec).Type)
	if typ == nil {
		return false
	}
	if iface, ok := typ.Underlying().(*types.Interface); ok {
		if !iface.IsMethodSet() {
			return true
		}
	}
	return false
}

// embedsComparable reports whether s is an interface type that embeds
// the predeclared type "comparable", directly or indirectly.
func embedsComparable(s ast.Spec, info *types.Info) bool {
	typ := info.TypeOf(s.(*ast.TypeSpec).Type)
	return typeEmbedsComparable(typ)
}

// typeEmbedsComparable reports whether typ is an interface type
// that embeds the predeclared type "comparable", directly or indirectly.
// This is like embedsComparable, but for a types.Type.
func typeEmbedsComparable(typ types.Type) bool {
	if typ == nil {
		return false
	}
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	n := iface.NumEmbeddeds()
	if n == 0 {
		return false
	}
	comparable := types.Universe.Lookup("comparable")
	for i := 0; i < n; i++ {
		et := iface.EmbeddedType(i)
		if named, ok := et.(*types.Named); ok && named.Obj() == comparable {
			return true
		}
		if typeEmbedsComparable(et) {
			return true
		}
	}
	return false
}

// A translator is used to translate a file from generic Go to Go 1.
type translator struct {
	fset         *token.FileSet
	importer     *Importer
	tpkg         *types.Package
	types        map[ast.Expr]types.Type
	newDecls     []ast.Decl
	typePackages map[*types.Package]bool

	// typeDepth tracks recursive type instantiations.
	typeDepth int

	rangeNames        map[string]bool
	rangeSerial       int
	rangeFunction     *rangeFunction
	rangeFrames       []*rangeFrame
	stringSliceRanges map[*ast.RangeStmt]bool

	// err is set if we have seen an error during this translation.
	// This is used by the rewrite methods.
	err error
}

// instantiations tracks all function and type instantiations for a package.
type instantiations struct {
	funcInstantiations map[string][]*funcInstantiation
	typeInstantiations map[types.Type][]*typeInstantiation
}

// A funcInstantiation is a single instantiation of a function.
type funcInstantiation struct {
	types []types.Type
	decl  *ast.Ident
}

// A typeInstantiation is a single instantiation of a type.
type typeInstantiation struct {
	types      []types.Type
	decl       *ast.Ident
	typ        types.Type
	inProgress bool
}

// funcInstantiations fetches the function instantiations defined in
// the current package, given a generic function name.
func (t *translator) funcInstantiations(key string) []*funcInstantiation {
	insts := t.importer.instantiations[t.tpkg]
	if insts == nil {
		return nil
	}
	return insts.funcInstantiations[key]
}

// addFuncInstantiation adds a new function instantiation.
func (t *translator) addFuncInstantiation(key string, inst *funcInstantiation) {
	insts := t.pkgInstantiations()
	insts.funcInstantiations[key] = append(insts.funcInstantiations[key], inst)
}

// typeInstantiations fetches the type instantiations defined in
// the current package, given a generic type.
func (t *translator) typeInstantiations(typ types.Type) []*typeInstantiation {
	insts := t.importer.instantiations[t.tpkg]
	if insts == nil {
		return nil
	}
	return insts.typeInstantiations[typ]
}

// addTypeInstantiations adds a new type instantiation.
func (t *translator) addTypeInstantiation(typ types.Type, inst *typeInstantiation) {
	insts := t.pkgInstantiations()
	insts.typeInstantiations[typ] = append(insts.typeInstantiations[typ], inst)
}

// pkgInstantiations returns the instantiations structure for the current
// package, creating it if necessary.
func (t *translator) pkgInstantiations() *instantiations {
	insts := t.importer.instantiations[t.tpkg]
	if insts == nil {
		insts = &instantiations{
			funcInstantiations: make(map[string][]*funcInstantiation),
			typeInstantiations: make(map[types.Type][]*typeInstantiation),
		}
		t.importer.instantiations[t.tpkg] = insts
	}
	return insts
}

// rewrite rewrites the contents of one file.
func rewriteFile(dir string, fset *token.FileSet, importer *Importer, importPath string, tpkg *types.Package, filename string, file *ast.File, addImportableName bool) (err error) {
	if err := rewriteAST(fset, importer, importPath, tpkg, file, addImportableName); err != nil {
		return err
	}

	// Rewritten identifiers can be longer than their original source spans.
	// A flat positional comment list then moves an inline comment into a
	// selector or signature, potentially inserting an implicit semicolon.
	// Generated compiler inputs retain attached compiler pragmas; original
	// comments remain intact in the original sources and source sidecars.
	filter := func(group *ast.CommentGroup) *ast.CommentGroup {
		if group == nil {
			return nil
		}
		var comments []*ast.Comment
		for _, comment := range group.List {
			text := comment.Text
			if strings.HasPrefix(text, "//go:build ") || strings.HasPrefix(text, "// +build ") {
				continue // Target selection has already happened.
			}
			if strings.HasPrefix(text, "//go:") || strings.HasPrefix(text, "//gccgo:") || strings.HasPrefix(text, "//extern ") || strings.HasPrefix(text, "//line ") {
				comments = append(comments, comment)
			}
		}
		if len(comments) == 0 {
			return nil
		}
		return &ast.CommentGroup{List: comments}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.File:
			node.Doc = filter(node.Doc)
		case *ast.GenDecl:
			node.Doc = filter(node.Doc)
		case *ast.FuncDecl:
			node.Doc = filter(node.Doc)
		case *ast.ImportSpec:
			node.Doc, node.Comment = filter(node.Doc), filter(node.Comment)
		case *ast.ValueSpec:
			node.Doc, node.Comment = filter(node.Doc), filter(node.Comment)
		case *ast.TypeSpec:
			node.Doc, node.Comment = filter(node.Doc), filter(node.Comment)
		case *ast.Field:
			node.Doc, node.Comment = filter(node.Doc), filter(node.Comment)
		}
		return true
	})
	file.Comments = nil

	filename = filepath.Base(filename)
	goFile := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".go"
	o, err := os.Create(filepath.Join(dir, goFile))
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := o.Close(); err == nil {
			err = closeErr
		}
	}()

	w := bufio.NewWriter(o)
	defer func() {
		if flushErr := w.Flush(); err == nil {
			err = flushErr
		}
	}()
	fmt.Fprintln(w, rewritePrefix)

	return printCompilerFile(w, fset, file, importer.nativeAsm[file])
}

// rewriteAST rewrites the AST for a file.
func rewriteAST(fset *token.FileSet, importer *Importer, importPath string, tpkg *types.Package, file *ast.File, addImportableName bool) (err error) {
	t := translator{
		fset:         fset,
		importer:     importer,
		tpkg:         tpkg,
		types:        make(map[ast.Expr]types.Type),
		typePackages: make(map[*types.Package]bool),
	}
	t.reserveRangeNames(file)
	t.translate(file)
	if addImportableName {
		// Generic bodies may refer to private constants and ordinary types
		// in their defining package. Compiler-only aliases preserve those
		// objects; original source access is checked before lowering.
		for _, name := range tpkg.Scope().Names() {
			obj := tpkg.Scope().Lookup(name)
			if obj.Exported() {
				continue
			}
			bridge := ast.NewIdent("GccgoCompat" + string(nameSep) + name)
			switch obj := obj.(type) {
			case *types.Const:
				file.Decls = append(file.Decls, &ast.GenDecl{Tok: token.CONST,
					Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{bridge}, Values: []ast.Expr{ast.NewIdent(name)}}}})
			case *types.Func:
				if len(typeParams(obj.Type().(*types.Signature))) == 0 {
					sig := obj.Type().(*types.Signature)
					ft := t.typeToAST(sig).(*ast.FuncType)
					// Result names belong to the original function body. In
					// this forwarding body they can shadow its callee (MSAL's
					// codeVerifier returns a result named codeVerifier).
					if ft.Results != nil {
						for _, field := range ft.Results.List {
							field.Names = nil
						}
					}
					args := make([]ast.Expr, len(ft.Params.List))
					for i, field := range ft.Params.List {
						id := ast.NewIdent(fmt.Sprintf("gccgoArg%d", i))
						if id.Name == name {
							id.Name += "_"
						}
						field.Names = []*ast.Ident{id}
						args[i] = id
					}
					call := &ast.CallExpr{Fun: ast.NewIdent(name), Args: args}
					if sig.Variadic() {
						call.Ellipsis = token.Pos(1)
					}
					var statement ast.Stmt = &ast.ExprStmt{X: call}
					if sig.Results().Len() != 0 {
						statement = &ast.ReturnStmt{Results: []ast.Expr{call}}
					}
					// A wrapper has no initialization side effects and does
					// not retain an unused runtime import at link time.
					file.Decls = append(file.Decls, &ast.FuncDecl{Name: bridge, Type: ft,
						Body: &ast.BlockStmt{List: []ast.Stmt{statement}}})
				}
			case *types.Var:
				file.Decls = append(file.Decls, &ast.GenDecl{Tok: token.VAR,
					Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{bridge}, Values: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent(name)}}}}})
			case *types.TypeName:
				if named, ok := obj.Type().(*types.Named); ok && len(typeParams(named)) != 0 {
					continue
				}
				if iface, ok := obj.Type().Underlying().(*types.Interface); ok && !iface.IsMethodSet() {
					continue
				}
				file.Decls = append(file.Decls, &ast.GenDecl{Tok: token.TYPE,
					Specs: []ast.Spec{&ast.TypeSpec{Name: bridge, Assign: token.Pos(1), Type: ast.NewIdent(name)}}})
			}
		}
	}

	// Private bridges can introduce instantiated signature types after the
	// original declarations have been translated. Emit and translate those
	// new declarations before collecting their imports.
	if len(t.newDecls) != 0 {
		generated := &ast.File{Decls: t.newDecls}
		t.newDecls = nil
		t.translate(generated)
		file.Decls = append(file.Decls, generated.Decls...)
	}

	// Add all the transitive imports. This is more than we need,
	// but we're not trying to be elegant here.
	imps := make(map[string]bool)

	for _, p := range importer.transitiveImports(importPath) {
		imps[p] = true
	}
	for pkg := range t.typePackages {
		if pkg != t.tpkg {
			imps[pkg.Path()] = true
		}
	}

	decls := make([]ast.Decl, 0, len(file.Decls))
	var specs []ast.Spec
	// Synthetic imports must stay before original declaration comments.
	// NoPos can make go/printer consume go:embed or other pragmas while
	// printing the import block, changing their compiler attachment.
	importPos := file.Name.End()
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			decls = append(decls, decl)
			continue
		}
		for _, spec := range gen.Specs {
			imp := spec.(*ast.ImportSpec)
			// We picked up Go 2 imports above, but we still
			// need to pick up Go 1 imports here.
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil || imps[path] {
				continue
			}
			imps[path] = true
			for _, p := range importer.transitiveImports(path) {
				imps[p] = true
			}
		}
	}
	file.Decls = decls

	// If we have a ./ import, let it override a standard import
	// we may have added due to t.typePackages.
	for path := range imps {
		if strings.HasPrefix(path, "./") {
			delete(imps, strings.TrimPrefix(path, "./"))
		}
	}

	paths := make([]string, 0, len(imps))
	for p := range imps {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		specs = append(specs, ast.Spec(&ast.ImportSpec{
			Name: &ast.Ident{Name: compatImportName(p), NamePos: importPos},
			Path: &ast.BasicLit{
				Kind:     token.STRING,
				Value:    strconv.Quote(p),
				ValuePos: importPos,
			},
		}))
	}
	if len(specs) > 0 {
		first := &ast.GenDecl{
			Tok:    token.IMPORT,
			TokPos: importPos,
			Lparen: importPos,
			Rparen: importPos,
			Specs:  specs,
		}
		file.Decls = append([]ast.Decl{first}, file.Decls...)
	}

	// Add a name that other packages can reference to avoid an error
	// about an unused package.
	if addImportableName {
		file.Decls = append(file.Decls,
			&ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{
						Name: ast.NewIdent(t.importableName()),
						Type: ast.NewIdent("int"),
					},
				},
			})
	}

	// Unused transitive imports are needed only for package initialization.
	// Blank imports preserve that behavior without retaining arbitrary
	// exported function descriptors or colliding with package declarations.
	used := make(map[string]bool)
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				used[strings.SplitN(id.Name, ".", 2)[0]] = true
			}
			return true
		})
	}
	for _, spec := range specs {
		imp := spec.(*ast.ImportSpec)
		if !used[imp.Name.Name] {
			imp.Name = &ast.Ident{Name: "_", NamePos: importPos}
		}
	}

	return t.err
}

// translate translates the AST for a file from generic Go to Go 1.
func (t *translator) translate(file *ast.File) {
	declsToDo := file.Decls
	file.Decls = nil
	c := 0
	for len(declsToDo) > 0 {
		if c > 50 {
			var sb strings.Builder
			printer.Fprint(&sb, t.fset, declsToDo[0])
			t.err = fmt.Errorf("looping while expanding %v", &sb)
			return
		}
		c++

		newDecls := make([]ast.Decl, 0, len(declsToDo))
		for i, decl := range declsToDo {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if !isParameterizedFuncDecl(decl, t.importer.info) {
					t.translateFuncDecl(&declsToDo[i])
					newDecls = append(newDecls, decl)
				}
			case *ast.GenDecl:
				switch decl.Tok {
				case token.TYPE:
					newSpecs := make([]ast.Spec, 0, len(decl.Specs))
					for j := range decl.Specs {
						if isTranslatableType(decl.Specs[j], t.importer.info) {
							t.translateTypeSpec(&decl.Specs[j])
							newSpecs = append(newSpecs, decl.Specs[j])
						}
					}
					if len(newSpecs) == 0 {
						decl = nil
					} else {
						decl.Specs = newSpecs
					}
				case token.VAR, token.CONST:
					for j := range decl.Specs {
						t.translateValueSpec(&decl.Specs[j])
					}
				}
				if decl != nil {
					newDecls = append(newDecls, decl)
				}
			default:
				newDecls = append(newDecls, decl)
			}
		}
		file.Decls = append(file.Decls, newDecls...)
		declsToDo = t.newDecls
		t.newDecls = nil
	}
}

// translateTypeSpec translates a type from generic Go to Go 1.
func (t *translator) translateTypeSpec(ps *ast.Spec) {
	ts := (*ps).(*ast.TypeSpec)
	if ts.TypeParams != nil && len(ts.TypeParams.List) > 0 {
		t.err = fmt.Errorf("%s: go2go tool does not support parameterized type here", t.fset.Position((*ps).Pos()))
		return
	}
	t.translateExpr(&ts.Type)
}

// translateValueSpec translates a variable or constant from generic Go to Go 1.
func (t *translator) translateValueSpec(ps *ast.Spec) {
	vs := (*ps).(*ast.ValueSpec)
	t.translateExpr(&vs.Type)
	for i := range vs.Values {
		t.translateExpr(&vs.Values[i])
	}
}

// translateFuncDecl translates a function from generic Go to Go 1.
func (t *translator) translateFuncDecl(pd *ast.Decl) {
	if t.err != nil {
		return
	}
	fd := (*pd).(*ast.FuncDecl)
	restore := t.enterRangeFunction(fd.Type, fd.Body)
	defer restore()
	if fd.Type.TypeParams != nil {
		if len(fd.Type.TypeParams.List) > 0 {
			panic("parameterized function")
		}
		fd.Type.TypeParams = nil
	}
	if fd.Recv != nil {
		t.translateFieldList(fd.Recv)
	}
	t.translateFieldList(fd.Type.Params)
	t.translateFieldList(fd.Type.Results)
	t.translateBlockStmt(fd.Body)
}

// translateBlockStmt translates a block statement from generic Go to Go 1.
func (t *translator) translateBlockStmt(pbs *ast.BlockStmt) {
	if pbs == nil {
		return
	}
	t.lowerStoredStringSequences(pbs)
	for i := range pbs.List {
		t.translateStmt(&pbs.List[i])
	}
}

// translateStmt translates a statement from generic Go to Go 1.
func (t *translator) translateStmt(ps *ast.Stmt) {
	if t.err != nil {
		return
	}
	if *ps == nil {
		return
	}
	switch s := (*ps).(type) {
	case *ast.DeclStmt:
		d := s.Decl.(*ast.GenDecl)
		switch d.Tok {
		case token.TYPE:
			for i := range d.Specs {
				t.translateTypeSpec(&d.Specs[i])
			}
		case token.CONST, token.VAR:
			for i := range d.Specs {
				t.translateValueSpec(&d.Specs[i])
			}
		default:
			panic(fmt.Sprintf("unknown decl type %v", d.Tok))
		}
	case *ast.EmptyStmt:
	case *ast.LabeledStmt:
		if loop, ok := s.Stmt.(*ast.RangeStmt); ok && !t.lowerStringSequenceRange(loop) && t.functionRangeSignature(loop) != nil {
			t.err = fmt.Errorf("%s: labeled function range is not yet supported by gccgo compatibility", t.fset.Position(s.Pos()))
			return
		}
		t.translateStmt(&s.Stmt)
	case *ast.ExprStmt:
		t.translateExpr(&s.X)
	case *ast.SendStmt:
		t.translateExpr(&s.Chan)
		t.translateExpr(&s.Value)
	case *ast.IncDecStmt:
		t.translateExpr(&s.X)
	case *ast.AssignStmt:
		t.translateExprList(s.Lhs)
		t.translateExprList(s.Rhs)
	case *ast.GoStmt:
		e := ast.Expr(s.Call)
		t.translateExpr(&e)
		s.Call = e.(*ast.CallExpr)
	case *ast.DeferStmt:
		e := ast.Expr(s.Call)
		t.translateExpr(&e)
		s.Call = e.(*ast.CallExpr)
	case *ast.ReturnStmt:
		t.translateExprList(s.Results)
	case *ast.BranchStmt:
	case *ast.BlockStmt:
		t.translateBlockStmt(s)
	case *ast.IfStmt:
		t.translateStmt(&s.Init)
		t.translateExpr(&s.Cond)
		t.translateBlockStmt(s.Body)
		t.translateStmt(&s.Else)
	case *ast.CaseClause:
		t.translateExprList(s.List)
		t.translateStmtList(s.Body)
	case *ast.SwitchStmt:
		t.translateStmt(&s.Init)
		t.translateExpr(&s.Tag)
		t.translateBlockStmt(s.Body)
	case *ast.TypeSwitchStmt:
		t.translateStmt(&s.Init)
		t.translateStmt(&s.Assign)
		t.translateBlockStmt(s.Body)
	case *ast.CommClause:
		t.translateStmt(&s.Comm)
		t.translateStmtList(s.Body)
	case *ast.SelectStmt:
		t.translateBlockStmt(s.Body)
	case *ast.ForStmt:
		t.translateStmt(&s.Init)
		t.translateExpr(&s.Cond)
		t.translateStmt(&s.Post)
		t.translateBlockStmt(s.Body)
	case *ast.RangeStmt:
		if !t.lowerStringSequenceRange(s) && t.functionRangeSignature(s) != nil {
			*ps = t.lowerFunctionRange(s)
			return
		}
		t.translateExpr(&s.Key)
		t.translateExpr(&s.Value)
		t.translateExpr(&s.X)
		t.translateBlockStmt(s.Body)
	default:
		panic(fmt.Sprintf("unimplemented Stmt %T", s))
	}
}

// translateStmtList translates a list of statements generic Go to Go 1.
func (t *translator) translateStmtList(sl []ast.Stmt) {
	for i := range sl {
		t.translateStmt(&sl[i])
	}
}

// translateExpr translates an expression from generic Go to Go 1.
func (t *translator) translateExpr(pe *ast.Expr) {
	if t.err != nil {
		return
	}
	if *pe == nil {
		return
	}
	switch e := (*pe).(type) {
	case *ast.Ident:
		t.translateIdent(pe)
	case *ast.Ellipsis:
		t.translateExpr(&e.Elt)
	case *ast.BasicLit:
	case *ast.FuncLit:
		restore := t.enterRangeFunction(e.Type, e.Body)
		defer restore()
		t.translateFieldList(e.Type.TypeParams)
		t.translateFieldList(e.Type.Params)
		t.translateFieldList(e.Type.Results)
		t.translateBlockStmt(e.Body)
	case *ast.CompositeLit:
		// go/types has already inferred an ellipsis array's exact length.
		// Emit it explicitly: gccgo can infer too small a length when keyed
		// indexes refer to constants declared in a later source file.
		if syntax, ok := e.Type.(*ast.ArrayType); ok {
			if _, inferred := syntax.Len.(*ast.Ellipsis); inferred {
				if array, ok := t.lookupType(e).(*types.Array); ok {
					syntax.Len = &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(array.Len())}
				}
			}
		}
		// A checked template can construct an ordinary private struct from
		// its defining package. GCC rejects imported struct literals even
		// after selector ownership is restored. Construct the identical
		// underlying struct, then apply the normal Go value conversion.
		originalType := e.Type
		foreignStruct := false
		if named, ok := t.lookupType(e.Type).(*types.Named); ok && len(e.Elts) != 0 && named.Obj().Pkg() != nil && named.Obj().Pkg() != t.tpkg && len(typeArgsList(named)) == 0 {
			if underlying, ok := named.Underlying().(*types.Struct); ok {
				for i := 0; i < underlying.NumFields(); i++ {
					field := underlying.Field(i)
					if field.Exported() {
						continue
					}
					for _, element := range e.Elts {
						keyed, ok := element.(*ast.KeyValueExpr)
						if !ok {
							foreignStruct = true
							continue
						}
						if key, ok := keyed.Key.(*ast.Ident); ok && (key.Name == field.Name() || key.Name == compatMemberName(field)) {
							foreignStruct = true
						}
					}
				}
				if foreignStruct {
					e.Type = t.typeToAST(underlying)
				}
			}
		}
		t.translateExpr(&e.Type)
		t.translateExprList(e.Elts)
		if foreignStruct {
			t.translateExpr(&originalType)
			*pe = &ast.CallExpr{Fun: originalType, Args: []ast.Expr{e}}
		}
	case *ast.ParenExpr:
		t.translateExpr(&e.X)
	case *ast.SelectorExpr:
		t.translateSelectorExpr(pe)
	case *ast.IndexExpr:
		t.translateFunctionOrTypeInstantiation(pe)
		t.translateExpr(&e.X)
		t.translateExpr(&e.Index)
	case *ast.SliceExpr:
		t.translateExpr(&e.X)
		t.translateExpr(&e.Low)
		t.translateExpr(&e.High)
		t.translateExpr(&e.Max)
	case *ast.TypeAssertExpr:
		t.translateExpr(&e.X)
		t.translateExpr(&e.Type)
	case *ast.CallExpr:
		var arrayConversion *types.Array
		var sourceSlice types.Type
		if len(e.Args) == 1 {
			if target := t.lookupType(e.Fun); target != nil {
				if array, ok := target.Underlying().(*types.Array); ok {
					if source := t.lookupType(e.Args[0]); source != nil {
						if _, ok := source.Underlying().(*types.Slice); ok {
							arrayConversion, sourceSlice = array, source
						}
					}
				}
			}
		}
		if inst, ok := t.importer.inferred[e]; ok && inst.Sig != nil {
			switch e.Fun.(type) {
			case *ast.Ident, *ast.SelectorExpr:
				t.translateFunctionInstantiation(pe)
			}
		}
		t.translateExprList(e.Args)
		t.translateExpr(&e.Fun)
		if arrayConversion != nil {
			if arrayConversion.Len() == 0 {
				// A nil slice converts to a zero-length array successfully,
				// but converting it to *[0]T gives nil. Evaluate the source
				// once and return the empty array without dereferencing it.
				e.Fun = &ast.FuncLit{
					Type: &ast.FuncType{
						Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("_")}, Type: t.typeToAST(sourceSlice)}}},
						Results: &ast.FieldList{List: []*ast.Field{{Type: e.Fun}}},
					},
					Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CompositeLit{Type: e.Fun}}}}},
				}
			} else {
				// GCC already implements Go 1.17's checked slice-to-array
				// pointer conversion. Dereferencing yields Go 1.20's value
				// copy, retaining its length check and single evaluation.
				e.Fun = &ast.ParenExpr{X: &ast.StarExpr{X: e.Fun}}
				*pe = &ast.StarExpr{X: e}
			}
		}
	case *ast.IndexListExpr:
		t.translateFunctionOrTypeInstantiation(pe)
		t.translateExpr(&e.X)
		t.translateExprList(e.Indices)
	case *ast.StarExpr:
		t.translateExpr(&e.X)
	case *ast.UnaryExpr:
		_, addressedLiteral := e.X.(*ast.CompositeLit)
		t.translateExpr(&e.X)
		if e.Op == token.AND && addressedLiteral {
			if converted, ok := e.X.(*ast.CallExpr); ok && len(converted.Args) == 1 {
				if _, ok := converted.Args[0].(*ast.CompositeLit); ok {
					// Preserve the address of the allocated literal, rather
					// than attempting to address a converted struct value.
					*pe = &ast.CallExpr{Fun: &ast.ParenExpr{X: &ast.StarExpr{X: converted.Fun}},
						Args: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: converted.Args[0]}}}
				}
			}
		}
	case *ast.BinaryExpr:
		t.translateExpr(&e.X)
		t.translateExpr(&e.Y)
	case *ast.KeyValueExpr:
		t.translateExpr(&e.Key)
		t.translateExpr(&e.Value)
	case *ast.ArrayType:
		t.translateExpr(&e.Len)
		t.translateExpr(&e.Elt)
	case *ast.StructType:
		t.translateFieldList(e.Fields)
	case *ast.FuncType:
		t.translateFieldList(e.TypeParams)
		t.translateFieldList(e.Params)
		t.translateFieldList(e.Results)
	case *ast.InterfaceType:
		t.lowerUniversalInterfaceUnions(e)
		methods, types := splitFieldList(e.Methods)
		t.translateFieldList(methods)
		t.translateExprList(types)
	case *ast.MapType:
		t.translateExpr(&e.Key)
		t.translateExpr(&e.Value)
	case *ast.ChanType:
		t.translateExpr(&e.Value)
	default:
		panic(fmt.Sprintf("unimplemented Expr %T", e))
	}
}

// translateIdent translates a simple identifier generic Go to Go 1.
// These are usually fine as is, but a reference
// to a non-generic name in another package may need a package qualifier.
func (t *translator) translateIdent(pe *ast.Expr) {
	e := (*pe).(*ast.Ident)
	obj := t.importer.info.ObjectOf(e)
	// In an embedded field such as struct{ Reader } with a dot-imported
	// io.Reader, go/types records both Defs[e] (the field variable) and
	// Uses[e] (the imported type). ObjectOf prefers Defs, but this expression
	// is the type spelling and must retain its package qualifier.
	if typeName, ok := t.importer.info.Uses[e].(*types.TypeName); ok {
		obj = typeName
	}
	if obj == nil {
		return
	}
	if pkg, ok := obj.(*types.PkgName); ok {
		t.typePackages[pkg.Imported()] = true
		id := ast.NewIdent(compatImportName(pkg.Imported().Path()))
		t.importer.info.Uses[id] = pkg
		*pe = id
		return
	}
	// A field may itself have a generic named type. Restore its defining
	// package before the generic-type early return below, which applies to
	// type names, not to private keys in imported struct literals.
	if field, ok := obj.(*types.Var); ok && field.IsField() && !obj.Exported() && obj.Pkg() != nil && obj.Pkg() != t.tpkg {
		*pe = ast.NewIdent(compatMemberName(obj))
		return
	}
	if sig, ok := obj.Type().(*types.Signature); ok && len(typeParams(sig)) > 0 {
		if _, known := t.importer.inferred[e]; known {
			t.translateFunctionInstantiation(pe)
			return
		}
	}
	if named, ok := obj.Type().(*types.Named); ok && len(typeParams(named)) > 0 {
		// A generic function that will be instantiated locally.
		return
	}
	ipkg := obj.Pkg()
	if ipkg == nil || ipkg == t.tpkg {
		// We don't need a package qualifier if it's defined
		// in the current package.
		return
	}
	if obj.Parent() != ipkg.Scope() {
		// We only need a package qualifier if it's defined in
		// package scope.
		return
	}
	t.typePackages[ipkg] = true

	// Add package qualifier.
	if !obj.Exported() {
		switch obj.(type) {
		case *types.Const, *types.TypeName, *types.Func, *types.Var:
			e = ast.NewIdent("GccgoCompat" + string(nameSep) + obj.Name())
		}
	}
	*pe = &ast.SelectorExpr{
		X:   ast.NewIdent(compatImportName(ipkg.Path())),
		Sel: e,
	}
	if _, ok := obj.(*types.Var); ok && !obj.Exported() {
		*pe = &ast.ParenExpr{X: &ast.StarExpr{X: *pe}}
	}
}

// translateSelectorExpr translates a selector expression
// from generic Go to Go 1.
func (t *translator) translateSelectorExpr(pe *ast.Expr) {
	e := (*pe).(*ast.SelectorExpr)

	t.translateExpr(&e.X)

	obj := t.importer.info.ObjectOf(e.Sel)
	if obj == nil {
		return
	}
	if obj.Pkg() != nil && obj.Pkg() != t.tpkg && !obj.Exported() {
		e.Sel = ast.NewIdent(compatMemberName(obj))
	}

	// The native frontend restores the original embedded field name from
	// specialization metadata, preserving keyed literals, promoted methods
	// and reflection without introducing access to private embedded fields.
}

// TODO(iant) refactor code and get rid of this?
func splitFieldList(fl *ast.FieldList) (methods *ast.FieldList, types []ast.Expr) {
	if fl == nil {
		return
	}
	var mfields []*ast.Field
	for _, f := range fl.List {
		if len(f.Names) > 0 && f.Names[0].Name == "type" {
			// type list type
			types = append(types, f.Type)
		} else {
			mfields = append(mfields, f)
		}
	}
	copy := *fl
	copy.List = mfields
	methods = &copy
	return
}

// TODO(iant) refactor code and get rid of this?
func mergeFieldList(methods *ast.FieldList, types []ast.Expr) (fl *ast.FieldList) {
	fl = methods
	if len(types) == 0 {
		return
	}
	if fl == nil {
		fl = new(ast.FieldList)
	}
	name := []*ast.Ident{ast.NewIdent("type")}
	for _, typ := range types {
		fl.List = append(fl.List, &ast.Field{Names: name, Type: typ})
	}
	return
}

// translateExprList translate an expression list generic Go to Go 1.
func (t *translator) translateExprList(el []ast.Expr) {
	for i := range el {
		t.translateExpr(&el[i])
	}
}

// translateFieldList translates a field list generic Go to Go 1.
func (t *translator) translateFieldList(fl *ast.FieldList) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		t.translateField(f)
	}
}

// translateField translates a field generic Go to Go 1.
func (t *translator) translateField(f *ast.Field) {
	t.translateExpr(&f.Type)
	for i, id := range f.Names {
		if obj, ok := t.importer.info.Defs[id].(*types.Var); ok && obj.IsField() && obj.Pkg() != nil && obj.Pkg() != t.tpkg && !obj.Exported() {
			f.Names[i] = ast.NewIdent(compatMemberName(obj))
		}
	}
}

// translateFunctionInstantiation translates an instantiated function
// to Go 1.
func (t *translator) translateFunctionInstantiation(pe *ast.Expr) {
	expr := *pe
	qid := t.instantiatedIdent(expr)
	argList, typeList, typeArgs := t.instantiationTypes(expr)
	if t.err != nil {
		return
	}

	var instIdent *ast.Ident
	key := qid.String()
	insts := t.funcInstantiations(key)
	for _, inst := range insts {
		if t.sameTypes(typeList, inst.types) {
			instIdent = inst.decl
			break
		}
	}

	if instIdent == nil {
		var err error
		instIdent, err = t.instantiateFunction(qid, argList, typeList)
		if err != nil {
			t.err = err
			return
		}

		n := &funcInstantiation{
			types: typeList,
			decl:  instIdent,
		}
		t.addFuncInstantiation(key, n)
	}

	if typeArgs {
		*pe = instIdent
	} else {
		switch e := expr.(type) {
		case *ast.CallExpr:
			newCall := *e
			newCall.Fun = instIdent
			*pe = &newCall
		case *ast.IndexExpr, *ast.IndexListExpr, *ast.Ident, *ast.SelectorExpr:
			*pe = instIdent
		default:
			panic("unexpected AST type")
		}
	}
}

// translateTypeInstantiation translates an instantiated type to Go 1.
func (t *translator) translateTypeInstantiation(pe *ast.Expr) {
	expr := *pe
	qid := t.instantiatedIdent(expr)
	typ := t.findTypesObject(qid).Type().(*types.Named)
	argList, typeList, typeArgs := t.instantiationTypes(expr)
	if t.err != nil {
		return
	}
	if !typeArgs {
		panic("no type arguments for type")
	}

	var seen *typeInstantiation
	key := t.typeWithoutArgs(typ)
	for _, inst := range t.typeInstantiations(key) {
		if t.sameTypes(typeList, inst.types) {
			if inst.inProgress {
				panic(fmt.Sprintf("%s: circular type instantiation", t.fset.Position((*pe).Pos())))
			}
			if inst.decl == nil {
				// This can happen if we've instantiated
				// the type in instantiateType.
				seen = inst
				break
			}
			*pe = inst.decl
			return
		}
	}

	name, err := t.instantiatedName(qid, typeList)
	if err != nil {
		t.err = err
		return
	}
	instIdent := ast.NewIdent(name)

	if seen != nil {
		seen.decl = instIdent
		seen.inProgress = true
	} else {
		seen = &typeInstantiation{
			types:      typeList,
			decl:       instIdent,
			typ:        nil,
			inProgress: true,
		}
		t.addTypeInstantiation(key, seen)
	}

	defer func() {
		seen.inProgress = false
	}()

	instType, err := t.instantiateTypeDecl(qid, typ, argList, typeList, instIdent)
	if err != nil {
		t.err = err
		return
	}

	if seen.typ == nil {
		seen.typ = instType
	}

	*pe = instIdent
}

// instantiatedIdent returns the qualified identifer that is being
// instantiated.
func (t *translator) instantiatedIdent(x ast.Expr) qualifiedIdent {
	var fun ast.Expr
	switch x := x.(type) {
	case *ast.CallExpr:
		fun = x.Fun
	case *ast.IndexExpr:
		fun = x.X
	case *ast.IndexListExpr:
		fun = x.X
	case *ast.Ident, *ast.SelectorExpr:
		fun = x
	default:
		panic(fmt.Sprintf("unexpected AST %T", x))
	}

	switch fun := fun.(type) {
	case *ast.Ident:
		if obj := t.importer.info.Uses[fun]; obj != nil && obj.Pkg() != t.tpkg {
			return qualifiedIdent{pkg: obj.Pkg(), ident: fun}
		}
		return qualifiedIdent{ident: fun}
	case *ast.SelectorExpr:
		if obj := t.importer.info.Uses[fun.Sel]; obj != nil && obj.Pkg() != nil {
			return qualifiedIdent{pkg: obj.Pkg(), ident: fun.Sel}
		}
		pkgname, ok := fun.X.(*ast.Ident)
		if !ok {
			break
		}
		pkgobj, ok := t.importer.info.Uses[pkgname]
		if !ok {
			break
		}
		pn, ok := pkgobj.(*types.PkgName)
		if !ok {
			break
		}
		return qualifiedIdent{pkg: pn.Imported(), ident: fun.Sel}
	}
	panic(fmt.Sprintf("instantiated object %T %v is not an identifier", fun, fun))
}

// instantiationTypes returns the type arguments of an instantiation.
// It also returns the AST arguments if they are present.
// The typeArgs result reports whether the AST arguments are types.
func (t *translator) instantiationTypes(x ast.Expr) (argList []ast.Expr, typeList []types.Type, typeArgs bool) {
	var args []ast.Expr
	switch x := x.(type) {
	case *ast.CallExpr:
		// CallExprs may result in an implicit instantiation via type inference.
	case *ast.IndexExpr:
		args = unpackExpr(x.Index)
		typeArgs = true
	case *ast.IndexListExpr:
		args = x.Indices
		typeArgs = true
	case *ast.Ident, *ast.SelectorExpr:
	default:
		panic(fmt.Sprintf("unexpected AST type %T", x))
	}
	inferredInfo := t.importer.inferred
	inferred, haveInferred := inferredInfo[x]

	if !haveInferred {
		argList = args
		typeList = make([]types.Type, 0, len(argList))
		for _, arg := range argList {
			if id, ok := arg.(*ast.Ident); ok && id.Name == "_" {
				t.err = fmt.Errorf("%s: go2go tool does not support using _ here", t.fset.Position(arg.Pos()))
				return
			}
			if at := t.lookupType(arg); at == nil {
				panic(fmt.Sprintf("%s: no type found for %T %v", t.fset.Position(arg.Pos()), arg, arg))
			} else {
				typeList = append(typeList, at)
			}
		}
		typeArgs = true
	} else {
		typeList, argList = t.typeListToASTList(inferred.Targs)
	}

	// Ordinary local definitions have canonical package-level compiler names.
	// Unlifted types in generic bodies still require per-instantiation identity.
	for _, typ := range typeList {
		if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != nil {
			if scope := named.Obj().Parent(); scope != nil && scope != named.Obj().Pkg().Scope() && t.importer.localTypes[named.Obj()] == "" {
				t.err = fmt.Errorf("%s: go2go tool does not support using locally defined type as type argument", t.fset.Position(x.Pos()))
				return
			}
		}
	}

	return
}

// unpackExpr unpacks an *ast.ListExpr into a list of ast.Expr.
func unpackExpr(x ast.Expr) []ast.Expr {
	if x != nil {
		return []ast.Expr{x}
	}
	return nil
}

// lookupInstantiatedType looks for an existing instantiation of an
// instantiated type.
func (t *translator) lookupInstantiatedType(typ *types.Named) (types.Type, *ast.Ident) {
	copyType := func(typ *types.Named, newName string) types.Type {
		nm := typ.NumMethods()
		methods := make([]*types.Func, 0, nm)
		for i := 0; i < nm; i++ {
			methods = append(methods, typ.Method(i))
		}
		obj := typ.Obj()
		obj = types.NewTypeName(obj.Pos(), obj.Pkg(), newName, nil)
		nt := types.NewNamed(obj, typ.Underlying(), methods)
		return nt
	}

	targs := typeArgsList(typ)
	key := t.typeWithoutArgs(typ)
	var seen *typeInstantiation
	for _, inst := range t.typeInstantiations(key) {
		if t.sameTypes(targs, inst.types) {
			if inst.inProgress {
				panic(fmt.Sprintf("instantiation for %v in progress", typ))
			}
			if inst.decl == nil {
				// This can happen if we've instantiated
				// the type in instantiateType.
				seen = inst
				break
			}
			if inst.typ == nil {
				panic(fmt.Sprintf("no type for instantiation entry for %v", typ))
			}
			if instNamed, ok := inst.typ.(*types.Named); ok {
				return copyType(instNamed, inst.decl.Name), inst.decl
			}
			return inst.typ, inst.decl
		}
	}

	typeList, argList := t.typeListToASTList(targs)

	qid := qualifiedIdent{ident: ast.NewIdent(typ.Obj().Name())}
	if typPkg := typ.Obj().Pkg(); typPkg != t.tpkg {
		qid.pkg = typPkg
	}

	name, err := t.instantiatedName(qid, typeList)
	if err != nil {
		t.err = err
		return nil, nil
	}
	instIdent := ast.NewIdent(name)

	if seen != nil {
		seen.decl = instIdent
		seen.inProgress = true
	} else {
		seen = &typeInstantiation{
			types:      targs,
			decl:       instIdent,
			typ:        nil,
			inProgress: true,
		}
		t.addTypeInstantiation(key, seen)
	}

	defer func() {
		seen.inProgress = false
	}()

	instType, err := t.instantiateTypeDecl(qid, typ, argList, typeList, instIdent)
	if err != nil {
		t.err = err
		return nil, nil
	}

	if seen.typ == nil {
		seen.typ = instType
	} else {
		instType = seen.typ
	}

	if instNamed, ok := instType.(*types.Named); ok {
		return copyType(instNamed, instIdent.Name), instIdent
	}
	return instType, instIdent
}

// typeWithoutArgs takes a named type with arguments and returns the
// same type without arguments.
func (t *translator) typeWithoutArgs(typ *types.Named) *types.Named {
	return typ.Origin()
}

// typeListToASTList returns an AST list for a type list,
// as well as an updated type list.
func (t *translator) typeListToASTList(typeList []types.Type) ([]types.Type, []ast.Expr) {
	argList := make([]ast.Expr, 0, len(typeList))
	for _, typ := range typeList {
		argList = append(argList, t.typeToAST(typ))

		// This inferred type may introduce a reference to
		// packages that we don't otherwise import, and that
		// package name may wind up in arg. Record all packages
		// seen in inferred types so that we add import
		// statements for them, just in case.
		t.addTypePackages(typ)
	}
	return typeList, argList
}

// relativeTo is like types.RelativeTo, but returns just the package name,
// not the package path.
func relativeTo(pkg *types.Package) types.Qualifier {
	return func(other *types.Package) string {
		if pkg == other {
			return "" // same package; unqualified
		}
		return other.Name()
	}
}

// sameTypes reports whether two type slices are the same.
func (t *translator) sameTypes(a, b []types.Type) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if !t.sameType(x, b[i]) {
			return false
		}
	}
	return true
}

// sameType reports whether two types are the same.
// We have to check type arguments ourselves.
func (t *translator) sameType(a, b types.Type) bool {
	a, b = types.Unalias(a), types.Unalias(b)
	if types.Identical(a, b) {
		return true
	}
	if ap, bp := asPointer(a), asPointer(b); ap != nil && bp != nil {
		a = ap.Elem()
		b = bp.Elem()
	}
	an, ok := a.(*types.Named)
	if !ok {
		return false
	}
	bn, ok := b.(*types.Named)
	if !ok {
		return false
	}
	if an.Obj().Name() != bn.Obj().Name() {
		return false
	}
	if an.Obj().Pkg() != bn.Obj().Pkg() && (an.Obj().Pkg() == nil || bn.Obj().Pkg() == nil || an.Obj().Pkg().Path() != bn.Obj().Pkg().Path()) {
		return false
	}
	if len(typeArgsList(an)) == 0 || len(typeArgsList(an)) != len(typeArgsList(bn)) {
		return false
	}
	for i, typ := range typeArgsList(an) {
		if !t.sameType(typ, typeArgsList(bn)[i]) {
			return false
		}
	}
	return true
}

// qualifiedIdent is an identifier possibly qualified with a package.
type qualifiedIdent struct {
	pkg   *types.Package // identifier's package; nil for current package
	ident *ast.Ident
}

// String returns a printable name for qid.
func (qid qualifiedIdent) String() string {
	if qid.pkg == nil {
		return qid.ident.Name
	}
	return qid.pkg.Path() + "." + qid.ident.Name
}
