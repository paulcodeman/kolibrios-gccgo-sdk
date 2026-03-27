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
		Width:       image.Width,
		Height:      image.Height,
		Pixels:      pixels,
		opaqueState: image.opaqueState,
	}
}

func ScaleImageNearest(image *Image, width int, height int) *Image {
	if image == nil || !image.Valid() || width <= 0 || height <= 0 {
		return nil
	}
	pixels := make([]uint32, width*height)
	stepX := (int64(image.Width) << 16) / int64(width)
	xFixed := int64(0)
	xMap := make([]int, width)
	for dstX := 0; dstX < width; dstX++ {
		srcX := int(xFixed >> 16)
		if srcX >= image.Width {
			srcX = image.Width - 1
		}
		xMap[dstX] = srcX
		xFixed += stepX
	}
	stepY := (int64(image.Height) << 16) / int64(height)
	yFixed := int64(0)
	for dstY := 0; dstY < height; dstY++ {
		srcY := int(yFixed >> 16)
		if srcY >= image.Height {
			srcY = image.Height - 1
		}
		row := srcY * image.Width
		for dstX := 0; dstX < width; dstX++ {
			pixels[dstY*width+dstX] = image.Pixels[row+xMap[dstX]]
		}
		yFixed += stepY
	}
	return &Image{
		Width:       width,
		Height:      height,
		Pixels:      pixels,
		opaqueState: image.opaqueState,
	}
}
