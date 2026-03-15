package kos

const ProcLibDLLPath = "/sys/lib/proc_lib.obj"

const (
	openDialogRawSize  = 56
	colorDialogRawSize = 44

	openDialogProcInfoSize   = 1024
	openDialogPathBufferSize = 4096
	openDialogNameBufferSize = 256
	openDialogFilterSize     = 4096

	openDialogAreaNameTemplate  = "FFFFFFFF_open_dialog"
	openDialogDefaultDir        = "/sys"
	openDialogStartPath         = "/sys/File managers/opendial"
	colorDialogAreaNameTemplate = "FFFFFFFF_color_dialog"
	colorDialogStartPath        = "/sys/colrdial"

	openDialogOffsetMode         = 0
	openDialogOffsetProcInfo     = 4
	openDialogOffsetAreaName     = 8
	openDialogOffsetComArea      = 12
	openDialogOffsetOpenDir      = 16
	openDialogOffsetDefaultDir   = 20
	openDialogOffsetStartPath    = 24
	openDialogOffsetDrawWindow   = 28
	openDialogOffsetStatus       = 32
	openDialogOffsetOpenFilePath = 36
	openDialogOffsetFileName     = 40
	openDialogOffsetFilter       = 44
	openDialogOffsetWidth        = 48
	openDialogOffsetX            = 50
	openDialogOffsetHeight       = 52
	openDialogOffsetY            = 54

	colorDialogOffsetType       = 0
	colorDialogOffsetProcInfo   = 4
	colorDialogOffsetAreaName   = 8
	colorDialogOffsetComArea    = 12
	colorDialogOffsetStartPath  = 16
	colorDialogOffsetDrawWindow = 20
	colorDialogOffsetStatus     = 24
	colorDialogOffsetWidth      = 28
	colorDialogOffsetX          = 30
	colorDialogOffsetHeight     = 32
	colorDialogOffsetY          = 34
	colorDialogOffsetColorType  = 36
	colorDialogOffsetColor      = 40
)

type ProcLib struct {
	table                 DLLExportTable
	openDialogInitProc    DLLProc
	openDialogStartProc   DLLProc
	openDialogSetNameProc DLLProc
	openDialogSetExtProc  DLLProc
	colorDialogInitProc   DLLProc
	colorDialogStartProc  DLLProc
	version               uint32
	openDialogVersion     uint32
	colorDialogVersion    uint32
	ready                 bool
}

type OpenDialogMode uint32
type OpenDialogStatus uint32
type ColorDialogType uint32
type ColorDialogStatus uint32

const (
	OpenDialogOpen OpenDialogMode = iota
	OpenDialogSave
	OpenDialogSelectDirectory
)

const (
	OpenDialogCanceled OpenDialogStatus = iota
	OpenDialogOK
	OpenDialogAlternative
)

const (
	ColorDialogPaletteAndTone ColorDialogType = 0
)

const (
	ColorDialogCanceled ColorDialogStatus = iota
	ColorDialogOK
	ColorDialogAlternative
)

type OpenDialog struct {
	raw         [openDialogRawSize]byte
	procInfo    [openDialogProcInfoSize]byte
	areaName    [32]byte
	openDirPath [openDialogPathBufferSize]byte
	defaultDir  [openDialogPathBufferSize]byte
	startPath   [64]byte
	openFile    [openDialogPathBufferSize]byte
	fileName    [openDialogNameBufferSize]byte
	filter      [openDialogFilterSize]byte
}

type ColorDialog struct {
	raw       [colorDialogRawSize]byte
	procInfo  [openDialogProcInfoSize]byte
	areaName  [32]byte
	startPath [32]byte
}

func LoadProcLibDLL() DLLExportTable {
	return LoadDLLFile(ProcLibDLLPath)
}

func LoadProcLib() (ProcLib, bool) {
	return LoadProcLibFromDLL(LoadProcLibDLL())
}

