package kos

func Exit() {
	closeActiveConsole(true)
	ExitRaw()
}
