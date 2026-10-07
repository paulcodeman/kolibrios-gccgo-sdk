// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package go2go

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// This is the go/ast adaptation of cmd/compile/internal/rangefunc/rewrite.go
// (Go 1.23). Keep its callback, #next and RF_State protocol: a pull iterator
// would move user code to another goroutine and change return/panic semantics.
// Labeled transfers and defer in the range body need frontend/runtime support
// beyond this adaptation; reject them rather than attach them to the callback.
const (
	rfDone = iota
	rfReady
	rfPanic
	rfExhausted
	rfMissingPanic
)

type rangeFunction struct{ results []string }
type rangeFrame struct{ state, next string }

func (t *translator) reserveRangeNames(node ast.Node) {
	if t.rangeNames == nil {
		t.rangeNames = make(map[string]bool)
	}
	ast.Inspect(node, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			t.rangeNames[id.Name] = true
		}
		return true
	})
	if t.tpkg != nil {
		for _, name := range t.tpkg.Scope().Names() {
			t.rangeNames[name] = true
		}
	}
}

func (t *translator) rangeName() string {
	for {
		t.rangeSerial++
		name := fmt.Sprintf("gccgoRange%d", t.rangeSerial)
		if !t.rangeNames[name] {
			t.rangeNames[name] = true
			return name
		}
	}
}

func (t *translator) functionRangeSignature(s *ast.RangeStmt) *types.Signature {
	if t.stringSliceRanges[s] {
		return nil
	}
	typ := t.lookupType(s.X)
	if typ == nil {
		return nil
	}
	sig, _ := typ.Underlying().(*types.Signature)
	return sig
}

// Like upstream, make results named before moving a valued return into yield.
// Rename existing results by object identity, so shadowed result names inside
// the body cannot capture the generated return assignment.
func (t *translator) enterRangeFunction(ft *ast.FuncType, body *ast.BlockStmt) func() {
	oldFunction, oldFrames := t.rangeFunction, t.rangeFrames
	t.rangeFunction, t.rangeFrames = &rangeFunction{}, nil
	if body != nil {
		t.reserveRangeNames(body)
		hasRange := false
		ast.Inspect(body, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			if loop, ok := n.(*ast.RangeStmt); ok && t.functionRangeSignature(loop) != nil {
				hasRange = true
			}
			return true
		})
		if hasRange && ft.Results != nil {
			renames := make(map[types.Object]string)
			for _, field := range ft.Results.List {
				if len(field.Names) == 0 {
					field.Names = []*ast.Ident{ast.NewIdent(t.rangeName())}
				}
				for _, id := range field.Names {
					name := t.rangeName()
					if obj := t.importer.info.Defs[id]; obj != nil && id.Name != "_" {
						renames[obj] = name
					}
					id.Name = name
					t.rangeFunction.results = append(t.rangeFunction.results, name)
				}
			}
			ast.Inspect(body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					if name := renames[t.importer.info.ObjectOf(id)]; name != "" {
						id.Name = name
					}
				}
				return true
			})
		}
	}
	return func() { t.rangeFunction, t.rangeFrames = oldFunction, oldFrames }
}

func rangeInt(n int) ast.Expr { return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(n)} }
func rangeAssign(name string, value ast.Expr, tok token.Token) ast.Stmt {
	return &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(name)}, Tok: tok, Rhs: []ast.Expr{value}}
}
func rangeCompare(name string, op token.Token, value int) ast.Expr {
	return &ast.BinaryExpr{X: ast.NewIdent(name), Op: op, Y: rangeInt(value)}
}
func rangeIf(cond ast.Expr, body ...ast.Stmt) ast.Stmt {
	return &ast.IfStmt{Cond: cond, Body: &ast.BlockStmt{List: body}}
}
func (f *rangeFrame) yieldReturn(value bool) []ast.Stmt {
	state, op := rfDone, token.NEQ
	if value {
		state, op = rfReady, token.EQL
	}
	// Predeclared true/false may be shadowed by original range variables or
	// body declarations. Comparison constants do not consult those bindings.
	result := &ast.BinaryExpr{X: rangeInt(0), Op: op, Y: rangeInt(0)}
	return []ast.Stmt{rangeAssign(f.state, rangeInt(state), token.ASSIGN), &ast.ReturnStmt{Results: []ast.Expr{result}}}
}
func (t *translator) rangePanic(state ast.Expr) ast.Stmt {
	pkg, err := t.importer.ImportFrom("runtime", "", 0)
	if err != nil {
		t.err = err
		return &ast.EmptyStmt{}
	}
	t.typePackages[pkg] = true
	return &ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(compatImportName("runtime")), Sel: ast.NewIdent("GccgoCompatPanicRangeState")}, Args: []ast.Expr{state}}}
}

