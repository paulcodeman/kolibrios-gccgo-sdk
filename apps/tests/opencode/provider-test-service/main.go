// A controlled native HTTP peer for the unchanged CLI integration test.
// Its fixed reply validates the provider/stream/DB path, not actual inference.
package main

import (
	"encoding/json"
	"fmt"
	"kos"
	"net"
	"net/http"
	"os"
)

const reply = "KolibriOS original CLI integration PASS"

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:18080")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	modelList := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"object":"list","data":[{"id":"native-test","object":"model","type":"llm","state":"loaded","loaded_context_length":8192}]}`)
		kos.DebugString("OPENCODE_TEST_MODELS_REQUEST\n")
	}
	mux.HandleFunc("/api/v0/models", modelList)
	mux.HandleFunc("/v1/models", modelList)
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Model    string            `json:"model"`
			Stream   bool              `json:"stream"`
			Messages []json.RawMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Model != "native-test" || len(request.Messages) == 0 {
			http.Error(w, "invalid native CLI request", http.StatusBadRequest)
			return
		}
		kos.DebugString("OPENCODE_TEST_COMPLETION_REQUEST\n")
		if !request.Stream {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"native-reply","object":"chat.completion","created":1,"model":"native-test","choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":7,"total_tokens":8}}`, reply)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprintf(w, "data: {\"id\":\"native-reply\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"native-test\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":%q},\"finish_reason\":null}]}\n\n", reply)
		w.(http.Flusher).Flush()
		fmt.Fprint(w, "data: {\"id\":\"native-reply\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"native-test\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":7,\"total_tokens\":8}}\n\ndata: [DONE]\n\n")
		w.(http.Flusher).Flush()
		kos.DebugString("OPENCODE_TEST_STREAM_SENT\n")
	})
	if err := os.WriteFile("/hd0/1/SERVICE.READY", []byte("ready"), 0600); err != nil {
		panic(err)
	}
	kos.DebugString("OPENCODE_TEST_SERVICE_READY\n")
	if err := http.Serve(listener, mux); err != nil {
		panic(err)
	}
}
