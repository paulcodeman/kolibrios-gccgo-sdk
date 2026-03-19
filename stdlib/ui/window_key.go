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
		if window.focused != nil {
			if aware, ok := window.focused.(TabAware); ok {
				window.noteHandlerMayMutate(window.focused)
				handled := aware.HandleTab(kos.ControlKeysStatus().Shift())
				if handled {
					window.noteDirty(window.focused)
					return true
				}
			}
		}
		if kos.ControlKeysStatus().Shift() {
			return window.focusPrev()
		}
		return window.focusNext()
	}
	if window.focused != nil {
		if aware, ok := window.focused.(KeyAware); ok {
			window.noteHandlerMayMutate(window.focused)
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
