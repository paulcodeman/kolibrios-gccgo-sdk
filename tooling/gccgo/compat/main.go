package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"

	"kolibrios/gccgo-compat/embedcfg"
	"kolibrios/gccgo-compat/go2go"
)

type roots []string

func (r *roots) String() string     { return fmt.Sprint([]string(*r)) }
func (r *roots) Set(s string) error { *r = append(*r, s); return nil }

func main() {
	scan := flag.Bool("scan-generics", false, "detect generic declarations without rewriting sources")
	embedOutput := flag.String("embedcfg", "", "resolve original-source go:embed resources into compiler JSON")
	output := flag.String("out", "", "separate directory for lowered compiler inputs")
	packagePath := flag.String("package", "main", "Go package import path")
	var includes roots
	flag.Var(&includes, "gccgo-import-root", "directory containing target .gox and generic compiler metadata")
	flag.Parse()
	if *embedOutput != "" {
		if err := embedcfg.Write(*embedOutput, flag.Args()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *scan {
		found := false
		gccgoAsm := regexp.MustCompile(`__asm__\s*\(\s*"(?:\\.|[^"\\])*"\s*\)`)
		for _, name := range flag.Args() {
			data, err := os.ReadFile(name)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			// GCC's native symbol annotations are outside the standard Go
			// grammar. They do not affect generic declaration detection.
			data = gccgoAsm.ReplaceAllFunc(data, func(match []byte) []byte {
				for i, b := range match {
					if b != '\n' {
						match[i] = ' '
					}
				}
				return match
			})
			f, err := parser.ParseFile(token.NewFileSet(), name, data, 0)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.FuncDecl:
					found = found || n.Type.TypeParams != nil
				case *ast.TypeSpec:
					found = found || n.TypeParams != nil
				}
				return true
			})
		}
		fmt.Println(found)
		return
	}
	if *output == "" || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	imp := go2go.NewImporter(*output)
	if len(includes) != 0 {
		imp.SetGccgoRoots(includes)
	}
	if err := go2go.RewritePackage(imp, *packagePath, *output, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
