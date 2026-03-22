package duit

import (
	"image"
	"io"

	"9fans.net/go/draw"
)

func ReadImage(display *draw.Display, r io.Reader) (*draw.Image, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	img, err := display.AllocImage(bounds, draw.ABGR32, false, draw.White)
	if err != nil {
		return nil, err
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, y, src.At(x, y))
		}
	}
	return img, nil
}
