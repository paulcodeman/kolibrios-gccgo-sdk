package main

import (
	"kos"
	"os"
	"runtime"
	"sync"
)

// Provider discovery runs during initialization too. Capture the environment
// before main, not after test setup has had a chance to populate it.
var initializedHome = os.Getenv("HOME")
var initializedEndpoint = os.Getenv("LOCAL_ENDPOINT")
var initializedEmpty, initializedEmptyFound = os.LookupEnv("EMPTY")

func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func main() {
	console, ok := kos.OpenConsole("OpenCode native provider environment")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	require(initializedHome == "/hd0/1", "HOME missing during package initialization")
	require(initializedEndpoint == "http://127.0.0.1:18080/v1", "provider endpoint missing during initialization")
	require(initializedEmpty == "" && initializedEmptyFound, "empty variable confused with absent variable")
	require(os.Getenv("DUP") == "first", "duplicate did not retain first occurrence")
	require(os.Getenv("LITERAL") == " $HOME = ${HOME} ", "sidecar expanded or trimmed literal value")
	require(os.Getenv("UNICODE") == "привет мир", "UTF-8 environment value")
	_, absent := os.LookupEnv("ABSENT")
	require(!absent, "missing variable present")
	require(os.ExpandEnv("${HOME}/$DUP") == "/hd0/1/first", "upstream ExpandEnv")
	require(os.Setenv("", "x") != nil && os.Setenv("BAD=KEY", "x") != nil && os.Setenv("X", "a\x00b") != nil, "invalid Setenv accepted")
	require(os.Unsetenv("DUP") == nil, "Unsetenv")
	_, present := os.LookupEnv("DUP")
	require(!present, "Unsetenv exposed duplicate")
	snapshot := os.Environ()
	require(len(snapshot) > 0, "environment snapshot empty")
	snapshot[0] = "HOME=changed"
	require(os.Getenv("HOME") == "/hd0/1", "snapshot mutated process environment")
	runtime.GOMAXPROCS(2)
	var group sync.WaitGroup
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func(key string) {
			defer group.Done()
			for j := 0; j < 200; j++ {
				require(os.Setenv(key, "value") == nil, "concurrent Setenv")
				value, ok := os.LookupEnv(key)
				require(ok && value == "value", "concurrent LookupEnv")
				require(len(os.Environ()) > 0, "concurrent Environ")
				require(os.Unsetenv(key) == nil, "concurrent Unsetenv")
			}
		}(string(rune('A' + i)))
	}
	group.Wait()
	os.Clearenv()
	require(len(os.Environ()) == 0 && os.Getenv("HOME") == "", "Clearenv")
	require(os.Setenv("AFTER_CLEAR", "ok") == nil && os.Getenv("AFTER_CLEAR") == "ok", "Setenv after Clearenv")
	console.WriteString("OpenCode native pre-init environment and upstream concurrent table PASS\n")
	kos.DebugString("OPENCODE_ENVIRONMENT_PASS")
}
