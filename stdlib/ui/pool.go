package ui

import "sync"

const (
	maxPooledTextLines   = 256
	maxPooledTextPixels  = 16384
)

var textLineSlicePool = sync.Pool{
	New: func() any {
		return make([]textLine, 0, 8)
	},
}

var textPixelSlicePool = sync.Pool{
	New: func() any {
		return make([]uint32, 0, 256)
	},
}

func getTextLineSlice(capacity int) []textLine {
	if capacity < 0 {
		capacity = 0
	}
	value := textLineSlicePool.Get()
	if value == nil {
		return make([]textLine, 0, capacity)
	}
	lines := value.([]textLine)
	if cap(lines) < capacity {
		return make([]textLine, 0, capacity)
	}
	return lines[:0]
}

func putTextLineSlice(lines []textLine) {
	if lines == nil {
		return
	}
	if cap(lines) > maxPooledTextLines {
		return
	}
	for i := range lines {
		lines[i].text = ""
		lines[i].start = 0
		lines[i].end = 0
	}
	textLineSlicePool.Put(lines[:0])
}

func releaseTextLines(lines []textLine) {
	putTextLineSlice(lines)
}

func getTextPixelSlice(size int) []uint32 {
	if size < 0 {
		size = 0
	}
	value := textPixelSlicePool.Get()
	if value == nil {
		return make([]uint32, size)
	}
	pixels := value.([]uint32)
	if cap(pixels) < size {
		return make([]uint32, size)
	}
	return pixels[:size]
}

func putTextPixelSlice(pixels []uint32) {
	if pixels == nil {
		return
	}
	if cap(pixels) > maxPooledTextPixels {
		return
	}
	textPixelSlicePool.Put(pixels[:0])
}

func releaseTextPixels(pixels []uint32) {
	putTextPixelSlice(pixels)
}
