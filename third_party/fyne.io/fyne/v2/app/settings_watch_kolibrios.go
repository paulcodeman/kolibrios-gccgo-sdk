//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package app

func watchFile(_ string, _ func()) {
}

func (s *settings) watchSettings() {
	watchTheme()
}

func (s *settings) stopWatching() {
}
