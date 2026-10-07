package go2go

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"sort"
)

// parseCompilerFile preserves GCC's native symbol annotations while checking
// the surrounding original Go program with the upstream parser/type checker.
// The scanner distinguishes annotations from examples in comments and strings.
func (imp *Importer) parseCompilerFile(fset *token.FileSet, filename string, data []byte) (*ast.File, error) {
	if data == nil {
		var err error
		data, err = os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	}
	masked := append([]byte(nil), data...)
	type annotation struct {
		offset int
		value  string
	}
	var annotations []annotation
	positions := token.NewFileSet()
	scanFile := positions.AddFile(filename, -1, len(data))
	var scan scanner.Scanner
	scan.Init(scanFile, data, nil, 0)
	for {
		position, tok, literal := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.IDENT || literal != "__asm__" {
			continue
		}
		_, paren, _ := scan.Scan()
		_, quoted, symbol := scan.Scan()
		end, close, _ := scan.Scan()
		if paren != token.LPAREN || quoted != token.STRING || close != token.RPAREN {
			return nil, fmt.Errorf("%s: invalid GCC native symbol annotation", positions.Position(position))
		}
		start, limit := scanFile.Offset(position), scanFile.Offset(end)+1
		annotations = append(annotations, annotation{start, symbol})
		for index := start; index < limit; index++ {
			if masked[index] != '\n' && masked[index] != '\r' {
				masked[index] = ' '
			}
		}
	}
	file, err := parser.ParseFile(fset, filename, masked, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	aliases := make(map[string]string)
	for _, annotation := range annotations {
		found := false
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body != nil {
				continue
			}
			end := fset.File(function.End()).Offset(function.End())
			if end <= annotation.offset && len(bytes.TrimSpace(data[end:annotation.offset])) == 0 {
				aliases[function.Name.Name] = annotation.value
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%s: native symbol annotation must follow an external function signature", filename)
		}
	}
	if imp.nativeAsm == nil {
		imp.nativeAsm = make(map[*ast.File]map[string]string)
	}
	imp.nativeAsm[file] = aliases
	return file, nil
}

func printCompilerFile(output io.Writer, fset *token.FileSet, file *ast.File, aliases map[string]string) error {
	if len(aliases) == 0 {
		return config.Fprint(output, fset, file)
	}
	var buffer bytes.Buffer
	if err := config.Fprint(&buffer, fset, file); err != nil {
		return err
	}
	data := buffer.Bytes()
	positions := token.NewFileSet()
	parsed, err := parser.ParseFile(positions, "generated.go", data, parser.ParseComments)
	if err != nil {
		return err
	}
	type insertion struct {
		offset int
		value  string
	}
	var insertions []insertion
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body != nil {
			continue
		}
		if symbol, ok := aliases[function.Name.Name]; ok {
			insertions = append(insertions, insertion{positions.File(function.End()).Offset(function.End()), " __asm__(" + symbol + ")"})
		}
	}
	if len(insertions) != len(aliases) {
		return fmt.Errorf("compiler lowering lost a native symbol annotation")
	}
	sort.Slice(insertions, func(i, j int) bool { return insertions[i].offset < insertions[j].offset })
	previous := 0
	for _, insertion := range insertions {
		if _, err := output.Write(data[previous:insertion.offset]); err != nil {
			return err
		}
		if _, err := io.WriteString(output, insertion.value); err != nil {
			return err
		}
		previous = insertion.offset
	}
	_, err = output.Write(data[previous:])
	return err
}
