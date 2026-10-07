package go2go

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrivateFunctionBridgeDoesNotShadowCallee(t *testing.T) {
	const source = `package main
func codeVerifier() (codeVerifier string, challenge string, err error) { return "abc", "xyz", nil }
func gccgoArg0(values ...int) (gccgoArg0 int) { for _, v := range values { gccgoArg0 += v }; return }
func noResult() {}
func main() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	imp := NewImporter(t.TempDir())
	config := types.Config{}
	pkg, err := config.Check("main", fset, []*ast.File{file}, imp.info)
	if err != nil {
		t.Fatal(err)
	}
	if err := rewriteAST(fset, imp, "main", pkg, file, true); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Check("main", fset, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	// Exercise forwarding, including multiple results and variadic arguments.
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			body, err := parser.ParseFile(fset, "body.go", `package main; func exercise() {
				v, c, err := GccgoCompat୦codeVerifier()
				if v != "abc" || c != "xyz" || err != nil { panic("results") }
				if GccgoCompat୦gccgoArg0(2, 3, 5) != 10 { panic("variadic") }
				GccgoCompat୦noResult()
			}`, 0)
			if err != nil {
				t.Fatal(err)
			}
			fn.Body = body.Decls[0].(*ast.FuncDecl).Body
		}
	}
	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, file); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "run", path)
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("forwarding failed: %v\n%s", err, result)
	}
}
