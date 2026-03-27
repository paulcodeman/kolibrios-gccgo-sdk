package kos

const (
	redrawBegin = 1
	redrawEnd   = 2
)

const (
	textFlagUTF8     = 0x30
	textFlagToBuffer = 0x08
)

const (
	WindowStyleSkinnedFixed uint32 = 0x14FFFFFF
	WindowStyleBorderless   uint32 = 0x01000000
)

func BeginRedraw() {
	Redraw(redrawBegin)
}

func EndRedraw() {
	Redraw(redrawEnd)
}

func OpenWindow(x int, y int, width int, height int, title string) {
	OpenWindowStyle(x, y, width, height, WindowStyleSkinnedFixed, title)
}

func OpenWindowStyle(x int, y int, width int, height int, style uint32, title string) {
	WindowWithStyle(x, y, width, height, style, title)
}

func SetWindowTitle(title string) {
	SetCaption(title)
}

func SetWindowTitleWithEncodingPrefix(encoding StringEncoding, title string) {
	if encoding == EncodingDefault {
		SetCaption(title)
		return
	}

	SetCaptionWithPrefix(encoding, title)
}

func DrawText(x int, y int, color Color, text string) {
	flagsColor := (uint32(textFlagUTF8) << 24) | uint32(color)
	writeTextWithLength(x, y, flagsColor, text, nil)
}

func DrawTextBuffer(x int, y int, color Color, text string, buffer *byte) {
	flagsColor := (uint32(textFlagUTF8|textFlagToBuffer) << 24) | uint32(color)
	writeTextWithLength(x, y, flagsColor, text, buffer)
}

func FillRect(x int, y int, width int, height int, color Color) {
	DrawBar(x, y, width, height, uint32(color))
}

func StrokeLine(x1 int, y1 int, x2 int, y2 int, color Color) {
	DrawLine(x1, y1, x2, y2, uint32(color))
}

func DrawButton(x int, y int, width int, height int, id ButtonID, color Color) {
	CreateButton(x, y, width, height, int(id), uint32(color))
}

func PutImage32(buffer *byte, width int, height int, x int, y int) {
	PutPaletteImage(buffer, width, height, x, y, 32, nil, 0)
}
