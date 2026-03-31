//go:build kolibrios
// +build kolibrios

package fyne

func callerLocation(int) (string, int, bool) {
	return "", 0, false
}
