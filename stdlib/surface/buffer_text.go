package surface

import "kos"

func (buffer *Buffer) DrawText(x int, y int, color kos.Color, text string) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.DrawText(x, y, uint32(color), text)
}

func (buffer *Buffer) DrawTextFont(x int, y int, color kos.Color, text string, font *Font) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.DrawTextFont(x, y, uint32(color), text, font)
}
