package core

type gradientStops struct {
	fromR int
	fromG int
	fromB int
	toR   int
	toG   int
	toB   int

	fromAlpha uint8
	toAlpha   uint8
}

func newGradientStops(gradient Gradient) gradientStops {
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	return gradientStops{
		fromR:     int((fromValue >> 16) & 0xFF),
		fromG:     int((fromValue >> 8) & 0xFF),
		fromB:     int(fromValue & 0xFF),
		toR:       int((toValue >> 16) & 0xFF),
		toG:       int((toValue >> 8) & 0xFF),
		toB:       int(toValue & 0xFF),
		fromAlpha: fromAlpha,
		toAlpha:   toAlpha,
	}
}

func lerpGradientValue(from int, to int, pos int, den int) int {
	return (from*(den-pos) + to*pos) / den
}

func (st gradientStops) opaqueSample(pos int, den int) uint32 {
	r := lerpGradientValue(st.fromR, st.toR, pos, den)
	g := lerpGradientValue(st.fromG, st.toG, pos, den)
	b := lerpGradientValue(st.fromB, st.toB, pos, den)
	return 0xFF000000 | uint32(r<<16|g<<8|b)
}

func (st gradientStops) sample(pos int, den int, alpha uint8) uint32 {
	r := lerpGradientValue(st.fromR, st.toR, pos, den)
	g := lerpGradientValue(st.fromG, st.toG, pos, den)
	b := lerpGradientValue(st.fromB, st.toB, pos, den)
	stopAlpha := uint8(lerpGradientValue(int(st.fromAlpha), int(st.toAlpha), pos, den))
	effective := combineAlpha(alpha, stopAlpha)
	return uint32(effective)<<24 | uint32(r<<16|g<<8|b)
}

func gradientAreaActive(area Rect) bool {
	return area.Width > 0 && area.Height > 0 && area != (Rect{})
}

func clampGradientPos(pos int, length int) int {
	if length <= 1 {
		return 0
	}
	if pos < 0 {
		return 0
	}
	if pos >= length {
		return length - 1
	}
	return pos
}

func gradientDen(length int) int {
	if length <= 1 {
		return 1
	}
	return length - 1
}

func (buffer *Buffer) FillRectGradient(x int, y int, width int, height int, gradient Gradient) {
	buffer.fillGradient(x, y, width, height, gradient, Rect{}, false, 255, CornerRadii{}, false)
}

func (buffer *Buffer) FillRoundedRectGradient(x int, y int, width int, height int, radii CornerRadii, gradient Gradient) {
	buffer.fillGradient(x, y, width, height, gradient, Rect{}, false, 255, radii, true)
}

func (buffer *Buffer) FillRectGradientAlpha(x int, y int, width int, height int, gradient Gradient, alpha uint8) {
	buffer.fillGradient(x, y, width, height, gradient, Rect{}, false, alpha, CornerRadii{}, false)
}

func (buffer *Buffer) FillRoundedRectGradientAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, alpha uint8) {
	buffer.fillGradient(x, y, width, height, gradient, Rect{}, false, alpha, radii, true)
}

func (buffer *Buffer) FillRectGradientArea(x int, y int, width int, height int, gradient Gradient, area Rect) {
	buffer.fillGradient(x, y, width, height, gradient, area, true, 255, CornerRadii{}, false)
}

func (buffer *Buffer) FillRoundedRectGradientArea(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect) {
	buffer.fillGradient(x, y, width, height, gradient, area, true, 255, radii, true)
}

func (buffer *Buffer) FillRectGradientAreaAlpha(x int, y int, width int, height int, gradient Gradient, area Rect, alpha uint8) {
	buffer.fillGradient(x, y, width, height, gradient, area, true, alpha, CornerRadii{}, false)
}

func (buffer *Buffer) FillRoundedRectGradientAreaAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect, alpha uint8) {
	buffer.fillGradient(x, y, width, height, gradient, area, true, alpha, radii, true)
}

func (buffer *Buffer) fillGradient(x int, y int, width int, height int, gradient Gradient, area Rect, useArea bool, alpha uint8, radii CornerRadii, rounded bool) {
	if buffer == nil || alpha == 0 {
		return
	}
	if useArea && !gradientAreaActive(area) {
		useArea = false
	}
	if rounded && !radii.Active() {
		rounded = false
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	if rounded {
		radii = normalizeRadii(width, height, radii)
		if !radii.Active() {
			rounded = false
		}
	}
	stops := newGradientStops(gradient)
	opaque := alpha >= 255 && stops.fromAlpha >= 255 && stops.toAlpha >= 255
	if opaque && width == 1 && height == 1 {
		buffer.FillRect(x, y, width, height, gradient.From)
		return
	}
	if gradient.Direction == GradientHorizontal {
		length := width
		offset := x
		if useArea {
			length = area.Width
			offset = area.X
		}
		den := gradientDen(length)
		samples := buffer.scratchPixels(width)
		for col := 0; col < width; col++ {
			pos := col
			if useArea {
				pos = clampGradientPos(x+col-offset, length)
			}
			if opaque {
				samples[col] = stops.opaqueSample(pos, den)
			} else {
				samples[col] = stops.sample(pos, den, alpha)
			}
		}
		for row := 0; row < height; row++ {
			rowStart := 2 + (y+row)*buffer.width + x
			switch {
			case rounded && opaque:
				buffer.paintRoundedRowOpaqueSamples(rowStart, row, width, height, radii, samples)
			case rounded:
				buffer.paintRoundedRowAlphaSamples(rowStart, row, width, height, radii, samples)
			case opaque:
				copy(buffer.data[rowStart:rowStart+width], samples)
			default:
				buffer.blendRowSamples(rowStart, samples)
			}
		}
		return
	}
	length := height
	offset := y
	if useArea {
		length = area.Height
		offset = area.Y
	}
	den := gradientDen(length)
	for row := 0; row < height; row++ {
		pos := row
		if useArea {
			pos = clampGradientPos(y+row-offset, length)
		}
		rowStart := 2 + (y+row)*buffer.width + x
		if opaque {
			value := stops.opaqueSample(pos, den)
			if rounded {
				buffer.paintRoundedRowOpaqueValue(rowStart, row, width, height, radii, value)
			} else {
				fill32(buffer.data[rowStart:rowStart+width], value)
			}
			continue
		}
		sample := stops.sample(pos, den, alpha)
		colorValue := sample & 0xFFFFFF
		rowAlpha := uint8(sample >> 24)
		if rounded {
			buffer.paintRoundedRowAlphaValue(rowStart, row, width, height, radii, colorValue, rowAlpha)
		} else {
			buffer.blendRowValue(rowStart, width, colorValue, rowAlpha)
		}
	}
}
