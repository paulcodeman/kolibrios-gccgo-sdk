// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package go2go

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// SetGccgoRoots uses actual target export data for ordinary dependencies.
// Generic dependencies retain original ASTs in compiler sidecar records.
func (imp *Importer) SetGccgoRoots(roots []string) {
	imp.gccgoRoots = roots
	imp.defaultImporter = importer.ForCompiler(token.NewFileSet(), "gccgo", func(path string) (io.ReadCloser, error) {
		for _, root := range roots {
			file, err := os.Open(filepath.Join(root, filepath.FromSlash(path)+".gox"))
			if err == nil {
				object, elfErr := elf.NewFile(file)
				if elfErr != nil {
					if _, err := file.Seek(0, io.SeekStart); err != nil {
						file.Close()
						return nil, err
					}
					return file, nil
				}
				section := object.Section(".go_export")
				if section == nil {
					file.Close()
					return nil, fmt.Errorf("no Go export section in %s", file.Name())
				}
				data, err := section.Data()
				file.Close()
				if err != nil {
					return nil, err
				}
				return exportReader{bytes.NewReader(data)}, nil
			}
		}
		return nil, fmt.Errorf("gccgo export data not found for %q", path)
	}).(types.ImporterFrom)
}

type exportReader struct{ *bytes.Reader }

func (exportReader) Close() error { return nil }

func (imp *Importer) importGccgoPackage(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if pkg, ok := imp.packages[path]; ok {
		return pkg, nil
	}
	for _, root := range imp.gccgoRoots {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)+".generics.json"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var record struct {
			Sources      []string
			SourceSHA256 map[string]string `json:"source_sha256"`
		}
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		var files []*ast.File
		var named []namedAST
		for _, filename := range record.Sources {
			data, err := os.ReadFile(filename)
			if err != nil {
				return nil, err
			}
			if expected := record.SourceSHA256[filename]; expected == "" || expected != fmt.Sprintf("%x", sha256.Sum256(data)) {
				return nil, fmt.Errorf("stale generic compiler metadata for %s; rebuild package %s", filename, path)
			}
			file, err := imp.parseCompilerFile(fset, filename, data)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
			named = append(named, namedAST{filename, file})
		}
		conf := types.Config{Importer: imp, Sizes: types.SizesFor("gc", "386"), GoVersion: "go1.24"}
		pkg, err := conf.Check(path, fset, files, imp.info)
		if err != nil {
			return nil, err
		}
		imp.record(pkg.Name(), named, path, pkg, files)
		imp.liftLocalTypes(fset, pkg, files)
		return pkg, nil
	}
	return imp.defaultImporter.ImportFrom(path, dir, mode)
}

type inferred struct {
	Targs []types.Type
	Sig   *types.Signature
}

func typeParams(typ interface{ TypeParams() *types.TypeParamList }) []*types.TypeParam {
	list := typ.TypeParams()
	if list.Len() == 0 {
		return nil
	}
	out := make([]*types.TypeParam, list.Len())
	for i := range out {
		out[i] = list.At(i)
	}
	return out
}

func typeArgsList(typ *types.Named) []types.Type {
	list := typ.TypeArgs()
	out := make([]types.Type, list.Len())
	for i := range out {
		out[i] = list.At(i)
	}
	return out
}

func asPointer(typ types.Type) *types.Pointer {
	p, _ := typ.Underlying().(*types.Pointer)
	return p
}

func asStruct(typ types.Type) *types.Struct {
	s, _ := typ.Underlying().(*types.Struct)
	return s
}

func instanceIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return instanceIdent(e.Fun)
	case *ast.IndexExpr:
		switch e.X.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			return instanceIdent(e.X)
		}
	case *ast.IndexListExpr:
		switch e.X.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			return instanceIdent(e.X)
		}
	case *ast.SelectorExpr:
		return e.Sel
	case *ast.Ident:
		return e
	case *ast.ParenExpr:
		return instanceIdent(e.X)
	}
	return nil
}

