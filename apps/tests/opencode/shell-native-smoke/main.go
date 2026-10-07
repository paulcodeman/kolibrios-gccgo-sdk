package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"kos"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "emit" {
		fmt.Fprint(os.Stdout, "native external Привет\n")
		fmt.Fprint(os.Stderr, "native external stderr\n")
		os.Exit(7)
	}
	console, ok := kos.OpenConsole("Native upstream shell")
	if !ok {
		panic("console")
	}
	defer console.Exit(false)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/hd0/1/gosh.kex", "-c", `value=Привет; for n in 1 2 3; do printf '%s:%s\n' "$value" "$n"; done; test "$(printf abc)" = abc; printf pipe | read result; false; printf 'status=%s\n' "$?"`)
	result, err := command.CombinedOutput()
	if err != nil || string(result) != "Привет:1\nПривет:2\nПривет:3\nstatus=1\n" {
		fmt.Println(string(result), err)
		panic("upstream shell syntax/builtins")
	}
	command = exec.CommandContext(ctx, "/hd0/1/gosh.kex", "-c", "/hd0/1/child.kex emit; printf 'external-status=%s\\n' \"$?\"")
	result, err = command.CombinedOutput()
	if err != nil || string(result) != "native external Привет\nnative external stderr\nexternal-status=7\n" {
		fmt.Println(string(result), err)
		panic("native shell external process/status")
	}
	// Keep stdin open while waiting for each status file, exactly as the
	// unchanged OpenCode PersistentShell does. Buffered whole-file parsing
	// would deadlock here instead of completing the first command.
	command = exec.CommandContext(ctx, "/hd0/1/gosh.kex", "-l")
	stdin, err := command.StdinPipe()
	if err != nil {
		panic(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err = command.Start(); err != nil {
		panic(err)
	}
	scripts := []string{
		"eval 'value=Привет; printf \"%s\\n\" \"$value\"' < /dev/null > /hd0/1/shell-out1 2> /hd0/1/shell-err1\nEXEC_EXIT_CODE=$?\npwd > /hd0/1/shell-cwd1\necho $EXEC_EXIT_CODE > /hd0/1/shell-status1\n",
		"eval 'printf \"%s\\n\" \"$value\"; false' < /dev/null > /hd0/1/shell-out2 2> /hd0/1/shell-err2\nEXEC_EXIT_CODE=$?\npwd > /hd0/1/shell-cwd2\necho $EXEC_EXIT_CODE > /hd0/1/shell-status2\n",
	}
	for index, script := range scripts {
		if _, err = io.WriteString(stdin, script); err != nil {
			panic(err)
		}
		status := fmt.Sprintf("/hd0/1/shell-status%d", index+1)
		for {
			if ctx.Err() != nil {
				fmt.Println(stderr.String())
				panic("persistent shell timeout")
			}
			data, err := os.ReadFile(status)
			if err == nil && len(data) > 0 {
				if string(data) != fmt.Sprintf("%d\n", index) {
					fmt.Println(string(data))
					panic("persistent status")
				}
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		data, err := os.ReadFile(fmt.Sprintf("/hd0/1/shell-out%d", index+1))
		if err != nil || string(data) != "Привет\n" {
			fmt.Println(string(data), err)
			panic("persistent variables/eval/redirection")
		}
	}
	_, err = io.WriteString(stdin, "exit\n")
	if err != nil {
		panic(err)
	}
	_ = stdin.Close()
	if err = command.Wait(); err != nil {
		fmt.Println(stderr.String(), err)
		panic("shell exit")
	}
	fmt.Println("Upstream shell loops, substitutions, pipelines and original persistent protocol PASS")
	kos.DebugString("OPENCODE_SHELL_NATIVE_PASS\n")
}
