package kos

const LibImgDLLPath = "/sys/lib/libimg.obj"

type ImageFormat uint32
type ImageType uint32
type ImageHandle uint32

const (
	ImageFormatBMP  ImageFormat = 1
	ImageFormatICO  ImageFormat = 2
	ImageFormatCUR  ImageFormat = 3
	ImageFormatGIF  ImageFormat = 4
	ImageFormatPNG  ImageFormat = 5
	ImageFormatJPEG ImageFormat = 6
	ImageFormatTGA  ImageFormat = 7
	ImageFormatPCX  ImageFormat = 8
	ImageFormatXCF  ImageFormat = 9
	ImageFormatTIFF ImageFormat = 10
	ImageFormatPNM  ImageFormat = 11
	ImageFormatWBMP ImageFormat = 12
	ImageFormatXBM  ImageFormat = 13
	ImageFormatZ80  ImageFormat = 14
)

const (
	ImageTypeBPP8Indexed ImageType = 1
	ImageTypeBPP24       ImageType = 2
	ImageTypeBPP32       ImageType = 3
	ImageTypeBPP15       ImageType = 4
	ImageTypeBPP16       ImageType = 5
	ImageTypeBPP1        ImageType = 6
	ImageTypeBPP8Gray    ImageType = 7
	ImageTypeBPP2Indexed ImageType = 8
	ImageTypeBPP4Indexed ImageType = 9
	ImageTypeBPP8Alpha   ImageType = 10
)

const (
	libImgOffsetWidth  = 4
	libImgOffsetHeight = 8
	libImgOffsetType   = 20
	libImgOffsetData   = 24
)

type LibImg struct {
	table       DLLExportTable
	fromFileProc DLLProc
	createProc   DLLProc
	destroyProc  DLLProc
	countProc    DLLProc
	convertProc  DLLProc
	drawProc     DLLProc
	version      uint32
	ready        bool
}

func LoadLibImgDLL() DLLExportTable {
	return LoadDLLFile(LibImgDLLPath)
}

func LoadLibImg() (LibImg, bool) {
	return LoadLibImgFromDLL(LoadLibImgDLL())
}

func LoadLibImgFromDLL(table DLLExportTable) (LibImg, bool) {
	lib := LibImg{
		table:        table,
		fromFileProc: LookupDLLExportAny(table, "img_from_file"),
		createProc:   LookupDLLExportAny(table, "img_create"),
		destroyProc:  LookupDLLExportAny(table, "img_destroy"),
		countProc:    LookupDLLExportAny(table, "img_count"),
		convertProc:  LookupDLLExportAny(table, "img_convert"),
		drawProc:     LookupDLLExportAny(table, "img_draw"),
		version:      uint32(LookupDLLExportAny(table, "version")),
		ready:        true,
	}
	initProc := LookupDLLExportAny(table, "lib_init")
	if !lib.Valid() {
		return LibImg{}, false
	}
	if initProc.Valid() {
		InitDLLLibraryRaw(uint32(initProc))
	}

	return lib, true
}

func (lib LibImg) Valid() bool {
	return lib.table != 0 &&
		lib.fromFileProc.Valid() &&
		lib.createProc.Valid() &&
		lib.destroyProc.Valid() &&
		lib.countProc.Valid() &&
		lib.convertProc.Valid() &&
		lib.drawProc.Valid()
}

func (lib LibImg) ExportTable() DLLExportTable {
	return lib.table
}

func (lib LibImg) Version() uint32 {
	return lib.version
}

func (lib LibImg) Ready() bool {
	return lib.ready
}

func (lib LibImg) FromFile(path string) (ImageHandle, bool) {
	var pathPtr *byte
	var pathAddr uint32
	var handle ImageHandle

	if !lib.ready || !lib.fromFileProc.Valid() {
		return 0, false
	}

	pathPtr, pathAddr = stringAddress(path)
	if pathPtr == nil {
		return 0, false
	}

	handle = ImageHandle(CallStdcall1Raw(uint32(lib.fromFileProc), pathAddr))
	freeCString(pathPtr)
	return handle, handle != 0
}

func (lib LibImg) Create(width int, height int, kind ImageType) (ImageHandle, bool) {
	var handle ImageHandle

	if !lib.ready || !lib.createProc.Valid() || width < 1 || height < 1 {
		return 0, false
	}

	handle = ImageHandle(CallStdcall3Raw(uint32(lib.createProc), uint32(width), uint32(height), uint32(kind)))
	return handle, handle != 0
}

func (lib LibImg) Convert(image ImageHandle, dstType ImageType) (ImageHandle, bool) {
	var handle ImageHandle

	if !lib.ready || !lib.convertProc.Valid() || image == 0 {
		return 0, false
	}

	handle = ImageHandle(CallStdcall5Raw(uint32(lib.convertProc), uint32(image), 0, uint32(dstType), 0, 0))
	return handle, handle != 0
}

func (lib LibImg) Count(image ImageHandle) int {
	if !lib.ready || !lib.countProc.Valid() || image == 0 {
		return 0
	}

	return int(int32(CallStdcall1Raw(uint32(lib.countProc), uint32(image))))
}

func (lib LibImg) Draw(image ImageHandle, x int, y int, width int, height int, xOffset int, yOffset int) bool {
	if !lib.ready || !lib.drawProc.Valid() || image == 0 {
		return false
	}

	CallStdcall7Raw(
		uint32(lib.drawProc),
		uint32(image),
		uint32(x),
		uint32(y),
		uint32(width),
		uint32(height),
		uint32(xOffset),
		uint32(yOffset),
	)
	return true
}

func (lib LibImg) Destroy(image ImageHandle) bool {
	if !lib.ready || !lib.destroyProc.Valid() || image == 0 {
		return false
	}

	return CallStdcall1Raw(uint32(lib.destroyProc), uint32(image)) != 0
}

func (image ImageHandle) Valid() bool {
	return image != 0
}

func (image ImageHandle) Width() int {
	if image == 0 {
		return 0
	}

	return int(ReadUint32Raw(uint32(image), libImgOffsetWidth))
}

func (image ImageHandle) Height() int {
	if image == 0 {
		return 0
	}

	return int(ReadUint32Raw(uint32(image), libImgOffsetHeight))
}

func (image ImageHandle) Type() ImageType {
	if image == 0 {
		return 0
	}

	return ImageType(ReadUint32Raw(uint32(image), libImgOffsetType))
}

func (image ImageHandle) DataPointer() uint32 {
	if image == 0 {
		return 0
	}

	return ReadUint32Raw(uint32(image), libImgOffsetData)
}
