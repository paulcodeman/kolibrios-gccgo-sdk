package surface

type Image struct {
	Width  int
	Height int
	Pixels []uint32

	opaqueState uint8
}

func (image *Image) Valid() bool {
	if image == nil {
		return false
	}
	if image.Width <= 0 || image.Height <= 0 {
		return false
	}
	return len(image.Pixels) >= image.Width*image.Height
}

func (image *Image) Opaque() bool {
	if !image.Valid() {
		return false
	}
	switch image.opaqueState {
	case 1:
		return false
	case 2:
		return true
	}
	limit := image.Width * image.Height
	for index := 0; index < limit; index++ {
		if image.Pixels[index]>>24 != 0xFF {
			image.opaqueState = 1
			return false
		}
	}
	image.opaqueState = 2
	return true
}
