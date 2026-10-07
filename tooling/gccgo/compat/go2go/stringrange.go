// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package go2go

import (
	"go/ast"
	"go/token"
	"go/types"
)

// lowerStringSequenceRange recognizes the standard library's immutable
// SplitSeq and SplitAfterSeq iterators. GCC's Go frontend cannot range over
// functions yet. Eager splitting gives the same elements and evaluates the
// arguments once while retaining an actual loop: return, labeled branches,
// and defer still refer to the enclosing function. The tradeoff is an eager
// allocation of the slice. Arbitrary iterator functions are not rewritten.
func (t *translator) lowerStringSequenceRange(s *ast.RangeStmt) bool {
	call, ok := s.X.(*ast.CallExpr)
	if !ok || s.Value != nil || !t.lowerStringSequenceCall(call, true) {
		return false
	}
	lowerStringSliceRange(s)
	t.markStringSliceRange(s)
	return true
}

func (t *translator) markStringSliceRange(s *ast.RangeStmt) {
	if t.stringSliceRanges == nil {
		t.stringSliceRanges = make(map[*ast.RangeStmt]bool)
	}
	t.stringSliceRanges[s] = true
}

func (t *translator) lowerStringSequenceCall(call *ast.CallExpr, rewrite bool) bool {
	if len(call.Args) != 2 {
		return false
	}
	var id *ast.Ident
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		id = fun.Sel
	case *ast.Ident:
		id = fun
	default:
		return false
	}
	fn, ok := t.importer.info.Uses[id].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "strings" {
		return false
	}
	var replacement string
	switch fn.Name() {
	case "SplitSeq":
		replacement = "Split"
	case "SplitAfterSeq":
		replacement = "SplitAfter"
	default:
		return false
	}
	if !rewrite {
		return true
	}
	// Replace the object as well as its name: subsequent expression rewriting
	// must see the target function's slice result rather than the iterator.
	newID := &ast.Ident{NamePos: id.NamePos, Name: replacement}
	t.importer.info.Uses[newID] = fn.Pkg().Scope().Lookup(replacement)
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		fun.Sel = newID
	case *ast.Ident:
		call.Fun = newID
	}
	return true
}

func lowerStringSliceRange(s *ast.RangeStmt) {
	if s.Key != nil {
		s.Value = s.Key
		s.Key = &ast.Ident{NamePos: s.Key.Pos(), Name: "_"}
	}
}

// An inferred local iterator can use this intrinsic when its only use is the
// immediately following range in the same block. These are single-use iterators:
// reusable variables must keep their stateful function representation. Reject
// gotos that could execute the range twice without reinitializing the variable.
func (t *translator) lowerStoredStringSequences(body *ast.BlockStmt) {
	type candidate struct {
		call    *ast.CallExpr
		loop    *ast.RangeStmt
		invalid bool
	}
	candidates := make(map[types.Object]*candidate)
	add := func(id *ast.Ident, expr ast.Expr, loop *ast.RangeStmt) {
		obj, ok := t.importer.info.Defs[id].(*types.Var)
		call, isCall := expr.(*ast.CallExpr)
		if !ok || !isCall || (obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()) || !t.lowerStringSequenceCall(call, false) {
			return
		}
		operand, ok := loop.X.(*ast.Ident)
		if !ok || loop.Value != nil || t.importer.info.Uses[operand] != obj {
			return
		}
		candidates[obj] = &candidate{call: call, loop: loop}
	}
	for i, statement := range body.List {
		if i+1 == len(body.List) {
			break
		}
		next := body.List[i+1]
		if labeled, ok := next.(*ast.LabeledStmt); ok {
			next = labeled.Stmt
		}
		loop, ok := next.(*ast.RangeStmt)
		if !ok {
			continue
		}
		switch node := statement.(type) {
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE && len(node.Lhs) == len(node.Rhs) {
				for i, expr := range node.Lhs {
					if id, ok := expr.(*ast.Ident); ok {
						add(id, node.Rhs[i], loop)
					}
				}
			}
		case *ast.DeclStmt:
			if declaration, ok := node.Decl.(*ast.GenDecl); ok && declaration.Tok == token.VAR {
				for _, spec := range declaration.Specs {
					value := spec.(*ast.ValueSpec)
					if value.Type == nil && len(value.Names) == len(value.Values) {
						for i, id := range value.Names {
							add(id, value.Values[i], loop)
						}
					}
				}
			}
		}
	}
	if len(candidates) == 0 {
		return
	}
	allowed := make(map[*ast.Ident]bool)
	for _, candidate := range candidates {
		allowed[candidate.loop.X.(*ast.Ident)] = true
	}
	hasGoto := false
	ast.Inspect(body, func(n ast.Node) bool {
		if branch, ok := n.(*ast.BranchStmt); ok && branch.Tok == token.GOTO {
			hasGoto = true
		}
		if id, ok := n.(*ast.Ident); ok && !allowed[id] {
			if candidate := candidates[t.importer.info.Uses[id]]; candidate != nil {
				candidate.invalid = true
			}
		}
		return true
	})
	for _, candidate := range candidates {
		if candidate.invalid || hasGoto {
			continue
		}
		t.lowerStringSequenceCall(candidate.call, true)
		lowerStringSliceRange(candidate.loop)
		t.markStringSliceRange(candidate.loop)
	}
}
