// Controlled tool-call peer. It tests the original CLI's tools and persistence,
// while the separate Qwen fixture proves actual model inference.
package main

import (
	"encoding/json"
	"fmt"
	"kos"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:18083")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	models := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"native-tools","object":"model","type":"llm","state":"loaded","loaded_context_length":32768}]}`)
	}
	mux.HandleFunc("/v1/models", models)
	mux.HandleFunc("/api/v0/models", models)
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Model    string    `json:"model"`
			Stream   bool      `json:"stream"`
			Messages []message `json:"messages"`
			Tools    []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Model != "native-tools" {
			http.Error(w, "invalid request", 400)
			return
		}
		if !request.Stream {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"native-title","object":"chat.completion","created":1,"model":"native-tools","choices":[{"index":0,"message":{"role":"assistant","content":"Native original tools"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
			return
		}
		var results []string
		for _, msg := range request.Messages {
			if msg.Role == "tool" {
				results = append(results, string(msg.Content))
			}
		}
		count := len(results)
		mcpMode := os.Getenv("KOLIBRI_TOOL_MODE") == "mcp"
		shellMode := os.Getenv("KOLIBRI_TOOL_MODE") == "shell"
		lspMode := os.Getenv("KOLIBRI_TOOL_MODE") == "lsp"
		if lspMode && count == 0 {
			// Coordinate the fixture with the original asynchronous LSP startup.
			deadline := time.Now().Add(30 * time.Second)
			for {
				if _, err := os.Stat("/hd0/1/LSP.READY"); err == nil {
					break
				}
				if time.Now().After(deadline) {
					http.Error(w, "native LSP readiness timeout", 500)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
			time.Sleep(2 * time.Second)
		}
		if mcpMode {
			found := false
			for _, tool := range request.Tools {
				if tool.Function.Name == "native_echo" {
					found = true
				}
			}
			if !found {
				kos.DebugString("OPENCODE_MCP_DISCOVERY_FAILED\n")
				http.Error(w, "native MCP tool was not discovered", 500)
				return
			}
			if count > 0 && !strings.Contains(results[count-1], "Привет 😀 from original MCP") {
				kos.DebugString("OPENCODE_MCP_RESULT_FAILED " + results[count-1] + "\n")
				http.Error(w, "unexpected MCP response", 500)
				return
			}
		}
		if count > 0 {
			if !lspMode && (strings.Contains(results[count-1], "error") || strings.Contains(results[count-1], "Error")) {
				kos.DebugString("OPENCODE_TOOL_PEER_FAILED " + results[count-1] + "\n")
				http.Error(w, "original tool returned error", 500)
				return
			}
			expected := ""
			switch count {
			case 1:
				expected = "Привет"
			case 3:
				expected = "Original write PASS"
			case 5:
				expected = "Original edit PASS"
			}
			if shellMode {
				switch count {
				case 1:
					expected = "first=Привет"
				case 2:
					expected = "second=Привет"
					if !strings.Contains(results[count-1], "Exit code 7") || !strings.Contains(results[count-1], "shell stderr") {
						http.Error(w, "native shell status/stderr lost", 500)
						return
					}
				case 3:
					expected = "Привет, KolibriOS!"
				case 4, 5:
					expected = "fixture.txt"
				}
			}
			if lspMode {
				expected = ""
				switch count {
				case 1:
					expected = "if true; then"
				case 2:
					if !strings.Contains(results[1], "mvdan/sh") || !strings.Contains(results[1], "fi") || !strings.Contains(results[1], "Current file: 1 errors") {
						kos.DebugString("OPENCODE_LSP_DIAGNOSTIC_FAILED " + results[1] + "\n")
						http.Error(w, "actual parser diagnostic missing", 500)
						return
					}
				case 3:
					expected = "File successfully written"
				case 4:
					expected = "echo okay"
					if strings.Contains(results[3], "<file_diagnostics>") || strings.Contains(results[3], "Current file: 1 errors") {
						kos.DebugString("OPENCODE_LSP_CLEAR_FAILED " + results[3] + "\n")
						http.Error(w, "diagnostic did not clear after original write tool", 500)
						return
					}
				}
			}
			if !mcpMode && expected != "" && !strings.Contains(results[count-1], expected) {
				kos.DebugString(fmt.Sprintf("OPENCODE_TOOL_PEER_FAILED unexpected tool result %q\n", results[count-1]))
				http.Error(w, "unexpected original tool result", 500)
				return
			}
		}
		var name string
		var args any
		switch count {
		case 0:
			name = "view"
			args = map[string]any{"file_path": "/hd0/1/fixture.txt"}
		case 1:
			name = "write"
			args = map[string]any{"file_path": "/hd0/1/generated.txt", "content": "Original write PASS\n"}
		case 2:
			name = "view"
			args = map[string]any{"file_path": "/hd0/1/generated.txt"}
		case 3:
			name = "edit"
			args = map[string]any{"file_path": "/hd0/1/generated.txt", "old_string": "Original write PASS", "new_string": "Original edit PASS"}
		case 4:
			name = "view"
			args = map[string]any{"file_path": "/hd0/1/generated.txt"}
		}
		if mcpMode {
			name = ""
			if count == 0 {
				name = "native_echo"
				args = map[string]any{"text": "Привет 😀 from original MCP"}
			}
		}
		if shellMode {
			name = ""
			switch count {
			case 0:
				name = "bash"
				args = map[string]any{"command": `shell_value=Привет; printf 'first=%s\n' "$shell_value"`, "timeout": 5000}
			case 1:
				name = "bash"
				args = map[string]any{"command": `printf 'second=%s\n' "$shell_value"; printf 'shell stderr\n' >&2; (exit 7)`, "timeout": 5000}
			case 2:
				name = "grep"
				args = map[string]any{"pattern": "Привет", "path": "/hd0/1", "include": "fixture.txt"}
			case 3:
				name = "glob"
				args = map[string]any{"pattern": "**/*.txt", "path": "/hd0/1"}
			case 4:
				name = "ls"
				args = map[string]any{"path": "/hd0/1"}
			}
		}
		if lspMode {
			name = ""
			switch count {
			case 0, 1, 3:
				// Original CoderAgentTools registers diagnostics only if LSP was
				// ready at construction, before its asynchronous init normally ends.
				// The always-present view/write tools use the live client map.
				if count == 1 {
					time.Sleep(time.Second)
				}
				name = "view"
				args = map[string]any{"file_path": "/hd0/1/fixture.sh"}
			case 2:
				name = "write"
				args = map[string]any{"file_path": "/hd0/1/fixture.sh", "content": "echo Привет\nif true; then\n echo okay\nfi\n"}
			}
		}
		delta := map[string]any{"role": "assistant"}
		finish := "stop"
		if name != "" {
			argumentJSON, _ := json.Marshal(args)
			delta["tool_calls"] = []any{map[string]any{"index": 0, "id": fmt.Sprintf("native-tool-%d", count), "type": "function", "function": map[string]any{"name": name, "arguments": string(argumentJSON)}}}
			finish = "tool_calls"
		} else {
			if lspMode && count == 4 {
				delta["content"] = "KolibriOS original LSP integration PASS"
				kos.DebugString("OPENCODE_ORIGINAL_LSP_RESULTS_PASS\n")
			} else if mcpMode && count == 1 {
				delta["content"] = "KolibriOS original MCP integration PASS"
				kos.DebugString("OPENCODE_ORIGINAL_MCP_RESULTS_PASS\n")
			} else {
				if count != 5 {
					http.Error(w, "unexpected tool sequence", 500)
					return
				}
				if shellMode {
					delta["content"] = "KolibriOS original shell and search integration PASS"
					kos.DebugString("OPENCODE_ORIGINAL_SHELL_TOOLS_PASS\n")
				} else {
					delta["content"] = "KolibriOS original tool integration PASS"
					kos.DebugString("OPENCODE_ORIGINAL_TOOL_RESULTS_PASS\n")
				}
			}
		}
		chunk := func(change any, reason any) {
			data, _ := json.Marshal(map[string]any{"id": "native-tools", "object": "chat.completion.chunk", "created": 1, "model": "native-tools", "choices": []any{map[string]any{"index": 0, "delta": change, "finish_reason": reason}}, "usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}})
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		chunk(delta, nil)
		w.(http.Flusher).Flush()
		chunk(map[string]any{}, finish)
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
		kos.DebugString(fmt.Sprintf("OPENCODE_TOOL_STEP_SENT %d %s\n", count, name))
	})
	if err := os.WriteFile("/hd0/1/SERVICE.READY", []byte("ready"), 0600); err != nil {
		panic(err)
	}
	kos.DebugString("OPENCODE_TOOL_TEST_SERVICE_READY\n")
	if err := http.Serve(listener, mux); err != nil {
		panic(err)
	}
}