func LoadProcLibFromDLL(table DLLExportTable) (ProcLib, bool) {
	lib := ProcLib{
		table:                 table,
		openDialogInitProc:    LookupDLLExportAny(table, "OpenDialog_init"),
		openDialogStartProc:   LookupDLLExportAny(table, "OpenDialog_start"),
		openDialogSetNameProc: LookupDLLExportAny(table, "OpenDialog_set_file_name"),
		openDialogSetExtProc:  LookupDLLExportAny(table, "OpenDialog_set_file_ext"),
		colorDialogInitProc:   LookupDLLExportAny(table, "ColorDialog_init"),
		colorDialogStartProc:  LookupDLLExportAny(table, "ColorDialog_start"),
		version:               uint32(LookupDLLExportAny(table, "version")),
		openDialogVersion:     uint32(LookupDLLExportAny(table, "Version_OpenDialog")),
		colorDialogVersion:    uint32(LookupDLLExportAny(table, "Version_ColorDialog")),
		ready:                 true,
	}
	initProc := LookupDLLExportAny(table, "lib_init")
	if !lib.Valid() {
		return ProcLib{}, false
	}
	if initProc.Valid() {
		InitDLLLibraryRaw(uint32(initProc))
	}

	return lib, true
}

func NewOpenDialog(mode OpenDialogMode, x int, y int, width int, height int) *OpenDialog {
	dialog := new(OpenDialog)

	copyCStringBuffer(dialog.areaName[:], openDialogAreaNameTemplate)
	copyCStringBuffer(dialog.defaultDir[:], openDialogDefaultDir)
	copyCStringBuffer(dialog.startPath[:], openDialogStartPath)
	dialog.setMode(mode)
	dialog.setRect(x, y, width, height)
	dialog.setPointer(openDialogOffsetProcInfo, byteSliceAddress(dialog.procInfo[:]))
	dialog.setPointer(openDialogOffsetAreaName, byteSliceAddress(dialog.areaName[:]))
	dialog.setPointer(openDialogOffsetComArea, 0)
	dialog.setPointer(openDialogOffsetOpenDir, byteSliceAddress(dialog.openDirPath[:]))
	dialog.setPointer(openDialogOffsetDefaultDir, byteSliceAddress(dialog.defaultDir[:]))
	dialog.setPointer(openDialogOffsetStartPath, byteSliceAddress(dialog.startPath[:]))
	dialog.setPointer(openDialogOffsetDrawWindow, DialogNoopProcRaw())
	dialog.setPointer(openDialogOffsetStatus, 0)
	dialog.setPointer(openDialogOffsetOpenFilePath, byteSliceAddress(dialog.openFile[:]))
	dialog.setPointer(openDialogOffsetFileName, byteSliceAddress(dialog.fileName[:]))
	dialog.setPointer(openDialogOffsetFilter, byteSliceAddress(dialog.filter[:]))
	dialog.filter[0] = 0
	dialog.filter[1] = 0
	dialog.filter[2] = 0
	dialog.filter[3] = 0
	return dialog
}

func NewColorDialog(kind ColorDialogType, x int, y int, width int, height int) *ColorDialog {
	dialog := new(ColorDialog)

	copyCStringBuffer(dialog.areaName[:], colorDialogAreaNameTemplate)
	copyCStringBuffer(dialog.startPath[:], colorDialogStartPath)
	dialog.setType(kind)
	dialog.setRect(x, y, width, height)
	dialog.setPointer(colorDialogOffsetProcInfo, byteSliceAddress(dialog.procInfo[:]))
	dialog.setPointer(colorDialogOffsetAreaName, byteSliceAddress(dialog.areaName[:]))
	dialog.setPointer(colorDialogOffsetComArea, 0)
	dialog.setPointer(colorDialogOffsetStartPath, byteSliceAddress(dialog.startPath[:]))
	dialog.setPointer(colorDialogOffsetDrawWindow, DialogNoopProcRaw())
	dialog.setPointer(colorDialogOffsetStatus, 0)
	dialog.setPointer(colorDialogOffsetColorType, 0)
	dialog.setPointer(colorDialogOffsetColor, 0)
	return dialog
}

func (lib ProcLib) Valid() bool {
	return lib.table != 0 &&
		lib.openDialogInitProc.Valid() &&
		lib.openDialogStartProc.Valid() &&
		lib.openDialogSetNameProc.Valid() &&
		lib.openDialogSetExtProc.Valid() &&
		lib.colorDialogInitProc.Valid() &&
		lib.colorDialogStartProc.Valid()
}

