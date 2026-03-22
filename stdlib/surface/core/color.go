package core

import "kos"

func colorValue(color kos.Color) uint32 {
	return uint32(color) & 0xFFFFFF
}

func colorValueAndAlpha(color kos.Color) (uint32, uint8) {
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
	srcR := int((colorValue >> 16) & 0xFF)
	srcG := int((colorValue >> 8) & 0xFF)
	srcB := int(colorValue & 0xFF)
	dstR := int((dst >> 16) & 0xFF)
	dstG := int((dst >> 8) & 0xFF)
	dstB := int(dst & 0xFF)
	a := int(alpha)
	inv := 255 - a
	outR := (srcR*a + dstR*inv + 127) / 255
	outG := (srcG*a + dstG*inv + 127) / 255
	outB := (srcB*a + dstB*inv + 127) / 255
	return 0xFF000000 | uint32(outR<<16|outG<<8|outB)
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
