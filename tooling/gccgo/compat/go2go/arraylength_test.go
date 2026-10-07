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

func TestInferredArrayLengthWithForwardConstants(t *testing.T) {
	const first = `package main
var order int
func next() int { order = order*10+1; return order }
var values = [...]int{last: next(), first: next(), 4: next(), next()}
func main() {
 if len(values) != 10 || values[9] != 1 || values[0] != 11 || values[4] != 111 || values[5] != 1111 { panic("values or evaluation order") }
}
`
	const second = `package main; const (first = iota; second; third); const last = 9`
	fset := token.NewFileSet()
	files := make([]*ast.File, 2)
	for i, source := range []string{first, second} {
		file, err := parser.ParseFile(fset, "source.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		files[i] = file
	}
	imp := NewImporter(t.TempDir())
	config := types.Config{}
	pkg, err := config.Check("main", fset, files, imp.info)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if err := rewriteAST(fset, imp, "main", pkg, file, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := config.Check("main", fset, files, nil); err != nil {
		t.Fatal(err)
	}
	var array *ast.ArrayType
	ast.Inspect(files[0], func(n ast.Node) bool {
		if a, ok := n.(*ast.ArrayType); ok {
			array = a
		}
		return true
	})
	if array.Len.(*ast.BasicLit).Value != "10" {
		t.Fatal("incorrect inferred length")
	}
	dir := t.TempDir()
	var paths []string
	for i, file := range files {
		var data bytes.Buffer
		if err := printer.Fprint(&data, fset, file); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, []string{"first.go", "second.go"}[i])
		if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), append([]string{"run"}, paths...)...)
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("array initialization failed: %v\n%s", err, result)
	}
}
