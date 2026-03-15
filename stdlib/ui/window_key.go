package ui

import "kos"

func (window *Window) handleKey() bool {
	if window == nil {
		return false
	}
	key := kos.ReadKey()
	if key.Empty || key.Hotkey {
		return false
	}
	if key.Code == 9 {
		if kos.ControlKeysStatus().Shift() {
			return window.focusPrev()
		}
		return window.focusNext()
	}
	if window.focused != nil {
		if aware, ok := window.focused.(KeyAware); ok {
			handled := aware.HandleKey(key)
			if handled {
				window.noteDirty(window.focused)
				if element, ok := window.focused.(*Element); ok && element.isTextInput() {
					window.caretBlinkResetAt = kos.UptimeCentiseconds()
				}
			}
			return handled
		}
	}
	return false
}
