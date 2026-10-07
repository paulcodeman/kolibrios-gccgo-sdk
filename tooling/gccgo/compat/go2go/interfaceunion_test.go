package go2go

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestUniversalInterfaceUnionRetainsRuntimeMethodSet(t *testing.T) {
	const source = `package p
type publicKey interface{}
type verificationKey interface { publicKey | []byte }
type verificationSet struct { Keys []verificationKey }
type methods interface { any | []byte; M() int }
type implementation int
func (implementation) M() int { return 1 }
var _ verificationKey = 42
var _ methods = implementation(0)
type constraint interface { ~int | string }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	config := types.Config{GoVersion: "go1.26"}
	original, err := config.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	tr := &translator{importer: &Importer{info: info}}
	ast.Inspect(file, func(n ast.Node) bool {
		if iface, ok := n.(*ast.InterfaceType); ok {
			tr.lowerUniversalInterfaceUnions(iface)
		}
		return true
	})
	rewritten, err := config.Check("p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"verificationKey", "methods", "constraint"} {
		before := original.Scope().Lookup(name).Type().Underlying().(*types.Interface)
		after := rewritten.Scope().Lookup(name).Type().Underlying().(*types.Interface)
		if before.IsMethodSet() != after.IsMethodSet() || before.NumMethods() != after.NumMethods() {
			t.Fatalf("changed the runtime method set of %s", name)
		}
	}
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			spec := gen.Specs[0].(*ast.TypeSpec)
			if spec.Name.Name == "verificationKey" && len(spec.Type.(*ast.InterfaceType).Methods.List) != 0 {
				t.Fatal("universal key union was retained")
			}
			if spec.Name.Name == "constraint" {
				if _, ok := spec.Type.(*ast.InterfaceType).Methods.List[0].Type.(*ast.BinaryExpr); !ok {
					t.Fatal("restricted constraint was erased")
				}
			}
		}
	}
}
