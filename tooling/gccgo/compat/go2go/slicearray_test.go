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

func TestSliceToArrayConversionValuesAndPanics(t *testing.T) {
	const source = `package main
type pair [2]int
type empty [0]int
type numbers []int
var evaluations int
func input() numbers { evaluations++; return nil }
func short() (panicked bool) {
 defer func() { panicked = recover() != nil }()
 _ = [2]int(make([]int, 1, 3))
 return
}
func main() {
 s := numbers{3, 5, 7}
 v := pair(s)
 s[0] = 99
 if v != (pair{3, 5}) { panic("value conversion retained slice alias") }
 p := (*pair)(s)
 p[0] = 11
 if s[0] != 11 { panic("pointer conversion lost alias") }
 z := empty(input())
 if z != (empty{}) || evaluations != 1 { panic("nil empty conversion") }
 _ = [0]int([]int{})
 if !short() { panic("conversion tested capacity instead of length") }
}
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
	if err := rewriteAST(fset, imp, "main", pkg, file, false); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Check("main", fset, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := printer.Fprint(&data, fset, file); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "run", path)
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("slice conversion failed: %v\n%s", err, result)
	}
}
