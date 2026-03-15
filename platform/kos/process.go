package kos

func Exit() {
	closeActiveConsole(true)
	RuntimeExitProcessRaw()
	ExitRaw()
}
