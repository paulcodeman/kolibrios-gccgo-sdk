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

func ScaleImageNearest(image *Image, width int, height int) *Image {
	if image == nil || !image.Valid() || width <= 0 || height <= 0 {
		return nil
	}
	pixels := make([]uint32, width*height)
	for dstY := 0; dstY < height; dstY++ {
		srcY := (dstY * image.Height) / height
		if srcY >= image.Height {
			srcY = image.Height - 1
		}
		row := srcY * image.Width
		for dstX := 0; dstX < width; dstX++ {
			srcX := (dstX * image.Width) / width
			if srcX >= image.Width {
				srcX = image.Width - 1
			}
			pixels[dstY*width+dstX] = image.Pixels[row+srcX]
		}
	}
	return &Image{
		Width:  width,
		Height: height,
		Pixels: pixels,
	}
}
