package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/robertkrimen/otto"
	"kos"
)

const consoleTitle = "KolibriOS JS Console"

func main() {
	console, ok := kos.OpenConsole(consoleTitle)
	if !ok {
		kos.DebugString("js console: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}

	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / ready")
	}

	_, _ = fmt.Println("KolibriOS JavaScript Console (Otto)")
	_, _ = fmt.Println("Enter JavaScript and press Enter.")
	_, _ = fmt.Println("Commands: :help, :exit")

	vm := otto.New()
	_, _ = vm.Run("var print = console.log;")

	reader := bufio.NewReader(os.Stdin)
	for {
		if handleLine(reader, vm) {
			return
		}
	}
}

func bareIdentifier(line string) (string, bool) {
	if line == "" {
		return "", false
	}
	if isReservedWord(line) {
		return "", false
	}
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if i == 0 {
			if !(ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
				return "", false
			}
			continue
		}
		if !(ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
			return "", false
		}
	}
	return line, true
}

func isReservedWord(word string) bool {
	switch word {
	case "break", "case", "catch", "class", "const", "continue", "debugger",
		"default", "delete", "do", "else", "enum", "export", "extends", "false",
		"finally", "for", "function", "if", "import", "in", "instanceof", "let",
		"new", "null", "return", "super", "switch", "this", "throw", "true",
		"try", "typeof", "var", "void", "while", "with", "yield":
		return true
	default:
		return false
	}
}

func readConsoleLine(reader *bufio.Reader) (string, error) {
	buf := make([]byte, 0, 128)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if len(buf) > 0 {
				return string(buf), nil
			}
			return "", err
		}
		if b == '\n' || b == '\r' {
			return string(buf), nil
		}
		if b == 8 || b == 127 {
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			continue
		}
		buf = append(buf, b)
	}
}

func runLine(vm *otto.Otto, line string) (otto.Value, error) {
	var (
		value otto.Value
		err   error
	)
	defer func() {
		if recovered := recover(); recovered != nil {
			switch recovered := recovered.(type) {
			case error:
				err = recovered
			default:
				err = fmt.Errorf("%v", recovered)
			}
			value = otto.Value{}
		}
	}()
	value, err = vm.Run(line)
	return value, err
}

func handleLine(reader *bufio.Reader, vm *otto.Otto) (exit bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_, _ = fmt.Printf("Panic: %v\n", recovered)
			exit = false
		}
	}()

	_, _ = fmt.Print("> ")
	line, err := readConsoleLine(reader)
	if err != nil {
		_, _ = fmt.Printf("Input error: %v\n", err)
		return true
	}
	line = trimConsoleLine(line)
	if line == "" {
		return false
	}
	switch line {
	case ":exit", ".exit":
		return true
	case ":help":
		_, _ = fmt.Println("Type JavaScript code and press Enter.")
		_, _ = fmt.Println("Use :exit to quit.")
		return false
	}

	if name, ok := bareIdentifier(line); ok {
		if !vm.Has(name) {
			_, _ = fmt.Printf("Error: ReferenceError: '%s' is not defined\n", name)
			return false
		}
		value, err := vm.Get(name)
		if err != nil {
			_, _ = fmt.Printf("Error: %s\n", safeErrorString(err))
			return false
		}
		if !value.IsUndefined() {
			if text, err := valueString(value); err != nil {
				_, _ = fmt.Printf("Error: %s\n", safeErrorString(err))
			} else {
				_, _ = fmt.Println(text)
			}
		}
		return false
	}

	value, err := runLine(vm, line)
	if err != nil {
		_, _ = fmt.Printf("Error: %s\n", safeErrorString(err))
		return false
	}
	if !value.IsUndefined() {
		if text, err := valueString(value); err != nil {
			_, _ = fmt.Printf("Error: %s\n", safeErrorString(err))
		} else {
			_, _ = fmt.Println(text)
		}
	}
	return false
}

func valueString(value otto.Value) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while formatting value: %v", recovered)
			text = ""
		}
	}()
	return value.String(), nil
}

func safeErrorString(err error) (text string) {
	if err == nil {
		return ""
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			text = fmt.Sprintf("panic while formatting error: %v", recovered)
		}
	}()
	return err.Error()
}

func trimConsoleLine(line string) string {
	for len(line) > 0 {
		last := line[len(line)-1]
		if last != '\r' && last != '\n' {
			break
		}
		line = line[:len(line)-1]
	}

	return line
}
