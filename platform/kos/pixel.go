package kos

func GetPixelColorFromScreen(x int, y int) uint32 {
	if x < 0 || y < 0 {
		return 0
	}

	width, height := ScreenSize()
	if x >= width || y >= height {
		return 0
	}

	return GetPixelColorFromScreenRaw(y*width + x)
}

func PixelColor(x int, y int) Color {
	return Color(GetPixelColorFromScreen(x, y))
}
