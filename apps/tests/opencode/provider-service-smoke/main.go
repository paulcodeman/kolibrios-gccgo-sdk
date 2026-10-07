package main

import (
	"bufio"
	"encoding/json"
	"io"
	"kos"
	"net/http"
	"strings"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode native provider test peer")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	res, err := http.Get("http://127.0.0.1:18080/api/v0/models")
	if err != nil {
		panic(err)
	}
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&models); err != nil {
		panic(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || len(models.Data) != 1 || models.Data[0].ID != "native-test" {
		panic("model discovery")
	}
	res, err = http.Post("http://127.0.0.1:18080/v1/chat/completions", "application/json", strings.NewReader(`{"model":"native-test","stream":true,"messages":[{"role":"user","content":"native test"}]}`))
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(res.Body)
	sawContent, sawDone := false, false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "KolibriOS original CLI integration PASS") {
			sawContent = true
		}
		if line == "data: [DONE]" {
			sawDone = true
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || !sawContent || !sawDone {
		panic("native peer SSE")
	}
	res, err = http.Post("http://127.0.0.1:18080/v1/chat/completions", "application/json", strings.NewReader(`{"model":"native-test","stream":false,"messages":[{"role":"user","content":"title test"}]}`))
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil || res.StatusCode != 200 || !strings.Contains(string(data), "KolibriOS original CLI integration PASS") {
		panic("native peer ordinary completion")
	}
	console.WriteString("OpenCode native provider peer discovery and HTTP/SSE PASS\n")
	kos.DebugString("OPENCODE_PROVIDER_SERVICE_PASS")
}
