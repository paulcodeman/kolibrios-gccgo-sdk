package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"internal/godebug"
	"io"
	"kos"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "blockthreads" {
		runtime.GOMAXPROCS(2)
		runtime.LockOSThread()
		mainID, _ := kos.CurrentThreadID()
		go func() {
			runtime.LockOSThread()
			id, ok := kos.CurrentThreadID()
			if !ok || id == mainID {
				panic("worker identifier")
			}
			if err := os.WriteFile("/hd0/1/worker.pid", []byte(strconv.Itoa(int(id))), 0600); err != nil {
				panic(err)
			}
			for {
				time.Sleep(10 * time.Millisecond)
			}
		}()
		for {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if len(os.Args) > 1 && os.Args[1] == "threadexit" {
		runtime.GOMAXPROCS(2)
		runtime.LockOSThread()
		ready := make(chan struct{})
		go func() {
			runtime.LockOSThread()
			id, ok := kos.CurrentThreadID()
			if !ok {
				panic("worker identifier")
			}
			if err := os.WriteFile("/hd0/1/threadexit-worker.pid", []byte(strconv.Itoa(int(id))), 0600); err != nil {
				panic(err)
			}
			close(ready)
			for {
				time.Sleep(10 * time.Millisecond)
			}
		}()
		<-ready
		// Exercise a native exit that deliberately cannot report Go status.
		kos.ExitRaw()
	}
	if len(os.Args) > 1 && os.Args[1] == "replace" {
		var err error
		os.Stdout, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			panic(err)
		}
		fmt.Fprint(os.Stderr, "original inherited stderr\n")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "echo" {
		if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
			panic(err)
		}
		if _, err := os.Stderr.Write([]byte("stderr Привет\n")); err != nil {
			panic(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "combined" {
		fmt.Fprint(os.Stdout, "stdout1\n")
		fmt.Fprint(os.Stderr, "stderr2\n")
		if err := os.Stderr.Close(); err != nil {
			panic(err)
		}
		fmt.Fprint(os.Stdout, "stdout3\n")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "child" {
		want := []string{os.Args[0], "child", "Привет", "", "a b", `a\b`, `"quoted"`}
		if !reflect.DeepEqual(os.Args, want) {
			panic("native child arguments")
		}
		if os.Getenv("NATIVE_TEST_ENV") != "Привет" || os.Getenv("NATIVE_PARENT_ONLY") != "" {
			panic("native child environment")
		}
		dir, err := os.Getwd()
		if err != nil || dir != "/hd0/1/work" {
			panic("native child directory")
		}
		kos.DebugString("OPENCODE_PROCESS_CHILD_ATTRIBUTES_PASS\n")
		os.Exit(7)
	}
	if len(os.Args) > 1 && os.Args[1] == "return" {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "block" {
		for {
			time.Sleep(time.Second)
		}
	}
	console, ok := kos.OpenConsole("Native Go process attributes")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	setting := godebug.New("execerrdot")
	if err := os.Setenv("GODEBUG", "execerrdot=1,execerrdot=0,native-test=1,native-test=2"); err != nil {
		panic(err)
	}
	if setting.Value() != "0" || godebug.Get("native-test") != "2" {
		panic("upstream GODEBUG parser/update")
	}
	setting.IncNonDefault()
	if err := os.Unsetenv("GODEBUG"); err != nil {
		panic(err)
	}
	if setting.Value() != "" || godebug.Get("native-test") != "" {
		panic("upstream GODEBUG unset")
	}
	if err := os.Mkdir("/hd0/1/work", 0700); err != nil {
		panic(err)
	}
	args := []string{os.Args[0], "child", "Привет", "", "a b", `a\b`, `"quoted"`}
	child, err := os.StartProcess(os.Args[0], args, &os.ProcAttr{Dir: "/hd0/1/work", Env: []string{"NATIVE_TEST_ENV=Привет"}})
	if err != nil {
		fmt.Println(err)
		panic("native process start")
	}
	state, err := child.Wait()
	if err != nil || !state.Exited() || state.Success() || state.ExitCode() != 7 || state.Pid() != child.Pid {
		fmt.Println(state, err)
		panic("native child exit 7")
	}
	child, err = os.StartProcess(os.Args[0], []string{os.Args[0], "return"}, nil)
	if err != nil {
		panic("native return process start")
	}
	state, err = child.Wait()
	if err != nil || !state.Success() || state.ExitCode() != 0 {
		fmt.Println(state, err)
		panic("native main return status")
	}
	command := exec.Command(os.Args[0], args[1:]...)
	command.Env = []string{"NATIVE_TEST_ENV=Привет"}
	command.Dir = "/hd0/1/work"
	err = command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 7 || command.ProcessState.ExitCode() != 7 {
		panic("original Cmd exit status")
	}
	command = exec.Command(os.Args[0], "return")
	if err = command.Run(); err != nil || !command.ProcessState.Success() {
		fmt.Println(err)
		panic("original Cmd normal return")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command = exec.CommandContext(ctx, os.Args[0], "block")
	if err = command.Run(); err == nil || command.ProcessState == nil || command.ProcessState.Success() || command.ProcessState.ExitCode() != -1 {
		panic("original Cmd native cancellation")
	}
	kos.DebugString("OPENCODE_UPSTREAM_COMMAND_NATIVE_PASS\n")
	payload := make([]byte, 131073)
	for index := range payload {
		payload[index] = byte(index % 251)
	}
	var output, stderr bytes.Buffer
	command = exec.Command(os.Args[0], "echo")
	command.Stdin = bytes.NewReader(payload)
	command.Stdout, command.Stderr = &output, &stderr
	if err = command.Run(); err != nil || !bytes.Equal(output.Bytes(), payload) || stderr.String() != "stderr Привет\n" {
		fmt.Println("stream sizes", output.Len(), stderr.Len(), err)
		panic("original Cmd buffered streams")
	}
	kos.DebugString("OPENCODE_PROCESS_BUFFERED_STREAMS_PASS\n")
	command = exec.Command(os.Args[0], "combined")
	combined, err := command.CombinedOutput()
	if err != nil || string(combined) != "stdout1\nstderr2\nstdout3\n" {
		fmt.Println(string(combined), err)
		panic("original Cmd CombinedOutput/shared close")
	}
	kos.DebugString("OPENCODE_PROCESS_COMBINED_STREAMS_PASS\n")
	command = exec.Command(os.Args[0], "replace")
	combined, err = command.CombinedOutput()
	if err != nil || string(combined) != "original inherited stderr\n" {
		panic("reassigned stdio must close original inherited streams")
	}
	command = exec.Command(os.Args[0], "echo")
	stdin, err := command.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr.Reset()
	command.Stderr = &stderr
	if err = command.Start(); err != nil {
		panic(err)
	}
	written := make(chan error, 1)
	go func() { _, err := stdin.Write(payload); _ = stdin.Close(); written <- err }()
	result, err := io.ReadAll(stdout)
	if err != nil || !bytes.Equal(result, payload) {
		panic("original Cmd streaming pipes")
	}
	if err = <-written; err != nil {
		panic(err)
	}
	if err = command.Wait(); err != nil || stderr.String() != "stderr Привет\n" {
		panic("original Cmd pipe completion")
	}
	kos.DebugString("OPENCODE_PROCESS_INTERACTIVE_STREAMS_PASS\n")
	file, err := os.Create("/hd0/1/process-output.txt")
	if err != nil {
		panic(err)
	}
	command = exec.Command(os.Args[0], "echo")
	command.Stdin = bytes.NewReader(payload)
	command.Stdout = file
	if err = command.Run(); err != nil {
		panic(err)
	}
	if _, err = file.Write([]byte("tail")); err != nil {
		panic("caller file ownership")
	}
	_ = file.Close()
	result, err = os.ReadFile("/hd0/1/process-output.txt")
	if err != nil || !bytes.Equal(result, append(payload, []byte("tail")...)) {
		panic("native regular file redirection")
	}
	file, err = os.Create("/hd0/1/closed-process-output.txt")
	if err != nil {
		panic(err)
	}
	command = exec.Command(os.Args[0], "echo")
	command.Stdin = bytes.NewReader(payload)
	command.Stdout = file
	if err = command.Start(); err != nil {
		panic(err)
	}
	if err = file.Close(); err != nil {
		panic(err)
	}
	if err = command.Wait(); err != nil {
		panic(err)
	}
	result, err = os.ReadFile("/hd0/1/closed-process-output.txt")
	if err != nil || !bytes.Equal(result, payload) {
		panic("native inherited file survives caller close")
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	command = exec.CommandContext(ctx2, os.Args[0], "block")
	stdin, err = command.StdinPipe()
	if err != nil {
		panic(err)
	}
	command.Stdout = &output
	if err = command.Run(); err == nil {
		panic("pipe cancellation")
	}
	_ = stdin.Close()
	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	command = exec.CommandContext(ctx3, os.Args[0], "blockthreads")
	if err = command.Run(); err == nil {
		panic("multithreaded cancellation")
	}
	worker, err := os.ReadFile("/hd0/1/worker.pid")
	if err != nil {
		panic("multithreaded fixture did not start")
	}
	workerID, err := strconv.Atoi(string(worker))
	if err != nil {
		panic(err)
	}
	if kos.ThreadSlotByIdentifier(workerID) != 0 {
		// Clean up the reproduced leak even when testing the old backend.
		kos.TerminateByIdentifier(workerID)
		kos.DebugString("OPENCODE_PROCESS_WORKER_KILL_FAILED\n")
		panic("Process.Kill left a native Go worker alive")
	}
	kos.DebugString("OPENCODE_PROCESS_WORKER_KILL_PASS\n")
	child, err = os.StartProcess(os.Args[0], []string{os.Args[0], "threadexit"}, nil)
	if err != nil {
		panic(err)
	}
	state, err = child.Wait()
	if err == nil || state == nil || state.ExitCode() != -1 || state.Success() {
		panic("native unreported status must remain explicit")
	}
	worker, err = os.ReadFile("/hd0/1/threadexit-worker.pid")
	if err != nil {
		panic(err)
	}
	workerID, err = strconv.Atoi(string(worker))
	if err != nil || kos.ThreadSlotByIdentifier(workerID) != 0 {
		panic("native thread exit left a runtime worker")
	}
	kos.DebugString("OPENCODE_PROCESS_UNREPORTED_EXIT_PASS\n")
	child, err = os.StartProcess(os.Args[0], []string{os.Args[0], "return"}, nil)
	if err != nil {
		panic(err)
	}
	pid := child.Pid
	if err = child.Release(); err != nil || child.Pid != -1 {
		panic("native process release")
	}
	for kos.ThreadSlotByIdentifier(pid) != 0 {
		time.Sleep(10 * time.Millisecond)
	}
	reapDeadline := time.Now().Add(3 * time.Second)
	for {
		entries, err := os.ReadDir("/hd0/1")
		if err != nil {
			panic(err)
		}
		remaining := false
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "kos-process-") {
				remaining = true
			}
		}
		if !remaining {
			break
		}
		if time.Now().After(reapDeadline) {
			panic("released process left startup storage")
		}
		time.Sleep(10 * time.Millisecond)
	}
	kos.DebugString("OPENCODE_PROCESS_RELEASE_REAPER_PASS\n")
	kos.DebugString("OPENCODE_PROCESS_STREAMS_NATIVE_PASS\n")
	fmt.Println("Native arguments, environment, cwd, exit 7 and main return 0 PASS")
	kos.DebugString("OPENCODE_PROCESS_NATIVE_PASS\n")
}
