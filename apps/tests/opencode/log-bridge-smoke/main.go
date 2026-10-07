package main

import (
	"bytes"
	"encoding/json"
	"kos"
	"log"
	"log/slog"
	"runtime"
	"strings"
	"sync"
)

var initialLog = func() string {
	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetFlags(0)
	slog.Warn("before main", "step", 1)
	text := output.String()
	log.SetOutput(nil)
	return text
}()

func main() {
	console, ok := kos.OpenConsole("OpenCode upstream log and slog")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	if initialLog != "WARN before main step=1\n" {
		panic("slog initialization bridge")
	}
	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetFlags(0)
	log.Printf("plain %d", 2)
	slog.Info("structured", "number", 3)
	if output.String() != "plain 2\nINFO structured number=3\n" {
		panic("default log output bridge")
	}
	output.Reset()
	l := log.New(&output, "prefix ", log.Lshortfile|log.Lmsgprefix)
	l.Print("source")
	if !strings.Contains(output.String(), "main.go:") || !strings.HasSuffix(output.String(), ": prefix source\n") {
		panic("upstream source and prefix formatting")
	}
	pc, _, _, found := runtime.Caller(0)
	if !found || runtime.FuncForPC(pc) == nil {
		panic("native caller metadata")
	}
	output.Reset()
	l.SetFlags(0)
	l.SetPrefix("")
	var workers sync.WaitGroup
	runtime.GOMAXPROCS(2)
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for n := 0; n < 10; n++ {
				l.Printf("worker %d %d", worker, n)
			}
		}(worker)
	}
	workers.Wait()
	if strings.Count(output.String(), "\n") != 40 {
		panic("concurrent log records")
	}
	output.Reset()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	log.Print("reverse bridge")
	var record map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &record); err != nil || record["msg"] != "reverse bridge" || record["level"] != "INFO" {
		panic("log to slog reverse bridge")
	}
	console.WriteString("Original Go log/slog initialization, output bridges, callers and concurrent writes PASS\n")
	kos.DebugString("OPENCODE_LOG_BRIDGE_PASS")
}
