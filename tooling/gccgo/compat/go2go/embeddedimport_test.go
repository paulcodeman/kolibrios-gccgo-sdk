package go2go

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestDotImportedEmbeddedTypeKeepsQualifier(t *testing.T) {
	const source = `package p
import . "io"
type wrapper struct { Reader }
var _ Reader = (*wrapper)(nil)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object), Defs: make(map[*ast.Ident]types.Object)}
	config := types.Config{Importer: importer.Default()}
	pkg, err := config.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	var field *ast.Field
	ast.Inspect(file, func(n ast.Node) bool {
		if st, ok := n.(*ast.StructType); ok {
			field = st.Fields.List[0]
		}
		return true
	})
	id := field.Type.(*ast.Ident)
	if _, ok := info.Defs[id].(*types.Var); !ok {
		t.Fatal("missing embedded field definition")
	}
	if _, ok := info.Uses[id].(*types.TypeName); !ok {
		t.Fatal("missing embedded type use")
	}
	tr := &translator{importer: &Importer{info: info}, tpkg: pkg, typePackages: make(map[*types.Package]bool)}
	tr.translateField(field)
	selector, ok := field.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Reader" || selector.X.(*ast.Ident).Name != compatImportName("io") {
		t.Fatalf("embedded type was not qualified: %#v", field.Type)
	}
	if len(field.Names) != 0 {
		t.Fatal("anonymous field became named")
	}
}
