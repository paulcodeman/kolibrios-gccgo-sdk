//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kolibrios

import "fyne.io/fyne/v2"

type device struct{}

var _ fyne.Device = (*device)(nil)

func (*device) Orientation() fyne.DeviceOrientation {
	return fyne.OrientationHorizontalLeft
}

func (*device) IsMobile() bool {
	return false
}

func (*device) IsBrowser() bool {
	return false
}

func (*device) HasKeyboard() bool {
	return true
}

func (*device) SystemScaleForWindow(fyne.Window) float32 {
	return 1
}
