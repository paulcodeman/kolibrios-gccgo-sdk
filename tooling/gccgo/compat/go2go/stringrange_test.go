package go2go

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStringSequenceRangeControlFlow(t *testing.T) {
	const source = `package main
import ("fmt"; str "strings")
var evaluations int
func input() string { evaluations++; return "a,b,c," }
func exercise() (out []string) {
 defer func() { out = append(out, "outer-defer") }()
outer:
 for i := 0; i < 3; i++ {
  for s := range str.SplitSeq(input(), ",") {
   defer func(v string) { out = append(out, "defer:"+v) }(s)
   if s == "a" { continue }
   out = append(out, s)
   if s == "b" { continue outer }
   break outer
  }
 }
 for s := range str.SplitAfterSeq("x/y/", "/") { out = append(out, s) }
 once := str.SplitSeq(input(), ",")
 for s := range once { out = append(out, "once:"+s) }
 text := "first,second,"
 stored := str.SplitSeq(text, ",")
 text = "changed"
 for s := range stored { out = append(out, "stored:"+s) }
 for s := range stored { out = append(out, "repeat:"+s) }
 var after = str.SplitAfterSeq("left/right", "/")
 for s := range after { out = append(out, "after:"+s) }
 var assigned string
 for assigned = range str.SplitSeq("аб", "") { out = append(out, assigned) }
 for range str.SplitSeq("", "") { panic("empty iterator") }
 for s := range str.SplitSeq("return,ignored", ",") {
  out = append(out, s)
  return
 }
 return
}
func main() { fmt.Println(exercise(), evaluations) }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object), Defs: make(map[*ast.Ident]types.Object)}
	config := types.Config{Importer: importer.Default(), GoVersion: "go1.26"}
	if _, err := config.Check("main", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	tr := &translator{importer: &Importer{info: info}}
	ast.Inspect(file, func(n ast.Node) bool {
		if block, ok := n.(*ast.BlockStmt); ok {
			tr.lowerStoredStringSequences(block)
		}
		return true
	})
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		if loop, ok := n.(*ast.RangeStmt); ok && tr.lowerStringSequenceRange(loop) {
			count++
		}
		return true
	})
	if count != 5 {
		t.Fatalf("rewrote %d loops, want 5", count)
	}
	var lowered bytes.Buffer
	if err := printer.Fprint(&lowered, fset, file); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Check("main", fset, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	run := func(data []byte) []byte {
		name := filepath.Join(dir, "main.go")
		if err := os.WriteFile(name, data, 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "run", name)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go run: %v\n%s", err, output)
		}
		return output
	}
	original, rewritten := run([]byte(source)), run(lowered.Bytes())
	if !bytes.Equal(original, rewritten) {
		t.Fatalf("original %s != lowered %s", original, rewritten)
	}
}

func TestStringSequenceRangeRejectsOtherIterators(t *testing.T) {
	const source = `package p
func SplitSeq(a,b string) func(func(string)bool) { return nil }
func f() { for v := range SplitSeq("x", ",") { _ = v } }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
	config := types.Config{GoVersion: "go1.26"}
	if _, err := config.Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	tr := &translator{importer: &Importer{info: info}}
	ast.Inspect(file, func(n ast.Node) bool {
		if loop, ok := n.(*ast.RangeStmt); ok && tr.lowerStringSequenceRange(loop) {
			t.Fatal("rewrote a user-defined iterator")
		}
		return true
	})
}

func TestStringSequenceRangeRetainsEscapingAndReassignedLocals(t *testing.T) {
	const source = `package p
import ("strings"; "iter")
func consume(iter.Seq[string]) {}
func f() {
 escaped := strings.SplitSeq("a,b", ",")
 consume(escaped)
 for s := range escaped { _ = s }
 reassigned := strings.SplitSeq("a,b", ",")
 reassigned = strings.SplitSeq("c,d", ",")
 for s := range reassigned { _ = s }
 var explicit iter.Seq[string] = strings.SplitSeq("a,b", ",")
 for s := range explicit { _ = s }
 called := strings.SplitSeq("a,b", ",")
 called(func(string)bool {return false})
 for s := range called { _ = s }
 reused := strings.SplitSeq("a,b", ",")
 for s := range reused { _ = s }
 for s := range reused { _ = s }
 gotoIterator := strings.SplitSeq("a,b", ",")
again:
 for s := range gotoIterator { if s == "a" { goto again } }
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object), Defs: make(map[*ast.Ident]types.Object)}
	config := types.Config{Importer: importer.Default(), GoVersion: "go1.26"}
	if _, err := config.Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	tr := &translator{importer: &Importer{info: info}}
	ast.Inspect(file, func(n ast.Node) bool {
		if block, ok := n.(*ast.BlockStmt); ok {
			tr.lowerStoredStringSequences(block)
		}
		if selector, ok := n.(*ast.SelectorExpr); ok && selector.Sel.Name == "Split" {
			t.Fatal("rewrote an escaping or reassigned iterator")
		}
		return true
	})
}
