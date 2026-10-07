// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package go2go

import (
	"go/types"
	"strconv"
	"strings"
)

// normalizeAliases preserves all type identity (including struct tags),
// while making alias spellings immaterial to specialization names.
func normalizeAliases(t types.Type) types.Type {
	t = types.Unalias(t)
	switch t := t.(type) {
	case *types.Array:
		return types.NewArray(normalizeAliases(t.Elem()), t.Len())
	case *types.Slice:
		return types.NewSlice(normalizeAliases(t.Elem()))
	case *types.Pointer:
		return types.NewPointer(normalizeAliases(t.Elem()))
	case *types.Map:
		return types.NewMap(normalizeAliases(t.Key()), normalizeAliases(t.Elem()))
	case *types.Chan:
		return types.NewChan(t.Dir(), normalizeAliases(t.Elem()))
	case *types.Struct:
		fields := make([]*types.Var, t.NumFields())
		tags := make([]string, len(fields))
		for i := range fields {
			f := t.Field(i)
			fields[i] = types.NewField(f.Pos(), f.Pkg(), f.Name(), normalizeAliases(f.Type()), f.Embedded())
			tags[i] = t.Tag(i)
		}
		return types.NewStruct(fields, tags)
	case *types.Tuple:
		if t == nil {
			return t
		}
		vars := make([]*types.Var, t.Len())
		for i := range vars {
			v := t.At(i)
			vars[i] = types.NewVar(v.Pos(), v.Pkg(), v.Name(), normalizeAliases(v.Type()))
		}
		return types.NewTuple(vars...)
	case *types.Signature:
		return types.NewSignatureType(t.Recv(), nil, nil,
			normalizeAliases(t.Params()).(*types.Tuple), normalizeAliases(t.Results()).(*types.Tuple), t.Variadic())
	case *types.Interface:
		t.Complete()
		methods := make([]*types.Func, t.NumMethods())
		for i := range methods {
			m := t.Method(i)
			methods[i] = types.NewFunc(m.Pos(), m.Pkg(), m.Name(), normalizeAliases(m.Type()).(*types.Signature))
		}
		return types.NewInterfaceType(methods, nil).Complete()
	case *types.Named:
		if t.TypeArgs().Len() != 0 {
			args := make([]types.Type, t.TypeArgs().Len())
			for i := range args {
				args[i] = normalizeAliases(t.TypeArgs().At(i))
			}
			instance, err := types.Instantiate(nil, t.Origin(), args, false)
			if err != nil {
				panic(err)
			}
			return instance
		}
	}
	return t
}

// TypeString alone omits the package identity of private anonymous fields
// and methods. Those owners are part of Go type identity, just as tags are.
func (imp *Importer) typeIdentity(t types.Type) string {
	t = types.Unalias(t)
	switch t := t.(type) {
	case *types.Array:
		return "[" + strconv.FormatInt(t.Len(), 10) + "]" + imp.typeIdentity(t.Elem())
	case *types.Slice:
		return "[]" + imp.typeIdentity(t.Elem())
	case *types.Pointer:
		return "*" + imp.typeIdentity(t.Elem())
	case *types.Map:
		return "map[" + imp.typeIdentity(t.Key()) + "]" + imp.typeIdentity(t.Elem())
	case *types.Chan:
		return "chan(" + strconv.Itoa(int(t.Dir())) + ")" + imp.typeIdentity(t.Elem())
	case *types.Named:
		name := t.Obj().Name()
		if localName := imp.localTypes[t.Obj()]; localName != "" {
			name = localName
		}
		if t.Obj().Pkg() != nil {
			name = t.Obj().Pkg().Path() + "." + name
		}
		if t.TypeArgs().Len() != 0 {
			args := make([]string, t.TypeArgs().Len())
			for i := range args {
				args[i] = imp.typeIdentity(t.TypeArgs().At(i))
			}
			name += "[" + strings.Join(args, ",") + "]"
		}
		return name
	case *types.Tuple:
		args := make([]string, t.Len())
		for i := range args {
			args[i] = imp.typeIdentity(t.At(i).Type())
		}
		return "(" + strings.Join(args, ",") + ")"
	case *types.Signature:
		return "func" + imp.typeIdentity(t.Params()) + imp.typeIdentity(t.Results()) + strconv.FormatBool(t.Variadic())
	case *types.Struct:
		fields := make([]string, t.NumFields())
		for i := range fields {
			f := t.Field(i)
			name := f.Name()
			if !f.Exported() && f.Pkg() != nil {
				name = f.Pkg().Path() + "." + name
			}
			fields[i] = name + ":" + imp.typeIdentity(f.Type()) + ":" + strconv.FormatBool(f.Embedded()) + ":" + strconv.Quote(t.Tag(i))
		}
		return "struct{" + strings.Join(fields, ";") + "}"
	case *types.Interface:
		t.Complete()
		methods := make([]string, t.NumMethods())
		for i := range methods {
			m := t.Method(i)
			name := m.Name()
			if !m.Exported() && m.Pkg() != nil {
				name = m.Pkg().Path() + "." + name
			}
			methods[i] = name + ":" + imp.typeIdentity(m.Type())
		}
		return "interface{" + strings.Join(methods, ";") + "}"
	}
	return types.TypeString(t, func(p *types.Package) string { return p.Path() })
}