func (t *translator) lowerFunctionRange(loop *ast.RangeStmt) ast.Stmt {
	sig := t.functionRangeSignature(loop)
	if sig == nil || sig.Params().Len() != 1 {
		t.err = fmt.Errorf("%s: invalid function range signature", t.fset.Position(loop.Pos()))
		return loop
	}
	yield, ok := sig.Params().At(0).Type().Underlying().(*types.Signature)
	if !ok {
		t.err = fmt.Errorf("invalid range yield signature")
		return loop
	}
	f := &rangeFrame{state: t.rangeName()}
	var parent *rangeFrame
	if len(t.rangeFrames) != 0 {
		parent = t.rangeFrames[len(t.rangeFrames)-1]
		f.next = parent.next
	} else {
		f.next = t.rangeName()
	}
	t.rangeFrames = append(t.rangeFrames, f)
	defer func() { t.rangeFrames = t.rangeFrames[:len(t.rangeFrames)-1] }()

	// Assignment ranges bind new parameters, then assign the original lvalues
	// in parallel. Definition ranges use parameters as their per-iteration vars.
	keys := []ast.Expr{loop.Key, loop.Value}
	params := &ast.FieldList{}
	var lhs, rhs []ast.Expr
	for i := 0; i < yield.Params().Len(); i++ {
		id := ast.NewIdent("_")
		if i < len(keys) && keys[i] != nil {
			if loop.Tok == token.DEFINE {
				id = keys[i].(*ast.Ident)
			} else {
				id = ast.NewIdent(t.rangeName())
				lhs = append(lhs, keys[i])
				rhs = append(rhs, ast.NewIdent(id.Name))
			}
		}
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{id}, Type: t.typeToAST(yield.Params().At(i).Type())})
	}
	t.rewriteRangeBody(&loop.Body.List, f, 0, 0)
	if t.err != nil {
		return loop
	}
	t.translateExpr(&loop.X)
	t.translateBlockStmt(loop.Body)
	body := []ast.Stmt{
		rangeIf(rangeCompare(f.state, token.NEQ, rfReady), t.rangePanic(ast.NewIdent(f.state))),
		rangeAssign(f.state, rangeInt(rfPanic), token.ASSIGN),
	}
	if len(lhs) != 0 {
		assignment := &ast.AssignStmt{Lhs: lhs, Tok: token.ASSIGN, Rhs: rhs}
		t.translateExprList(assignment.Lhs)
		body = append(body, assignment)
	}
	body = append(body, loop.Body.List...)
	body = append(body, f.yieldReturn(true)...)
	callback := &ast.FuncLit{Type: &ast.FuncType{Params: params, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}, Body: &ast.BlockStmt{List: body}}
	list := []ast.Stmt{}
	if parent == nil {
		list = append(list, rangeAssign(f.next, rangeInt(0), token.DEFINE))
	}
	list = append(list,
		rangeAssign(f.state, rangeInt(rfReady), token.DEFINE),
		&ast.ExprStmt{X: &ast.CallExpr{Fun: loop.X, Args: []ast.Expr{callback}}},
		rangeIf(rangeCompare(f.state, token.EQL, rfPanic), t.rangePanic(rangeInt(rfMissingPanic))),
		rangeAssign(f.state, rangeInt(rfExhausted), token.ASSIGN),
	)
	if parent == nil {
		list = append(list, rangeIf(rangeCompare(f.next, token.NEQ, 0), &ast.ReturnStmt{}))
	} else {
		list = append(list, rangeIf(rangeCompare(f.next, token.NEQ, 0), parent.yieldReturn(false)...))
	}
	return &ast.BlockStmt{List: list}
}

func (t *translator) rewriteRangeBody(list *[]ast.Stmt, f *rangeFrame, breaks, continues int) {
	// Apply slice intrinsics before classifying nested range control flow.
	t.lowerStoredStringSequences(&ast.BlockStmt{List: *list})
	for i := range *list {
		s := (*list)[i]
		recurse := func(block *ast.BlockStmt, b, c int) {
			if block != nil {
				t.rewriteRangeBody(&block.List, f, b, c)
			}
		}
		switch s := s.(type) {
		case *ast.ReturnStmt:
			body := []ast.Stmt{}
			if len(s.Results) != 0 {
				lhs := []ast.Expr{}
				for _, name := range t.rangeFunction.results {
					lhs = append(lhs, ast.NewIdent(name))
				}
				body = append(body, &ast.AssignStmt{Lhs: lhs, Tok: token.ASSIGN, Rhs: s.Results})
			}
			body = append(body, rangeAssign(f.next, rangeInt(-1), token.ASSIGN))
			body = append(body, f.yieldReturn(false)...)
			(*list)[i] = &ast.BlockStmt{List: body}
		case *ast.BranchStmt:
			if s.Label != nil || s.Tok == token.GOTO {
				t.err = fmt.Errorf("%s: labeled transfer in function range is not yet supported by gccgo compatibility", t.fset.Position(s.Pos()))
				return
			}
			if s.Tok == token.BREAK && breaks == 0 {
				(*list)[i] = &ast.BlockStmt{List: f.yieldReturn(false)}
			}
			if s.Tok == token.CONTINUE && continues == 0 {
				(*list)[i] = &ast.BlockStmt{List: f.yieldReturn(true)}
			}
		case *ast.DeferStmt:
			t.err = fmt.Errorf("%s: defer in function range needs gccgo enclosing-frame support", t.fset.Position(s.Pos()))
			return
		case *ast.BlockStmt:
			recurse(s, breaks, continues)
		case *ast.IfStmt:
			recurse(s.Body, breaks, continues)
			if s.Else != nil {
				tail := []ast.Stmt{s.Else}
				t.rewriteRangeBody(&tail, f, breaks, continues)
				s.Else = tail[0]
			}
		case *ast.ForStmt:
			recurse(s.Body, breaks+1, continues+1)
		case *ast.RangeStmt:
			if t.lowerStringSequenceRange(s) || t.functionRangeSignature(s) == nil {
				recurse(s.Body, breaks+1, continues+1)
			}
		case *ast.SwitchStmt:
			recurse(s.Body, breaks+1, continues)
		case *ast.TypeSwitchStmt:
			recurse(s.Body, breaks+1, continues)
		case *ast.SelectStmt:
			recurse(s.Body, breaks+1, continues)
		case *ast.CaseClause:
			t.rewriteRangeBody(&s.Body, f, breaks, continues)
		case *ast.CommClause:
			t.rewriteRangeBody(&s.Body, f, breaks, continues)
		case *ast.LabeledStmt:
			t.err = fmt.Errorf("%s: label in function range is not yet supported by gccgo compatibility", t.fset.Position(s.Pos()))
			return
		}
	}
}
