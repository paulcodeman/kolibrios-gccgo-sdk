package kos

const IconvDLLPath = "/sys/lib/iconv.obj"

const (
	iconvCodeCP866    = 0
	iconvCodeCP1251   = 1
	iconvCodeCP1252   = 2
	iconvCodeKOI8RU   = 3
	iconvCodeISO88595 = 4
	iconvCodeUTF8     = 5
)

type IconvDescriptor uint32

const (
	IconvCharsetUTF8     = "UTF-8"
	IconvCharsetKOI8RU   = "KOI8-RU"
	IconvCharsetCP1251   = "CP1251"
	IconvCharsetCP1252   = "CP1252"
	IconvCharsetISO88595 = "ISO8859-5"
	IconvCharsetCP866    = "CP866"
)

type Iconv struct {
	table       DLLExportTable
	openProc    DLLProc
	convertProc DLLProc
	version     uint32
}

func LoadIconvDLL() DLLExportTable {
	return LoadDLLFile(IconvDLLPath)
}

func LoadIconv() (Iconv, bool) {
	return LoadIconvFromDLL(LoadIconvDLL())
}

func LoadIconvFromDLL(table DLLExportTable) (Iconv, bool) {
	lib := Iconv{
		table:       table,
		openProc:    LookupDLLExportAny(table, "iconv_open"),
		convertProc: LookupDLLExportAny(table, "iconv"),
		version:     uint32(LookupDLLExportAny(table, "version")),
	}
	if !lib.Valid() {
		return Iconv{}, false
	}

	return lib, true
}

func (lib Iconv) Valid() bool {
	return lib.table != 0 &&
		lib.openProc.Valid() &&
		lib.convertProc.Valid()
}

func (lib Iconv) ExportTable() DLLExportTable {
	return lib.table
}

func (lib Iconv) Version() uint32 {
	return lib.version
}

func (lib Iconv) Open(fromEncoding string, toEncoding string) (IconvDescriptor, bool) {
	var fromPtr *byte
	var toPtr *byte
	var fromAddr uint32
	var toAddr uint32
	var descriptor int32

	if !lib.Valid() {
		return 0, false
	}

	fromPtr, fromAddr = stringAddress(fromEncoding)
	if fromPtr == nil {
		return 0, false
	}
	toPtr, toAddr = stringAddress(toEncoding)
	if toPtr == nil {
		freeCString(fromPtr)
		return 0, false
	}

	// ICONV.OBJ is built from GCC C code and uses the historical
	// "from, to" calling order in existing Kolibri wrappers.
	descriptor = int32(CallCdecl2Raw(uint32(lib.openProc), fromAddr, toAddr))
	freeCString(toPtr)
	freeCString(fromPtr)
	if descriptor < 0 {
		return 0, false
	}

	return IconvDescriptor(uint32(descriptor)), true
}

func (lib Iconv) Convert(descriptor IconvDescriptor, input []byte, output []byte) (written int, remainingInput int, status int32, ok bool) {
	var inputCopy []byte
	var inputPtr [4]byte
	var inputLen [4]byte
	var outputPtr [4]byte
	var outputLen [4]byte
	var initialOutput int
	var remainingOutput int

	if !lib.Valid() || descriptor == 0 || len(output) == 0 {
		return 0, len(input), -1, false
	}

	written, remainingInput, status, ok = convertIconvToUTF8(descriptor, input, output)
	if ok {
		return written, remainingInput, status, true
	}

	inputCopy = make([]byte, len(input)+1)
	copy(inputCopy, input)
	putUint32LE(inputPtr[:], 0, byteSliceAddress(inputCopy))
	putUint32LE(inputLen[:], 0, uint32(len(input)))
	putUint32LE(outputPtr[:], 0, byteSliceAddress(output))
	initialOutput = len(output)
	putUint32LE(outputLen[:], 0, uint32(initialOutput))

	status = int32(CallCdecl5Raw(
		uint32(lib.convertProc),
		uint32(descriptor),
		byteSliceAddress(inputPtr[:]),
		byteSliceAddress(inputLen[:]),
		byteSliceAddress(outputPtr[:]),
		byteSliceAddress(outputLen[:]),
	))
	remainingInput = int(littleEndianUint32(inputLen[:], 0))
	remainingOutput = int(littleEndianUint32(outputLen[:], 0))
	written = initialOutput - remainingOutput
	if written < 0 {
		written = 0
	}

	return written, remainingInput, status, true
}

