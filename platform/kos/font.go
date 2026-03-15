package kos

// FontSmoothingMode controls system font smoothing.
// Values follow KolibriOS style settings (function 48, subfunctions 9/10).
type FontSmoothingMode uint8

const (
	FontSmoothingOff       FontSmoothingMode = 0
	FontSmoothingAntialias FontSmoothingMode = 1
	FontSmoothingSubpixel  FontSmoothingMode = 2
)

// FontSmoothing returns the current system font smoothing mode.
func FontSmoothing() FontSmoothingMode {
	return FontSmoothingMode(GetFontSmoothingRaw())
}

// SetFontSmoothing updates the system font smoothing mode.
func SetFontSmoothing(mode FontSmoothingMode) {
	SetFontSmoothingRaw(uint8(mode))
}
