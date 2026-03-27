package surface

func MirrorImage(image *Image) *Image {
	if image == nil || !image.Valid() {
		return nil
	}
	pixels := make([]uint32, len(image.Pixels))
	for y := 0; y < image.Height; y++ {
		row := y * image.Width
		for x := 0; x < image.Width; x++ {
			pixels[row+x] = image.Pixels[row+(image.Width-1-x)]
		}
	}
	return &Image{
		Width:  image.Width,
		Height: image.Height,
		Pixels: pixels,
	}
}
