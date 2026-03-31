//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kolibrios

import (
	"bytes"

	"fyne.io/fyne/v2"
	"kos"
)

type clipboard struct {
	fallback string
}

var _ fyne.Clipboard = (*clipboard)(nil)

func (c *clipboard) Content() string {
	if value, ok := readClipboardText(); ok {
		c.fallback = value
		return value
	}
	return c.fallback
}

func (c *clipboard) SetContent(content string) {
	c.fallback = content
	kos.ClipboardCopyText(content)
}

func readClipboardText() (string, bool) {
	count, status := kos.ClipboardSlotCount()
	if status != kos.ClipboardOK || count <= 0 {
		return "", false
	}
	ptr, status := kos.ClipboardSlotData(count - 1)
	if status != kos.ClipboardOK || ptr == 0 {
		return "", false
	}
	size := kos.ReadUint32Raw(ptr, 0)
	kind := kos.ReadUint32Raw(ptr, 4)
	if kind != uint32(kos.ClipboardTypeText) && kind != uint32(kos.ClipboardTypeTextBlock) {
		return "", false
	}
	offset := uint32(12)
	if size <= offset {
		return "", false
	}
	data := kos.CopyBytesRaw(ptr+offset, size-offset)
	if len(data) == 0 {
		return "", false
	}
	if idx := bytes.IndexByte(data, 0); idx >= 0 {
		data = data[:idx]
	}
	if len(data) == 0 {
		return "", false
	}
	return string(data), true
}