func (lib Iconv) ConvertString(fromEncoding string, toEncoding string, input string) (string, int32, bool) {
	var descriptor IconvDescriptor
	var ok bool
	var buffer []byte
	var written int
	var remaining int
	var status int32

	descriptor, ok = lib.Open(fromEncoding, toEncoding)
	if !ok {
		return "", -1, false
	}

	buffer = make([]byte, len(input)*4+8)
	written, remaining, status, ok = lib.Convert(descriptor, []byte(input), buffer)
	if !ok || status != 0 || remaining != 0 {
		return "", status, false
	}

	return string(buffer[:written]), status, true
}

func convertIconvToUTF8(descriptor IconvDescriptor, input []byte, output []byte) (written int, remainingInput int, status int32, ok bool) {
	fromCode, toCode := iconvDescriptorCodes(descriptor)
	if toCode != iconvCodeUTF8 {
		return 0, 0, 0, false
	}

	if fromCode == iconvCodeUTF8 {
		if len(input) <= len(output) {
			copy(output, input)
			return len(input), 0, 0, true
		}

		copy(output, input[:len(output)])
		return len(output), len(input) - len(output), -12, true
	}

	if fromCode != iconvCodeCP866 {
		return 0, len(input), -10, true
	}

	for index := 0; index < len(input); index++ {
		r := decodeCP866Rune(input[index])
		need := iconvRuneLen(r)
		if need < 0 {
			return written, len(input) - index, -10, true
		}
		if written+need > len(output) {
			return written, len(input) - index, -12, true
		}

		written += iconvEncodeRune(output[written:], r)
	}

	return written, 0, 0, true
}

func iconvDescriptorCodes(descriptor IconvDescriptor) (fromCode int32, toCode int32) {
	value := uint32(descriptor)
	return int32(value >> 16), int32(value & 0xFFFF)
}

func decodeCP866Rune(value byte) rune {
	if value < 0x80 {
		return rune(value)
	}
	if value < 0xB0 {
		return rune(value) + 0x0390
	}

	return rune(cp866ToUnicode[value-0xB0])
}

func iconvRuneLen(r rune) int {
	switch {
	case r < 0:
		return -1
	case r <= 0x7F:
		return 1
	case r <= 0x7FF:
		return 2
	case r <= 0xFFFF:
		return 3
	case r <= 0x10FFFF:
		return 4
	default:
		return -1
	}
}

func iconvEncodeRune(buffer []byte, r rune) int {
	switch {
	case r <= 0x7F:
		buffer[0] = byte(r)
		return 1
	case r <= 0x7FF:
		buffer[0] = 0xC0 | byte(r>>6)
		buffer[1] = 0x80 | byte(r)&0x3F
		return 2
	case r <= 0xFFFF:
		buffer[0] = 0xE0 | byte(r>>12)
		buffer[1] = 0x80 | byte(r>>6)&0x3F
		buffer[2] = 0x80 | byte(r)&0x3F
		return 3
	default:
		buffer[0] = 0xF0 | byte(r>>18)
		buffer[1] = 0x80 | byte(r>>12)&0x3F
		buffer[2] = 0x80 | byte(r>>6)&0x3F
		buffer[3] = 0x80 | byte(r)&0x3F
		return 4
	}
}

var cp866ToUnicode = [...]uint16{
	0x2591, 0x2592, 0x2593, 0x2502, 0x2524, 0x2561, 0x2562, 0x2556,
	0x2555, 0x2563, 0x2551, 0x2557, 0x255D, 0x255C, 0x255B, 0x2510,
	0x2514, 0x2534, 0x252C, 0x251C, 0x2500, 0x253C, 0x255E, 0x255F,
	0x255A, 0x2554, 0x2569, 0x2566, 0x2560, 0x2550, 0x256C, 0x2567,
	0x2568, 0x2564, 0x2565, 0x2559, 0x2558, 0x2552, 0x2553, 0x256B,
	0x256A, 0x2518, 0x250C, 0x2588, 0x2584, 0x258C, 0x2590, 0x2580,
	0x0440, 0x0441, 0x0442, 0x0443, 0x0444, 0x0445, 0x0446, 0x0447,
	0x0448, 0x0449, 0x044A, 0x044B, 0x044C, 0x044D, 0x044E, 0x044F,
	0x0401, 0x0451, 0x0404, 0x0454, 0x0407, 0x0457, 0x040E, 0x045E,
	0x00B0, 0x2219, 0x00B7, 0x221A, 0x2116, 0x00A4, 0x25A0, 0x00A0,
}
