package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MagicalTux/goro/core/phpctx"
	"github.com/MagicalTux/goro/core/phpv"
	_ "github.com/MagicalTux/goro/ext/standard"
	"gopkg.in/readline.v1"
	"kos"
)

const consoleTitle = "Goro PHP Console"

func main() {
	console, ok := kos.OpenConsole(consoleTitle)
	if !ok {
		kos.DebugString("goro: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / ready")
	}

	p := phpctx.NewProcess("cli")
	if err := p.CommandLine(os.Args); err != nil {
		_, _ = fmt.Printf("goro: %v\n", err)
		os.Exit(1)
		return
	}

	ctx := phpctx.NewGlobal(context.Background(), p)

	if len(os.Args) < 2 {
		if err := runREPL(ctx); err != nil {
			_, _ = fmt.Printf("goro: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := ctx.RunFile(os.Args[1]); err != nil {
		_, _ = fmt.Printf("goro: %v\n", err)
		os.Exit(1)
		return
	}
}

func runREPL(ctx *phpctx.Global) error {
	_, _ = fmt.Printf(
		"Goro PHP Console (KolibriOS)\n" +
			"Enter PHP code and press Enter.\n" +
			"Press Ctrl+C, Esc, or Ctrl+D to exit.\n",
	)

	rl, err := readline.NewEx(&readline.Config{Prompt: "php> "})
	if err != nil {
		return err
	}
	defer rl.Close()
	output := &replOutputWriter{w: rl.Stdout()}
	ctx.SetOutput(output)

	evalFn, err := ctx.GetFunction(ctx, phpv.ZString("eval"))
	if err != nil {
		return err
	}

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			_, _ = fmt.Printf("\n")
			return nil
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" || line == ".exit" {
			return nil
		}

		ctx.SetDeadline(time.Now().Add(30 * time.Second))
		output.reset()
		result, err := evalLine(ctx, evalFn, normalizeREPLLine(line))
		if err != nil {
			if _, ok := err.(*phpv.PhpExit); ok {
				return nil
			}
			output.finishLine()
			_, _ = fmt.Printf("Error: %v\n", err)
			if hint := replHint(line, err); hint != "" {
				_, _ = fmt.Printf("Hint: %s\n", hint)
			}
			rl.Refresh()
			continue
		}
		ctx.Flush()
		output.finishLine()
		if result != nil && result.GetType() != phpv.ZtNull {
			_, _ = fmt.Printf("%s\n", result.String())
		}
		rl.Refresh()
	}
}

func evalLine(ctx *phpctx.Global, evalFn phpv.Callable, line string) (result *phpv.ZVal, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return ctx.CallZVal(ctx, evalFn, []*phpv.ZVal{phpv.ZString(line).ZVal()}, nil)
}

func normalizeREPLLine(line string) string {
	switch {
	case strings.HasSuffix(line, ";"):
		return line
	case strings.HasSuffix(line, "{"):
		return line
	case strings.HasSuffix(line, "}"):
		return line
	case strings.HasSuffix(line, ":"):
		return line
	default:
		return line + ";"
	}
}

func replHint(line string, err error) string {
	if err == nil {
		return ""
	}
	if (strings.Contains(err.Error(), "write context") || strings.Contains(err.Error(), "not writable")) &&
		strings.Contains(line, "=") && !strings.Contains(line, "$") {
		return "PHP variables must start with $, for example $x = []"
	}
	return ""
}

type replOutputWriter struct {
	w                io.Writer
	wrote            bool
	endedWithNewline bool
}

func (w *replOutputWriter) Write(p []byte) (int, error) {
	if len(p) != 0 {
		w.wrote = true
		last := p[len(p)-1]
		w.endedWithNewline = last == '\n' || last == '\r'
	}
	return w.w.Write(p)
}

func (w *replOutputWriter) reset() {
	w.wrote = false
	w.endedWithNewline = false
}

func (w *replOutputWriter) finishLine() {
	if w.wrote && !w.endedWithNewline {
		_, _ = w.w.Write([]byte("\n"))
		w.endedWithNewline = true
	}
}
