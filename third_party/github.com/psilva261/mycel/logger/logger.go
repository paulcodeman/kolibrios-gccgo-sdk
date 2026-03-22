package logger

import (
	"fmt"
	stdlog "log"
	"sync"
)

var (
	mu    sync.Mutex
	quiet bool
	Debug bool
)

func SetQuiet() {
	mu.Lock()
	defer mu.Unlock()

	quiet = true
}

func Printf(format string, v ...interface{}) {
	if Debug && !quiet {
		stdlog.Printf(format, v...)
	}
}

func Infof(format string, v ...interface{}) {
	if !quiet {
		stdlog.Printf(format, v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if !quiet {
		stdlog.Printf(format, v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	stdlog.Printf("%s", message)
	panic(message)
}
