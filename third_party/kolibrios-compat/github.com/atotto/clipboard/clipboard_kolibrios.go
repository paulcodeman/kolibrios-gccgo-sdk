//go:build kolibrios && gccgo

package clipboard

import (
	"errors"
	"kos"
	"unsafe"

	"golang.org/x/text/encoding/charmap"
)

func readAll() (string, error) {
	count, status := kos.ClipboardSlotCount()
	if status != kos.ClipboardOK {
		return "", errors.New("clipboard: native clipboard unavailable")
	}
	if count == 0 {
		return "", nil
	}
	address, status := kos.ClipboardSlotData(count - 1)
	if status != kos.ClipboardOK {
		return "", errors.New("clipboard: cannot read native slot")
	}
	// Function 54/1 allocates a local-heap copy; ownership belongs to this
	// caller and must be released with 68/13 after copying the text.
	defer kos.HeapFreeRaw(address)
	header := (*[3]uint32)(unsafe.Pointer(uintptr(address)))
	if header[0] < 12 || header[0] > 64<<20 || header[1] != uint32(kos.ClipboardTypeText) {
		return "", errors.New("clipboard: latest slot is not a valid text slot")
	}
	data := (*[64 << 20]byte)(unsafe.Pointer(uintptr(address) + 12))[:int(header[0])-12]
	// Text slots may include their terminating NUL in the stored size.
	for i, b := range data {
		if b == 0 {
			data = data[:i]
			break
		}
	}
	switch kos.ClipboardEncoding(header[2]) {
	case kos.ClipboardEncodingUTF:
		return string(data), nil
	case kos.ClipboardEncodingCP866:
		decoded, err := charmap.CodePage866.NewDecoder().Bytes(data)
		return string(decoded), err
	case kos.ClipboardEncodingCP1251:
		decoded, err := charmap.Windows1251.NewDecoder().Bytes(data)
		return string(decoded), err
	default:
		return "", errors.New("clipboard: unsupported native text encoding")
	}
}

func writeAll(text string) error {
	if kos.ClipboardCopyText(text) != kos.ClipboardOK {
		return errors.New("clipboard: cannot write native text slot")
	}
	return nil
}
