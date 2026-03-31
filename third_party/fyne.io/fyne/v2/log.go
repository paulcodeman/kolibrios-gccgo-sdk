package fyne

import (
	"log"
)

// LogError reports an error to the command line with the specified err cause,
// if not nil.
// The function also reports basic information about the code location.
func LogError(reason string, err error) {
	log.Println("Fyne error: ", reason)
	if err != nil {
		log.Println("  Cause:", err)
	}

	file, line, ok := callerLocation(1)
	if ok {
		log.Printf("  At: %s:%d", file, line)
	}
}
