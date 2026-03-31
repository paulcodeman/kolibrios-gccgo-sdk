//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kolibrios

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"kos"
)

func currentModifiers() fyne.KeyModifier {
	controls := kos.ControlKeysStatus()
	var modifier fyne.KeyModifier
	if controls.Shift() {
		modifier |= fyne.KeyModifierShift
	}
	if controls.Ctrl() {
		modifier |= fyne.KeyModifierControl
	}
	if controls.Alt() {
		modifier |= fyne.KeyModifierAlt
	}
	if controls&(kos.ControlWinLeft|kos.ControlWinRight) != 0 {
		modifier |= fyne.KeyModifierSuper
	}
	return modifier
}

func convertKey(key kos.KeyEvent) (localized fyne.KeyName, ascii fyne.KeyName, printable rune, hasPrintable bool) {
	localized = keyNameFromCode(key.Code)
	ascii = keyNameFromScanCode(key.ScanCode)
	if localized == fyne.KeyUnknown {
		localized = keyNameFromScanCode(key.ScanCode)
	}
	if key.Code >= 32 && key.Code != 127 {
		printable = rune(key.Code)
		hasPrintable = true
	}
	return
}

func keyNameFromCode(code byte) fyne.KeyName {
	switch code {
	case 0:
		return fyne.KeyUnknown
	case 8:
		return fyne.KeyBackspace
	case 9:
		return fyne.KeyTab
	case 13:
		return fyne.KeyReturn
	case 27:
		return fyne.KeyEscape
	case 32:
		return fyne.KeySpace
	case 127:
		return fyne.KeyDelete
	}
	switch {
	case code >= 'a' && code <= 'z':
		return fyne.KeyName(string([]byte{code - ('a' - 'A')}))
	case code >= 'A' && code <= 'Z':
		return fyne.KeyName(string([]byte{code}))
	case code >= '0' && code <= '9':
		return fyne.KeyName(string([]byte{code}))
	}
	return fyne.KeyUnknown
}

func keyNameFromScanCode(scan byte) fyne.KeyName {
	switch scan {
	case 0x01:
		return fyne.KeyEscape
	case 0x0E:
		return fyne.KeyBackspace
	case 0x0F:
		return fyne.KeyTab
	case 0x1C:
		return fyne.KeyReturn
	case 0x39:
		return fyne.KeySpace
	case 0x47:
		return fyne.KeyHome
	case 0x48:
		return fyne.KeyUp
	case 0x49:
		return fyne.KeyPageUp
	case 0x4B:
		return fyne.KeyLeft
	case 0x4D:
		return fyne.KeyRight
	case 0x4F:
		return fyne.KeyEnd
	case 0x50:
		return fyne.KeyDown
	case 0x51:
		return fyne.KeyPageDown
	case 0x52:
		return fyne.KeyInsert
	case 0x53:
		return fyne.KeyDelete
	case 0x3B:
		return fyne.KeyF1
	case 0x3C:
		return fyne.KeyF2
	case 0x3D:
		return fyne.KeyF3
	case 0x3E:
		return fyne.KeyF4
	case 0x3F:
		return fyne.KeyF5
	case 0x40:
		return fyne.KeyF6
	case 0x41:
		return fyne.KeyF7
	case 0x42:
		return fyne.KeyF8
	case 0x43:
		return fyne.KeyF9
	case 0x44:
		return fyne.KeyF10
	case 0x57:
		return fyne.KeyF11
	case 0x58:
		return fyne.KeyF12
	case 0x02:
		return fyne.Key1
	case 0x03:
		return fyne.Key2
	case 0x04:
		return fyne.Key3
	case 0x05:
		return fyne.Key4
	case 0x06:
		return fyne.Key5
	case 0x07:
		return fyne.Key6
	case 0x08:
		return fyne.Key7
	case 0x09:
		return fyne.Key8
	case 0x0A:
		return fyne.Key9
	case 0x0B:
		return fyne.Key0
	case 0x10:
		return fyne.KeyQ
	case 0x11:
		return fyne.KeyW
	case 0x12:
		return fyne.KeyE
	case 0x13:
		return fyne.KeyR
	case 0x14:
		return fyne.KeyT
	case 0x15:
		return fyne.KeyY
	case 0x16:
		return fyne.KeyU
	case 0x17:
		return fyne.KeyI
	case 0x18:
		return fyne.KeyO
	case 0x19:
		return fyne.KeyP
	case 0x1E:
		return fyne.KeyA
	case 0x1F:
		return fyne.KeyS
	case 0x20:
		return fyne.KeyD
	case 0x21:
		return fyne.KeyF
	case 0x22:
		return fyne.KeyG
	case 0x23:
		return fyne.KeyH
	case 0x24:
		return fyne.KeyJ
	case 0x25:
		return fyne.KeyK
	case 0x26:
		return fyne.KeyL
	case 0x2C:
		return fyne.KeyZ
	case 0x2D:
		return fyne.KeyX
	case 0x2E:
		return fyne.KeyC
	case 0x2F:
		return fyne.KeyV
	case 0x30:
		return fyne.KeyB
	case 0x31:
		return fyne.KeyN
	case 0x32:
		return fyne.KeyM
	case 0x2A:
		return desktop.KeyShiftLeft
	case 0x36:
		return desktop.KeyShiftRight
	case 0x1D:
		return desktop.KeyControlLeft
	case 0x38:
		return desktop.KeyAltLeft
	case 0x5B:
		return desktop.KeySuperLeft
	case 0x5C:
		return desktop.KeySuperRight
	}
	return fyne.KeyUnknown
}

func isKeyModifier(keyName fyne.KeyName) bool {
	return keyName == desktop.KeyShiftLeft || keyName == desktop.KeyShiftRight ||
		keyName == desktop.KeyControlLeft || keyName == desktop.KeyControlRight ||
		keyName == desktop.KeyAltLeft || keyName == desktop.KeyAltRight ||
		keyName == desktop.KeySuperLeft || keyName == desktop.KeySuperRight
}
