package core

func colorToRGB(color uint32) (int, int, int) {
	value := uint32(color)
	return int((value >> 16) & 0xFF), int((value >> 8) & 0xFF), int(value & 0xFF)
}

func colorValue(color uint32) uint32 {
	return uint32(color) & 0xFFFFFF
}

func colorValueAndAlpha(color uint32) (uint32, uint8) {
	value := uint32(color)
	rgb := value & 0xFFFFFF
	if value > 0xFFFFFF {
		return rgb, uint8(value >> 24)
	}
	return rgb, 255
}

func combineAlpha(a uint8, b uint8) uint8 {
	return uint8((int(a)*int(b) + 127) / 255)
}

func blendPixelValue(dst uint32, colorValue uint32, alpha uint8) uint32 {
	if alpha == 0 {
		return dst
	}
	colorValue &= 0xFFFFFF
	if alpha >= 255 {
		return 0xFF000000 | colorValue
	}
	a := uint32(alpha)
	inv := uint32(255 - alpha)
	rb := (dst&0x00FF00FF)*inv + colorValue&0x00FF00FF*a + 0x00800080
	rb = (rb + ((rb >> 8) & 0x00FF00FF)) >> 8
	g := ((dst>>8)&0xFF)*inv + ((colorValue>>8)&0xFF)*a + 0x80
	g = (g + (g >> 8)) >> 8
	return 0xFF000000 | (rb & 0x00FF00FF) | ((g & 0xFF) << 8)
}

func darkenPixelValue(dst uint32, alpha uint8) uint32 {
	if alpha == 0 {
		return dst
	}
	if alpha >= 255 {
		return 0xFF000000
	}
	inv := uint32(255 - alpha)
	rb := (dst&0x00FF00FF)*inv + 0x00800080
	rb = (rb + ((rb >> 8) & 0x00FF00FF)) >> 8
	rb &= 0x00FF00FF
	g := ((dst>>8)&0xFF)*inv + 0x80
	g = (g + (g >> 8)) >> 8
	return 0xFF000000 | rb | ((g & 0xFF) << 8)
}

func blendPremultipliedOpaque(dst uint32, src uint32) uint32 {
	sa := uint8(src >> 24)
	if sa == 0 {
		return dst
	}
	if sa >= 255 {
		return src
	}
	inv := uint32(255 - sa)
	rb := (dst&0x00FF00FF)*inv + 0x00800080
	rb = (rb + ((rb >> 8) & 0x00FF00FF)) >> 8
	rb = (rb & 0x00FF00FF) + (src & 0x00FF00FF)
	g := ((dst>>8)&0xFF)*inv + 0x80
	g = (g + (g >> 8)) >> 8
	g = ((g & 0xFF) << 8) + (src & 0x0000FF00)
	return 0xFF000000 | (rb & 0x00FF00FF) | (g & 0x0000FF00)
}

func premultiplyColorValue(colorValue uint32, alpha uint8) uint32 {
	colorValue &= 0xFFFFFF
	if alpha == 0 {
		return 0
	}
	if alpha >= 255 {
		return 0xFF000000 | colorValue
	}
	a := uint32(alpha)
	r := ((colorValue >> 16) & 0xFF) * a / 255
	g := ((colorValue >> 8) & 0xFF) * a / 255
	b := (colorValue & 0xFF) * a / 255
	return (a << 24) | (r << 16) | (g << 8) | b
}

func blendPremultiplied(dst uint32, src uint32) uint32 {
	sa := int((src >> 24) & 0xFF)
	if sa == 0 {
		return dst
	}
	if sa >= 255 {
		return src
	}
	inv := 255 - sa
	da := int((dst >> 24) & 0xFF)
	outA := sa + (da*inv+127)/255
	outR := int((src>>16)&0xFF) + (int((dst>>16)&0xFF)*inv+127)/255
	outG := int((src>>8)&0xFF) + (int((dst>>8)&0xFF)*inv+127)/255
	outB := int(src&0xFF) + (int(dst&0xFF)*inv+127)/255
	return uint32(outA<<24 | outR<<16 | outG<<8 | outB)
}

func (buffer *Buffer) blendPixel(dst uint32, colorValue uint32, alpha uint8) uint32 {
	if buffer == nil || !buffer.alpha {
		return blendPixelValue(dst, colorValue, alpha)
	}
	src := premultiplyColorValue(colorValue, alpha)
	return blendPremultiplied(dst, src)
}
