// Check OpenCode's unchanged native LSP client against real Bash parsing.
// The small server adapter uses upstream mvdan/sh for syntax diagnostics.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kos"
	"os"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/lsp"
	"github.com/opencode-ai/opencode/internal/lsp/protocol"
	"mvdan.cc/sh/v3/syntax"
)

func main() {
	if _, err := config.Load("/hd0/1", false); err != nil {
		panic(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}
	console, ok := kos.OpenConsole("Original OpenCode LSP and native Bash parser")
	if !ok {
		panic("console")
	}
	defer console.Exit(false)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := lsp.NewClient(ctx, os.Args[0], "serve")
	if err != nil {
		panic(err)
	}
	if _, err = client.InitializeLSPClient(ctx, "/hd0/1"); err != nil {
		panic(err)
	}
	if err = client.WaitForServerReady(ctx); err != nil {
		panic(err)
	}
	const filename = "/hd0/1/fixture.sh"
	if err = os.WriteFile(filename, []byte("echo Привет\nif true; then\n"), 0600); err != nil {
		panic(err)
	}
	if err = client.OpenFile(ctx, filename); err != nil {
		panic(err)
	}
	uri := protocol.DocumentUri("file://" + filename)
	for len(client.GetFileDiagnostics(uri)) == 0 {
		if ctx.Err() != nil {
			panic("native LSP diagnostic timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	diagnostics := client.GetFileDiagnostics(uri)
	if len(diagnostics) != 1 || diagnostics[0].Source != "mvdan/sh" || diagnostics[0].Severity != protocol.SeverityError || !strings.Contains(diagnostics[0].Message, "fi") {
		fmt.Println(diagnostics)
		panic("native LSP real parser error")
	}
	if err = os.WriteFile(filename, []byte("echo Привет\nif true; then\n echo okay\nfi\n"), 0600); err != nil {
		panic(err)
	}
	if err = client.NotifyChange(ctx, filename); err != nil {
		panic(err)
	}
	for len(client.GetFileDiagnostics(uri)) != 0 {
		if ctx.Err() != nil {
			panic("native LSP diagnostic clearing timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err = client.Close(); err != nil {
		panic(err)
	}
	fmt.Println("Original LSP initialize, ready, Bash syntax error, change, clear and child shutdown PASS")
	kos.DebugString("OPENCODE_LSP_NATIVE_PASS\n")
}

func serve() {
	reader := bufio.NewReader(os.Stdin)
	for {
		message, err := lsp.ReadMessage(reader)
		if err != nil {
			if strings.Contains(err.Error(), io.EOF.Error()) {
				return
			}
			panic(err)
		}
		var result any
		switch message.Method {
		case "initialize":
			result = map[string]any{"capabilities": map[string]any{"textDocumentSync": map[string]any{"openClose": true, "change": 1}, "workspaceSymbolProvider": true}, "serverInfo": map[string]any{"name": "KolibriOS mvdan Bash syntax", "version": "3.10.0"}}
		case "workspace/symbol", "textDocument/documentSymbol":
			result = []any{}
			if message.Method == "workspace/symbol" {
				if err = os.WriteFile("/hd0/1/LSP.READY", []byte("ready"), 0600); err != nil {
					panic(err)
				}
			}
		case "shutdown":
			result = nil
		case "exit":
			return
		case "textDocument/didOpen":
			var params struct {
				TextDocument struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
				} `json:"textDocument"`
			}
			if err = json.Unmarshal(message.Params, &params); err != nil {
				panic(err)
			}
			publish(params.TextDocument.URI, params.TextDocument.Text)
		case "textDocument/didChange":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				ContentChanges []struct {
					Text string `json:"text"`
				} `json:"contentChanges"`
			}
			if err = json.Unmarshal(message.Params, &params); err != nil {
				panic(err)
			}
			if len(params.ContentChanges) > 0 {
				publish(params.TextDocument.URI, params.ContentChanges[len(params.ContentChanges)-1].Text)
			}
		}
		if message.ID != 0 {
			data, err := json.Marshal(result)
			if err != nil {
				panic(err)
			}
			if err = lsp.WriteMessage(os.Stdout, &lsp.Message{JSONRPC: "2.0", ID: message.ID, Result: data}); err != nil {
				panic(err)
			}
		}
	}
}

func publish(uri, text string) {
	diagnostics := []any{}
	_, err := syntax.NewParser().Parse(strings.NewReader(text), uri)
	kos.DebugString(fmt.Sprintf("OPENCODE_LSP_PUBLISH %s error=%t\n", uri, err != nil))
	if err != nil {
		line, character := uint(0), uint(0)
		if parse, ok := err.(syntax.ParseError); ok {
			line = parse.Pos.Line() - 1
			// Parser columns are byte positions; LSP positions use UTF-16 units.
			lines := strings.Split(text, "\n")
			if int(line) < len(lines) {
				before := lines[line]
				column := int(parse.Pos.Col() - 1)
				if column < len(before) {
					before = before[:column]
				}
				for len(before) > 0 {
					r, size := utf8.DecodeRuneInString(before)
					character++
					if utf16.IsSurrogate(r) || r > 0xffff {
						character++
					}
					before = before[size:]
				}
			}
		}
		position := map[string]uint{"line": line, "character": character}
		diagnostics = append(diagnostics, map[string]any{"range": map[string]any{"start": position, "end": position}, "severity": 1, "source": "mvdan/sh", "message": err.Error()})
	}
	message, err := lsp.NewNotification("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": diagnostics})
	if err != nil {
		panic(err)
	}
	if err = lsp.WriteMessage(os.Stdout, message); err != nil {
		panic(err)
	}
}
