package kos

type CursorHandle uint32

const cursorARGBImageBytes = 32 * 32 * 4

type MouseButtonInfo struct {
	Raw              uint32
	LeftHeld         bool
	RightHeld        bool
	MiddleHeld       bool
	Button4Held      bool
	Button5Held      bool
	LeftPressed      bool
	RightPressed     bool
	MiddlePressed    bool
	LeftReleased     bool
	RightReleased    bool
	MiddleReleased   bool
	VerticalScroll   bool
	HorizontalScroll bool
	LeftDoubleClick  bool
}

const (
	MouseButtonLeftMask   uint32 = 1 << 0
	MouseButtonRightMask  uint32 = 1 << 1
	MouseButtonMiddleMask uint32 = 1 << 2
	MouseButton4Mask      uint32 = 1 << 3
	MouseButton5Mask      uint32 = 1 << 4
)

func MouseScreenPosition() Point {
	return unpackUnsignedPoint(GetMouseScreenPosition())
}

func MouseWindowPosition() Point {
	return unpackSignedPackedPoint(GetMouseWindowPosition())
}

func MouseHeldButtons() MouseButtonInfo {
	return decodeMouseButtonInfo(GetMouseButtonState())
}

func MouseButtons() MouseButtonInfo {
	return decodeMouseButtonInfo(GetMouseButtonEventState())
}

func MouseScrollDelta() Point {
	return unpackSignedPackedPoint(GetMouseScrollData())
}

func SetMousePointerPosition(x int, y int) {
	SetMousePointerPositionRaw(packUnsignedPoint(x, y))
}

func SimulateMouseButtons(mask uint32) {
	SimulateMouseButtonsRaw(mask)
}

func LoadCursorFile(path string) CursorHandle {
	return LoadCursorFileWithEncoding(path, EncodingUTF8)
}

func LoadCursorFileWithEncoding(path string, encoding StringEncoding) CursorHandle {
	return CursorHandle(LoadCursorWithEncoding(encoding, path))
}

func LoadCursorCURData(data []byte) CursorHandle {
	if len(data) == 0 {
		return 0
	}

	return CursorHandle(LoadCursorRaw(byteSliceAddress(data), 1))
}

func LoadCursorARGB(image []byte, hotX int, hotY int) CursorHandle {
	if len(image) < cursorARGBImageBytes {
		return 0
	}

	if hotX < 0 || hotX > 31 || hotY < 0 || hotY > 31 {
		return 0
	}

	descriptor := (uint32(hotX) << 24) | (uint32(hotY) << 16) | 2
	return CursorHandle(LoadCursorRaw(byteSliceAddress(image), descriptor))
}

func SetCursor(handle CursorHandle) CursorHandle {
	return CursorHandle(SetCursorRaw(uint32(handle)))
}

func RestoreDefaultCursor() CursorHandle {
	return SetCursor(0)
}

func DeleteCursor(handle CursorHandle) {
	if handle == 0 {
		return
	}

	DeleteCursorRaw(uint32(handle))
}

func decodeMouseButtonInfo(raw uint32) MouseButtonInfo {
	return MouseButtonInfo{
		Raw:              raw,
		LeftHeld:         raw&(1<<0) != 0,
		RightHeld:        raw&(1<<1) != 0,
		MiddleHeld:       raw&(1<<2) != 0,
		Button4Held:      raw&(1<<3) != 0,
		Button5Held:      raw&(1<<4) != 0,
		LeftPressed:      raw&(1<<8) != 0,
		RightPressed:     raw&(1<<9) != 0,
		MiddlePressed:    raw&(1<<10) != 0,
		VerticalScroll:   raw&(1<<15) != 0,
		LeftReleased:     raw&(1<<16) != 0,
		RightReleased:    raw&(1<<17) != 0,
		MiddleReleased:   raw&(1<<18) != 0,
		HorizontalScroll: raw&(1<<23) != 0,
		LeftDoubleClick:  raw&(1<<24) != 0,
	}
}