func (lib ProcLib) ExportTable() DLLExportTable {
	return lib.table
}

func (lib ProcLib) Version() uint32 {
	return lib.version
}

func (lib ProcLib) OpenDialogVersion() uint32 {
	return lib.openDialogVersion
}

func (lib ProcLib) ColorDialogVersion() uint32 {
	return lib.colorDialogVersion
}

func (lib ProcLib) Ready() bool {
	return lib.ready
}

func (lib ProcLib) InitOpenDialog(dialog *OpenDialog) bool {
	if !lib.ready || dialog == nil || !lib.openDialogInitProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.openDialogInitProc), dialog.address())
	return dialog.ComArea() != 0
}

func (lib ProcLib) StartOpenDialog(dialog *OpenDialog) (OpenDialogStatus, bool) {
	if !lib.ready || dialog == nil || !lib.openDialogStartProc.Valid() {
		return 0, false
	}

	CallStdcall1VoidRaw(uint32(lib.openDialogStartProc), dialog.address())
	return dialog.Status(), true
}

func (lib ProcLib) SetOpenDialogFileName(dialog *OpenDialog, value string) bool {
	return lib.setOpenDialogString(lib.openDialogSetNameProc, dialog, value)
}

func (lib ProcLib) SetOpenDialogFileExtension(dialog *OpenDialog, value string) bool {
	return lib.setOpenDialogString(lib.openDialogSetExtProc, dialog, value)
}

func (lib ProcLib) InitColorDialog(dialog *ColorDialog) bool {
	if !lib.ready || dialog == nil || !lib.colorDialogInitProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.colorDialogInitProc), dialog.address())
	return dialog.ComArea() != 0
}

func (lib ProcLib) StartColorDialog(dialog *ColorDialog) (ColorDialogStatus, bool) {
	if !lib.ready || dialog == nil || !lib.colorDialogStartProc.Valid() {
		return 0, false
	}

	CallStdcall1VoidRaw(uint32(lib.colorDialogStartProc), dialog.address())
	return dialog.Status(), true
}

func (dialog *OpenDialog) Mode() OpenDialogMode {
	if dialog == nil {
		return OpenDialogOpen
	}

	return OpenDialogMode(littleEndianUint32(dialog.raw[:], openDialogOffsetMode))
}

func (dialog *OpenDialog) SetMode(mode OpenDialogMode) {
	if dialog != nil {
		dialog.setMode(mode)
	}
}

func (dialog *OpenDialog) SetDirectory(path string) bool {
	if dialog == nil {
		return false
	}

	return copyCStringBuffer(dialog.openDirPath[:], path)
}

func (dialog *OpenDialog) SetDefaultDirectory(path string) bool {
	if dialog == nil {
		return false
	}

	return copyCStringBuffer(dialog.defaultDir[:], path)
}

func (dialog *OpenDialog) Status() OpenDialogStatus {
	if dialog == nil {
		return OpenDialogAlternative
	}

	return OpenDialogStatus(littleEndianUint32(dialog.raw[:], openDialogOffsetStatus))
}

func (dialog *OpenDialog) ComArea() uint32 {
	if dialog == nil {
		return 0
	}

	return littleEndianUint32(dialog.raw[:], openDialogOffsetComArea)
}

func (dialog *OpenDialog) AreaName() string {
	if dialog == nil {
		return ""
	}

	return cStringBufferToString(dialog.areaName[:])
}

func (dialog *OpenDialog) FilePath() string {
	if dialog == nil {
		return ""
	}

	return cStringBufferToString(dialog.openFile[:])
}

func (dialog *OpenDialog) FileName() string {
	if dialog == nil {
		return ""
	}

	return cStringBufferToString(dialog.fileName[:])
}

func (dialog *ColorDialog) Status() ColorDialogStatus {
	if dialog == nil {
		return ColorDialogAlternative
	}

	return ColorDialogStatus(littleEndianUint32(dialog.raw[:], colorDialogOffsetStatus))
}

