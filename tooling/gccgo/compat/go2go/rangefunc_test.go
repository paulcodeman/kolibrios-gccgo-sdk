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
	"runtime"
	"strings"
	"testing"
)

const rangeTestHelper = `
type rangeError string
func (e rangeError) RuntimeError() {}
func (e rangeError) Error() string { return "runtime error: " + string(e) }
func panicState(s int) {
 switch s {
 case 0: panic(rangeError("range function continued iteration after function for loop body returned false"))
 case 2: panic(rangeError("range function continued iteration after loop body panic"))
 case 3: panic(rangeError("range function continued iteration after whole loop exit"))
 case 4: panic(rangeError("range function recovered a loop body panic and did not resume panicking"))
 }
 panic("unexpected state")
}
`

const rangeTestSource = `package main
import "strings"
type seq func(func(int) bool)
type seq2 func(func(int,string) bool)
var trace string
func pair(y func(int,string)bool) { defer func(){trace += "P"}(); for i:=1;i<=4;i++ { if !y(i,"a") {return} } }
func many(y func(int)bool) { for i:=1;i<=4;i++ { if !y(i) { return } } }
func answer() (n int) {
 defer func(){n++;trace += "D"}()
 for i:=range seq(many) { for j:=range seq(many) { if i==2&&j==3 { n:=99;_ = n;return i*10+j } } }
 return 0
}
func bare() (n int) { for i:=range seq(many) { n=i; if i==3 { return } };return }
func tuples() (int,string) { for i,s:=range seq2(pair) { if i==2 {return i,s} };return 0,"" }
func mixed() string { for range seq(many) { for s:=range strings.SplitSeq("a,b",",") { return s } };return "" }
func storedMixed() string { for range seq(many) { split:=strings.SplitSeq("a,b",",");for s:=range split { return s } };return "" }
func storedSingle() string { split:=strings.SplitSeq("a,b",",");for s:=range split { return s };return "" }
func returned() { defer func(){trace += "R"}();for i:=range seq2(pair) { if i==2 {return} };panic("return lost") }
func brokenIterator(y func(int)bool) { y(1); y(2) }
func swallowed(y func(int)bool) { defer func(){recover()}();y(1) }
func resumed(y func(int)bool) { func(){defer func(){recover()}();y(1)}();y(2) }
func hygiene() {
 for false:=range seq(many) { _=false;break }
 for true:=range seq(many) { if true<3 {continue};break }
 int:=7;_ = int
 for range func(y func()bool){y()} { true,false:=0,7;_,_=true,false;break }
}
func getPanic(f func()) (text string) {
 defer func(){e,ok:=recover().(interface{Error()string;RuntimeError()});if !ok {panic("runtime.Error expected")};text=e.Error()}()
 f();panic("panic missing")
}
func main() {
 hygiene()
 var capture []func()int
 sum:=0
 for i,s:=range seq2(pair) { if s!="a" {panic("second value")};if i==2 {continue};capture=append(capture,func()int{return i});sum+=i;if i==3 {break} }
 if sum!=4||capture[0]()!=1||capture[1]()!=3||trace!="P" {panic("break/continue/capture")}
 var i int;var s string
 for i,s=range seq2(pair) { if i==2 {break} }
 if i!=2||s!="a" {panic("assignment range")}
 hits:=0
 for i:=range seq(many) { switch i {case 1:break;case 2:continue};for j:=0;j<4;j++ {if j==1 {continue};if j==2 {break};hits++} }
 if hits!=3 {panic("nested ordinary control flow")}
 if answer()!=24||bare()!=3 {panic("nested return")}
 if mixed()!="a"||storedMixed()!="a"||storedSingle()!="a" {panic("mixed slice intrinsic return")}
 n,s2:=tuples();if n!=2||s2!="a" {panic("tuple return")}
 returned()
 count:=0
 for range func(y func()bool){y();y()} {count++}
 if count!=2 {panic("zero values")}
 evaluations:=0
 for range func()seq{evaluations++;return many}() {break}
 if evaluations!=1 {panic("iterator evaluated twice")}
 stopped:=getPanic(func(){for range brokenIterator {break}})
 if stopped!="runtime error: range function continued iteration after function for loop body returned false" {panic(stopped)}
 missing:=getPanic(func(){for range swallowed {panic("body")}})
 if missing!="runtime error: range function recovered a loop body panic and did not resume panicking" {panic(missing)}
 resumedPanic:=getPanic(func(){for range resumed {panic("body")}})
 if resumedPanic!="runtime error: range function continued iteration after loop body panic" {panic(resumedPanic)}
 var saved func(int)bool
 for range func(y func(int)bool){saved=y} {}
 exhausted:=getPanic(func(){saved(1)})
 if exhausted!="runtime error: range function continued iteration after whole loop exit" {panic(exhausted)}
}
`

func rangeTestTranslate(t *testing.T, source string) (*ast.File, *token.FileSet, error) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	imp := NewImporter(t.TempDir())
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("main", fset, []*ast.File{file}, imp.info)
	if err != nil {
		t.Fatal(err)
	}
	// Use the copied upstream helper locally in host execution. Target builds
	// exercise the real SDK runtime entrypoint separately.
	imp.packages["runtime"] = types.NewPackage("runtime", "runtime")
	imp.translated["runtime"] = "test-runtime"
	tr := &translator{fset: fset, importer: imp, tpkg: pkg, types: make(map[ast.Expr]types.Type), typePackages: make(map[*types.Package]bool)}
	tr.reserveRangeNames(file)
	tr.translate(file)
	for _, spec := range file.Imports {
		spec.Name = ast.NewIdent(compatImportName(strings.Trim(spec.Path.Value, "\"")))
	}
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "GccgoCompatPanicRangeState" {
				call.Fun = ast.NewIdent("panicState")
			}
		}
		return true
	})
	return file, fset, tr.err
}

func TestFunctionRangeUpstreamControlFlowAndPanics(t *testing.T) {
	source := rangeTestSource + rangeTestHelper
	file, fset, err := rangeTestTranslate(t, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&types.Config{Importer: importer.Default()}).Check("main", fset, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := printer.Fprint(&data, fset, file); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string][]byte{"upstream": []byte(source), "lowered": data.Bytes()} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.go")
			if err := os.WriteFile(path, contents, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "run", path)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s\n%s", err, output, contents)
			}
		})
	}
}

func TestFunctionRangeRejectsUnsupportedFrameTransfers(t *testing.T) {
	for _, body := range []string{"defer func(){}()", "break L", "goto End"} {
		source := "package main;func seq(y func(int)bool){y(1)};func main(){L:for i:=0;i<1;i++{for range seq{" + body + "}};End:;_ = 0}" + rangeTestHelper
		// Avoid unused labels in the upstream type checker.
		if body != "break L" {
			source = strings.ReplaceAll(source, "L:", "")
		}
		if body != "goto End" {
			source = strings.ReplaceAll(source, "End:", "")
		}
		_, _, err := rangeTestTranslate(t, source)
		if err == nil {
			t.Fatalf("unsupported transfer accepted: %s", body)
		}
	}
}
