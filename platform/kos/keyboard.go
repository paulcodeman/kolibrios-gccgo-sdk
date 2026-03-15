package kos

type KeyboardLayoutKind int
type KeyboardLanguage int
type KeyboardLayoutTable [128]byte

const (
	KeyboardLayoutNormal KeyboardLayoutKind = 1
	KeyboardLayoutShift  KeyboardLayoutKind = 2
	KeyboardLayoutAlt    KeyboardLayoutKind = 3
)

const (
	KeyboardLanguageEnglish    KeyboardLanguage = 1
	KeyboardLanguageFinnish    KeyboardLanguage = 2
	KeyboardLanguageGerman     KeyboardLanguage = 3
	KeyboardLanguageRussian    KeyboardLanguage = 4
	KeyboardLanguageFrench     KeyboardLanguage = 5
	KeyboardLanguageEstonian   KeyboardLanguage = 6
	KeyboardLanguageUkrainian  KeyboardLanguage = 7
	KeyboardLanguageItalian    KeyboardLanguage = 8
	KeyboardLanguageBelarusian KeyboardLanguage = 9
	KeyboardLanguageSpanish    KeyboardLanguage = 10
	KeyboardLanguageCatalan    KeyboardLanguage = 11
)

type KeyEvent struct {
	Raw       int
	Empty     bool
	Hotkey    bool
	Code      byte
	ScanCode  byte
	Modifiers uint16
}

type ControlKeys uint32

const (
	ControlShiftLeft  ControlKeys = 1 << 0
	ControlShiftRight ControlKeys = 1 << 1
	ControlCtrlLeft   ControlKeys = 1 << 2
	ControlCtrlRight  ControlKeys = 1 << 3
	ControlAltLeft    ControlKeys = 1 << 4
	ControlAltRight   ControlKeys = 1 << 5
	ControlCapsLock   ControlKeys = 1 << 6
	ControlNumLock    ControlKeys = 1 << 7
	ControlScrollLock ControlKeys = 1 << 8
	ControlWinLeft    ControlKeys = 1 << 9
	ControlWinRight   ControlKeys = 1 << 10
)

func ReadKey() KeyEvent {
	raw := GetKey()
	value := uint32(raw)
	event := KeyEvent{
		Raw: raw,
	}

	if value == 1 {
		event.Empty = true
		return event
	}

	if byte(value) == 2 {
		event.Hotkey = true
		event.ScanCode = byte(value >> 8)
		event.Modifiers = uint16(value >> 16)
		return event
	}

	event.Code = byte(value >> 8)
	event.ScanCode = byte(value >> 16)
	return event
}

func ControlKeysStatus() ControlKeys {
	return ControlKeys(GetControlKeysRaw())
}

func (keys ControlKeys) Shift() bool {
	return keys&(ControlShiftLeft|ControlShiftRight) != 0
}

func (keys ControlKeys) Ctrl() bool {
	return keys&(ControlCtrlLeft|ControlCtrlRight) != 0
}

func (keys ControlKeys) Alt() bool {
	return keys&(ControlAltLeft|ControlAltRight) != 0
}

func ReadKeyboardLayoutTable(kind KeyboardLayoutKind) (KeyboardLayoutTable, bool) {
	var table KeyboardLayoutTable

	if !isValidKeyboardLayoutKind(kind) {
		return table, false
	}

	return table, GetKeyboardLayoutRaw(int(kind), &table[0]) != -1
}

func SetKeyboardLayoutTable(kind KeyboardLayoutKind, table *KeyboardLayoutTable) bool {
	if table == nil || !isValidKeyboardLayoutKind(kind) {
		return false
	}

	return SetKeyboardLayoutRaw(int(kind), &table[0]) == 0
}

func KeyboardLayoutLanguage() KeyboardLanguage {
	return KeyboardLanguage(GetKeyboardLanguageRaw())
}

func SetKeyboardLayoutLanguage(language KeyboardLanguage) bool {
	if !isValidKeyboardLanguage(language) {
		return false
	}

	return SetKeyboardLanguageRaw(int(language)) == 0
}

func SystemLanguage() KeyboardLanguage {
	return KeyboardLanguage(GetSystemLanguageRaw())
}

func SetSystemLanguage(language KeyboardLanguage) bool {
	if !isValidKeyboardLanguage(language) {
		return false
	}

	return SetSystemLanguageRaw(int(language)) == 0
}

func isValidKeyboardLayoutKind(kind KeyboardLayoutKind) bool {
	switch kind {
	case KeyboardLayoutNormal, KeyboardLayoutShift, KeyboardLayoutAlt:
		return true
	}

	return false
}

func isValidKeyboardLanguage(language KeyboardLanguage) bool {
	return language >= KeyboardLanguageEnglish && language <= KeyboardLanguageCatalan
}