func (dialog *ColorDialog) ComArea() uint32 {
	if dialog == nil {
		return 0
	}

	return littleEndianUint32(dialog.raw[:], colorDialogOffsetComArea)
}

func (dialog *ColorDialog) AreaName() string {
	if dialog == nil {
		return ""
	}

	return cStringBufferToString(dialog.areaName[:])
}

func (dialog *ColorDialog) ColorType() uint32 {
	if dialog == nil {
		return 0
	}

	return littleEndianUint32(dialog.raw[:], colorDialogOffsetColorType)
}

func (dialog *ColorDialog) Color() uint32 {
	if dialog == nil {
		return 0
	}

	return littleEndianUint32(dialog.raw[:], colorDialogOffsetColor)
}

func (dialog *ColorDialog) SetColorType(value uint32) {
	if dialog != nil {
		dialog.setPointer(colorDialogOffsetColorType, value)
	}
}

func (dialog *ColorDialog) SetColor(value uint32) {
	if dialog != nil {
		dialog.setPointer(colorDialogOffsetColor, value)
	}
}

func (lib ProcLib) setOpenDialogString(proc DLLProc, dialog *OpenDialog, value string) bool {
	if !lib.ready || dialog == nil || !proc.Valid() {
		return false
	}

	valuePtr, valueAddr := stringAddress(value)
	if valuePtr == nil {
		return false
	}

	CallStdcall2VoidRaw(uint32(proc), dialog.address(), valueAddr)
	freeCString(valuePtr)
	return true
}

func (dialog *OpenDialog) setMode(mode OpenDialogMode) {
	dialog.setPointer(openDialogOffsetMode, uint32(mode))
}

func (dialog *OpenDialog) setRect(x int, y int, width int, height int) {
	dialog.setUint16(openDialogOffsetWidth, clampUint16(width))
	dialog.setUint16(openDialogOffsetX, clampUint16(x))
	dialog.setUint16(openDialogOffsetHeight, clampUint16(height))
	dialog.setUint16(openDialogOffsetY, clampUint16(y))
}

func (dialog *OpenDialog) address() uint32 {
	if dialog == nil {
		return 0
	}

	return byteSliceAddress(dialog.raw[:])
}

func (dialog *OpenDialog) setPointer(offset int, value uint32) {
	putUint32LE(dialog.raw[:], offset, value)
}

func (dialog *OpenDialog) setUint16(offset int, value uint16) {
	putUint16LE(dialog.raw[:], offset, value)
}

func (dialog *ColorDialog) setType(kind ColorDialogType) {
	dialog.setPointer(colorDialogOffsetType, uint32(kind))
}

func (dialog *ColorDialog) setRect(x int, y int, width int, height int) {
	dialog.setUint16(colorDialogOffsetWidth, clampUint16(width))
	dialog.setUint16(colorDialogOffsetX, clampUint16(x))
	dialog.setUint16(colorDialogOffsetHeight, clampUint16(height))
	dialog.setUint16(colorDialogOffsetY, clampUint16(y))
}

func (dialog *ColorDialog) address() uint32 {
	if dialog == nil {
		return 0
	}

	return byteSliceAddress(dialog.raw[:])
}

func (dialog *ColorDialog) setPointer(offset int, value uint32) {
	putUint32LE(dialog.raw[:], offset, value)
}

func (dialog *ColorDialog) setUint16(offset int, value uint16) {
	putUint16LE(dialog.raw[:], offset, value)
}

func copyCStringBuffer(dst []byte, value string) bool {
	if len(dst) == 0 || len(value) >= len(dst) {
		return false
	}

	for index := 0; index < len(dst); index++ {
		dst[index] = 0
	}
	copy(dst[:len(value)], value)
	dst[len(value)] = 0
	return true
}

func clampUint16(value int) uint16 {
	if value < 0 {
		return 0
	}
	if value > 0xFFFF {
		return 0xFFFF
	}

	return uint16(value)
}

func putUint16LE(buffer []byte, offset int, value uint16) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
}

func putUint32LE(buffer []byte, offset int, value uint32) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
	buffer[offset+2] = byte(value >> 16)
	buffer[offset+3] = byte(value >> 24)
}
