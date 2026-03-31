package surface

import "image"

// NewImageFromSource converts a Go image.Image into the premultiplied pixel
// format used by surface.Image.
func NewImageFromSource(source image.Image) *Image {
	if source == nil {
		return nil
	}
	width, height, pixels := convertImagePixels(source)
	if width <= 0 || height <= 0 || len(pixels) == 0 {
		return nil
	}
	return &Image{
		Width:  width,
		Height: height,
		Pixels: pixels,
	}
}
