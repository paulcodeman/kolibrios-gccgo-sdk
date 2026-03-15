package main

import (
	"kos"
	"ui"
	"unsafe"
)

type pageBuffer struct {
	width  int
	height int
	data   []uint32
}

func newPageBuffer(width int, height int) *pageBuffer {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	size := 2 + width*height
	buf := make([]uint32, size)
	buf[0] = uint32(width)
	buf[1] = uint32(height)
	return &pageBuffer{
		width:  width,
		height: height,
		data:   buf,
	}
}

func (b *pageBuffer) headerPtr() *byte {
	if b == nil || len(b.data) == 0 {
		return nil
	}
	return (*byte)(unsafe.Pointer(&b.data[0]))
}

func (b *pageBuffer) pixelPtr(offsetPixels int) *byte {
	if b == nil {
		return nil
	}
	index := 2 + offsetPixels
	if index < 2 || index >= len(b.data) {
		return nil
	}
	return (*byte)(unsafe.Pointer(&b.data[index]))
}

func (b *pageBuffer) fill(color kos.Color) {
	if b == nil || len(b.data) < 2 {
		return
	}
	c := uint32(color) | 0xFF000000
	pixels := b.data[2:]
	for i := range pixels {
		pixels[i] = c
	}
}

func (b *pageBuffer) drawBar(x int, y int, width int, height int, color kos.Color) {
	if b == nil || width <= 0 || height <= 0 {
		return
	}
	if x < 0 {
		width += x
		x = 0
	}
	if y < 0 {
		height += y
		y = 0
	}
	if x >= b.width || y >= b.height || width <= 0 || height <= 0 {
		return
	}
	if x+width > b.width {
		width = b.width - x
	}
	if y+height > b.height {
		height = b.height - y
	}
	c := uint32(color) | 0xFF000000
	rowStart := 2 + y*b.width + x
	for row := 0; row < height; row++ {
		idx := rowStart + row*b.width
		for col := 0; col < width; col++ {
			b.data[idx+col] = c
		}
	}
}

func (b *pageBuffer) drawText(x int, y int, color kos.Color, text string) {
	if b == nil || text == "" || x >= b.width || y >= b.height {
		return
	}
	kos.DrawTextBuffer(x, y, color, text, b.headerPtr())
}

func (b *pageBuffer) show(x int, y int, offsetY int, height int) {
	if b == nil || b.width <= 0 || b.height <= 0 || height <= 0 {
		return
	}
	if offsetY < 0 {
		offsetY = 0
	}
	if offsetY >= b.height {
		return
	}
	if height > b.height-offsetY {
		height = b.height - offsetY
	}
	start := offsetY * b.width
	ptr := b.pixelPtr(start)
	if ptr == nil {
		return
	}
	kos.PutImage32(ptr, b.width, height, x, y)
}

func (app *App) buildPageBuffer() {
	if app.contentW <= 0 {
		app.pageBuf = nil
		return
	}
	height := len(app.lines) * fontH
	if height < app.contentH {
		height = app.contentH
	}
	buf := newPageBuffer(app.contentW, height)
	buf.fill(ui.White)
	for i, line := range app.lines {
		bufY := i * fontH
		app.drawLineToBuffer(buf, line, 0, bufY)
	}
	app.pageBuf = buf
}

func (app *App) drawLineToBuffer(buf *pageBuffer, line RenderLine, x int, y int) {
	if buf == nil || len(line.Text) == 0 {
		return
	}
	if len(line.Spans) == 0 {
		buf.drawText(x, y, ui.Black, line.Text)
		return
	}
	pos := 0
	for _, span := range line.Spans {
		if span.Start > pos {
			buf.drawText(x+pos*fontW, y, ui.Black, line.Text[pos:span.Start])
		}
		buf.drawText(x+span.Start*fontW, y, ui.Blue, line.Text[span.Start:span.End])
		pos = span.End
	}
	if pos < len(line.Text) {
		buf.drawText(x+pos*fontW, y, ui.Black, line.Text[pos:])
	}
}

func (app *App) drawHoverOverlay(visible int) {
	if app.hoverLink < 0 || visible <= 0 {
		return
	}
	for i := 0; i < visible; i++ {
		index := app.firstLine + i
		if index < 0 || index >= len(app.lines) {
			continue
		}
		line := app.lines[index]
		for _, span := range line.Spans {
			if span.Link != app.hoverLink {
				continue
			}
			kos.DrawText(app.contentX+span.Start*fontW, app.contentY+i*fontH, ui.Red, line.Text[span.Start:span.End])
		}
	}
}
