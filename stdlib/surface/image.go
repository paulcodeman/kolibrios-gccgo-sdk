package surface

type Image struct {
	Width  int
	Height int
	Pixels []uint32
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
