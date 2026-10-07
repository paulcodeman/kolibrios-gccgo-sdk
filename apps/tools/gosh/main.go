// Native launcher for mvdan's upstream Bash/POSIX parser and interpreter.
// Configure OpenCode's shell.path to this executable. This is not GNU Bash.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func main() {
	runner, err := interp.New(interp.StdIO(os.Stdin, os.Stdout, os.Stderr))
	if err != nil {
		fail(err)
	}
	arguments := os.Args[1:]
	login := false
	for len(arguments) > 0 && (arguments[0] == "-l" || arguments[0] == "--login") {
		login = true
		arguments = arguments[1:]
	}
	if login {
		for _, name := range []string{"/etc/profile", os.Getenv("HOME") + "/.profile"} {
			if file, err := os.Open(name); err == nil {
				program, err := syntax.NewParser().Parse(file, name)
				_ = file.Close()
				if err == nil {
					err = runner.Run(context.Background(), program)
				}
				if err != nil {
					fail(err)
				}
			}
		}
	}
	if len(arguments) > 0 {
		var reader io.Reader
		name := ""
		if arguments[0] == "-c" {
			if len(arguments) < 2 {
				fail(fmt.Errorf("-c requires a command"))
			}
			reader = strings.NewReader(arguments[1])
			arguments = arguments[2:]
		} else {
			name = arguments[0]
			arguments = arguments[1:]
			file, err := os.Open(name)
			if err != nil {
				fail(err)
			}
			defer file.Close()
			reader = file
		}
		if err := interp.Params(arguments...)(runner); err != nil {
			fail(err)
		}
		program, err := syntax.NewParser().Parse(reader, name)
		if err == nil {
			err = runner.Run(context.Background(), program)
		}
		if err != nil {
			fail(err)
		}
		return
	}
	// Incremental parsing is essential: OpenCode keeps stdin open and waits
	// for each command's status file before submitting its next command.
	parser := syntax.NewParser()
	var runErr error
	err = parser.Interactive(os.Stdin, func(statements []*syntax.Stmt) bool {
		if parser.Incomplete() {
			return true
		}
		for _, statement := range statements {
			runErr = runner.Run(context.Background(), statement)
			if runner.Exited() {
				return false
			}
		}
		return true
	})
	if err != nil {
		fail(err)
	}
	if runErr != nil {
		fail(runErr)
	}
}

func fail(err error) {
	if status, ok := interp.IsExitStatus(err); ok {
		os.Exit(int(status))
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