func (imp *Importer) recordInstances(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		inst, ok := imp.info.Instances[instanceIdent(expr)]
		if !ok {
			return true
		}
		args := make([]types.Type, inst.TypeArgs.Len())
		for i := range args {
			args[i] = inst.TypeArgs.At(i)
		}
		sig, _ := inst.Type.(*types.Signature)
		imp.inferred[expr] = inferred{args, sig}
		return true
	})
}

func (imp *Importer) copyInferred(t *translator, ta *typeArgs, original, replacement ast.Expr) {
	if old, ok := imp.inferred[original]; ok {
		updated, _ := t.instantiateInferred(ta, old)
		imp.inferred[replacement] = updated
	}
}

// instantiateInferred handles both generic function calls and named type
// conversions. A conversion has type arguments but no function signature.
func (t *translator) instantiateInferred(ta *typeArgs, old inferred) (inferred, bool) {
	var updated inferred
	changed := false
	for _, arg := range old.Targs {
		typ := t.instantiateType(ta, arg)
		updated.Targs = append(updated.Targs, typ)
		changed = changed || typ != arg
	}
	if old.Sig != nil {
		updated.Sig = t.instantiateType(ta, old.Sig).(*types.Signature)
		changed = changed || updated.Sig != old.Sig
	}
	return updated, changed
}

func (t *translator) translateFunctionOrTypeInstantiation(expr *ast.Expr) {
	id := instanceIdent(*expr)
	obj := t.importer.info.Uses[id]
	if obj == nil {
		obj = t.importer.info.ObjectOf(id)
	}
	if obj == nil {
		return
	}
	switch typ := obj.Type().(type) {
	case *types.Signature:
		if len(typeParams(typ)) > 0 {
			t.translateFunctionInstantiation(expr)
		}
	case *types.Named:
		if len(typeParams(typ)) > 0 {
			t.translateTypeInstantiation(expr)
		}
	}
}

// RewritePackage checks original sources and writes lowered compiler inputs
// into a separate output directory. It never overwrites its input files.
func RewritePackage(imp *Importer, packagePath, output string, filenames []string) error {
	fset := token.NewFileSet()
	var files []*ast.File
	var named []namedAST
	sort.Strings(filenames)
	baseCounts := make(map[string]int)
	for _, filename := range filenames {
		baseCounts[filepath.Base(filename)]++
	}
	outputs := make(map[string]string)
	for _, filename := range filenames {
		input, err := filepath.Abs(filename)
		if err != nil {
			return err
		}
		name := filepath.Base(filename)
		if baseCounts[name] > 1 {
			name = fmt.Sprintf("%x_%s", sha256.Sum256([]byte(input)), name)
		}
		destination, err := filepath.Abs(filepath.Join(output, name))
		if err != nil {
			return err
		}
		if input == destination {
			return fmt.Errorf("compiler lowering requires separate input and output directories")
		}
		file, err := imp.parseCompilerFile(fset, input, nil)
		if err != nil {
			return err
		}
		files = append(files, file)
		named = append(named, namedAST{input, file})
		outputs[input] = destination
	}
	if len(files) == 0 {
		return fmt.Errorf("no compiler input files")
	}
	conf := types.Config{Importer: imp, Sizes: types.SizesFor("gc", "386"), GoVersion: "go1.24"}
	pkg, err := conf.Check(packagePath, fset, files, imp.info)
	if err != nil {
		return err
	}
	imp.record(pkg.Name(), named, packagePath, pkg, files)
	imp.liftLocalTypes(fset, pkg, files)
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	for i, file := range named {
		if err := rewriteFile(output, fset, imp, packagePath, pkg, outputs[file.name], file.ast, i == 0); err != nil {
			return err
		}
	}
	data, err := json.Marshal(outputs)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "compiler-inputs.json"), data, 0644)
}
