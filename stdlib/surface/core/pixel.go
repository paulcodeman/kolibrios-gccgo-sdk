package core

import "kos"

func (buffer *Buffer) HasAlpha() bool {
	if buffer == nil {
		return false
	}
	return buffer.alpha
}

func (buffer *Buffer) PixelValue(x int, y int) uint32 {
	if buffer == nil || x < 0 || y < 0 || x >= buffer.width || y >= buffer.height {
		return 0
	}
	return buffer.data[2+y*buffer.width+x]
}

func (buffer *Buffer) SetPixelValue(x int, y int, value uint32) {
	if buffer == nil || x < 0 || y < 0 || x >= buffer.width || y >= buffer.height {
		return
	}
	if buffer.clip.set && !buffer.clip.rect.Contains(x, y) {
		return
	}
	buffer.data[2+y*buffer.width+x] = value
}

func (buffer *Buffer) BlendPremultipliedPixelValue(x int, y int, value uint32) {
	if buffer == nil || x < 0 || y < 0 || x >= buffer.width || y >= buffer.height {
		return
	}
	if buffer.clip.set && !buffer.clip.rect.Contains(x, y) {
		return
	}
	index := 2 + y*buffer.width + x
	if !buffer.alpha {
		alpha := uint8(value >> 24)
		if alpha == 0 {
			return
		}
		if alpha >= 255 {
			buffer.data[index] = 0xFF000000 | (value & 0xFFFFFF)
			return
		}
		buffer.data[index] = blendPixelValue(buffer.data[index], value&0xFFFFFF, alpha)
		return
	}
	buffer.data[index] = blendPremultiplied(buffer.data[index], value)
}

func (buffer *Buffer) SetPixel(x int, y int, color kos.Color) {
	if buffer == nil || x < 0 || y < 0 || x >= buffer.width || y >= buffer.height {
		return
	}
	if buffer.clip.set && !buffer.clip.rect.Contains(x, y) {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	index := 2 + y*buffer.width + x
	if alpha >= 255 {
		buffer.data[index] = 0xFF000000 | rgb
		return
	}
	if alpha == 0 {
		return
	}
	buffer.data[index] = buffer.blendPixel(buffer.data[index], rgb, alpha)
}
