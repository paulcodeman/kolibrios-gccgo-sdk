package kos

const DefaultSkinPath = "/DEFAULT.SKN"

type SkinMargins struct {
	Left   int
	Right  int
	Top    int
	Bottom int
}

func ScreenSize() (width int, height int) {
	packed := GetScreenSize()
	return int(packed>>16) + 1, int(packed&0xFFFF) + 1
}

func ScreenWorkingArea() Rect {
	var vertical uint32

	horizontal := GetScreenWorkingArea(&vertical)
	return Rect{
		Left:   int(horizontal >> 16),
		Top:    int(vertical >> 16),
		Right:  int(horizontal & 0xFFFF),
		Bottom: int(vertical & 0xFFFF),
	}
}

func SkinHeight() int {
	return GetSkinHeight()
}

func WindowSkinMargins() SkinMargins {
	var vertical uint32

	horizontal := GetSkinMarginsRaw(&vertical)
	return SkinMargins{
		Left:   int(horizontal >> 16),
		Right:  int(horizontal & 0xFFFF),
		Top:    int(vertical >> 16),
		Bottom: int(vertical & 0xFFFF),
	}
}

func SetSystemSkin(path string) FileSystemStatus {
	return SetSystemSkinWithEncoding(path, EncodingUTF8)
}

func SetSystemSkinWithEncoding(path string, encoding StringEncoding) FileSystemStatus {
	return FileSystemStatus(SetSkinWithEncoding(encoding, path))
}

func SetSystemSkinLegacy(path string) FileSystemStatus {
	return FileSystemStatus(SetSkin(path))
}
