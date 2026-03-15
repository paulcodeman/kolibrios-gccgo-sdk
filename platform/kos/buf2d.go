package kos

import "unsafe"

const Buf2DDLLPath = "/sys/lib/buf2d.obj"

type Buf2DColorBits uint8

type Buf2DDitherAlgorithm uint32

type Buf2DCropOption uint32

const (
	Buf2DColorBits8  Buf2DColorBits = 8
	Buf2DColorBits24 Buf2DColorBits = 24
	Buf2DColorBits32 Buf2DColorBits = 32
)

const (
	Buf2DDitherSierraLite Buf2DDitherAlgorithm = iota
	Buf2DDitherFloydSteinberg
	Buf2DDitherBurkers
	Buf2DDitherHeavyIronMod
	Buf2DDitherAtkinson
)

const (
	Buf2DCropTop    Buf2DCropOption = 1
	Buf2DCropLeft   Buf2DCropOption = 2
	Buf2DCropBottom Buf2DCropOption = 4
	Buf2DCropRight  Buf2DCropOption = 8
)

type Buf2DImage struct {
	Data      uint32
	Left      uint16
	Top       uint16
	Width     uint32
	Height    uint32
	BgColor   uint32
	ColorBits uint8
	_         [3]byte
}

type Buf2D struct {
	table               DLLExportTable
	createProc          DLLProc
	createFromImageProc DLLProc
	clearProc           DLLProc
	drawProc            DLLProc
	deleteProc          DLLProc
	resizeProc          DLLProc
	rotateProc          DLLProc
	lineProc            DLLProc
	lineSmoothProc      DLLProc
	rectProc            DLLProc
	filledRectProc      DLLProc
	circleProc          DLLProc
	imgHDiv2Proc        DLLProc
	imgWDiv2Proc        DLLProc
	conv24To8Proc       DLLProc
	conv24To32Proc      DLLProc
	bitBltProc          DLLProc
	bitBltTranspProc    DLLProc
	bitBltAlphaProc     DLLProc
	curveBezierProc     DLLProc
	convertTextProc     DLLProc
	drawTextProc        DLLProc
	cropColorProc       DLLProc
	offsetHProc         DLLProc
	floodFillProc       DLLProc
	setPixelProc        DLLProc
	getPixelProc        DLLProc
	flipHProc           DLLProc
	flipVProc           DLLProc
	filterDitherProc    DLLProc
	ready               bool
}

func LoadBuf2DDLL() DLLExportTable {
	return LoadDLLFile(Buf2DDLLPath)
}

func LoadBuf2D() (Buf2D, bool) {
	return LoadBuf2DFromDLL(LoadBuf2DDLL())
}

func LoadBuf2DFromDLL(table DLLExportTable) (Buf2D, bool) {
	lib := Buf2D{
		table:               table,
		createProc:          LookupDLLExportAny(table, "buf2d_create"),
		createFromImageProc: LookupDLLExportAny(table, "buf2d_create_f_img"),
		clearProc:           LookupDLLExportAny(table, "buf2d_clear"),
		drawProc:            LookupDLLExportAny(table, "buf2d_draw"),
		deleteProc:          LookupDLLExportAny(table, "buf2d_delete"),
		resizeProc:          LookupDLLExportAny(table, "buf2d_resize"),
		rotateProc:          LookupDLLExportAny(table, "buf2d_rotate"),
		lineProc:            LookupDLLExportAny(table, "buf2d_line"),
		lineSmoothProc:      LookupDLLExportAny(table, "buf2d_line_sm"),
		rectProc:            LookupDLLExportAny(table, "buf2d_rect_by_size"),
		filledRectProc:      LookupDLLExportAny(table, "buf2d_filled_rect_by_size"),
		circleProc:          LookupDLLExportAny(table, "buf2d_circle"),
		imgHDiv2Proc:        LookupDLLExportAny(table, "buf2d_img_hdiv2"),
		imgWDiv2Proc:        LookupDLLExportAny(table, "buf2d_img_wdiv2"),
		conv24To8Proc:       LookupDLLExportAny(table, "buf2d_conv_24_to_8"),
		conv24To32Proc:      LookupDLLExportAny(table, "buf2d_conv_24_to_32"),
		bitBltProc:          LookupDLLExportAny(table, "buf2d_bit_blt"),
		bitBltTranspProc:    LookupDLLExportAny(table, "buf2d_bit_blt_transp"),
		bitBltAlphaProc:     LookupDLLExportAny(table, "buf2d_bit_blt_alpha"),
		curveBezierProc:     LookupDLLExportAny(table, "buf2d_curve_bezier", "buf2d_cruve_bezier"),
		convertTextProc:     LookupDLLExportAny(table, "buf2d_convert_text_matrix"),
		drawTextProc:        LookupDLLExportAny(table, "buf2d_draw_text"),
		cropColorProc:       LookupDLLExportAny(table, "buf2d_crop_color"),
		offsetHProc:         LookupDLLExportAny(table, "buf2d_offset_h"),
		floodFillProc:       LookupDLLExportAny(table, "buf2d_flood_fill"),
		setPixelProc:        LookupDLLExportAny(table, "buf2d_set_pixel"),
		getPixelProc:        LookupDLLExportAny(table, "buf2d_get_pixel"),
		flipHProc:           LookupDLLExportAny(table, "buf2d_flip_h"),
		flipVProc:           LookupDLLExportAny(table, "buf2d_flip_v"),
		filterDitherProc:    LookupDLLExportAny(table, "buf2d_filter_dither"),
		ready:               true,
	}
	if !lib.Valid() {
		return Buf2D{}, false
	}

	initProc := LookupDLLExportAny(table, "lib_init")
	if initProc.Valid() {
		InitDLLLibraryRaw(uint32(initProc))
	}

	return lib, true
}

func NewBuf2DImage(left int, top int, width int, height int, bgColor uint32, colorBits Buf2DColorBits) *Buf2DImage {
	return &Buf2DImage{
		Left:      buf2dClampUint16(left),
		Top:       buf2dClampUint16(top),
		Width:     buf2dClampUint32(width),
		Height:    buf2dClampUint32(height),
		BgColor:   bgColor,
		ColorBits: uint8(colorBits),
	}
}

func (lib Buf2D) Valid() bool {
	return lib.table != 0 &&
		lib.createProc.Valid() &&
		lib.clearProc.Valid() &&
		lib.drawProc.Valid() &&
		lib.deleteProc.Valid()
}

func (lib Buf2D) ExportTable() DLLExportTable {
	return lib.table
}

func (lib Buf2D) Ready() bool {
	return lib.ready
}

func (lib Buf2D) CreateBuffer(left int, top int, width int, height int, bgColor uint32, colorBits Buf2DColorBits) (*Buf2DImage, bool) {
	buffer := NewBuf2DImage(left, top, width, height, bgColor, colorBits)
	if !lib.Create(buffer) {
		return nil, false
	}

	return buffer, true
}

func (lib Buf2D) Create(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.createProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.createProc), buffer.address())
	return buffer.Data != 0
}

func (lib Buf2D) CreateFromRGB(buffer *Buf2DImage, rgb []byte) bool {
	if !lib.ready || buffer == nil || len(rgb) == 0 || !lib.createFromImageProc.Valid() {
		return false
	}

	CallStdcall2VoidRaw(uint32(lib.createFromImageProc), buffer.address(), byteSliceAddress(rgb))
	return buffer.Data != 0
}

func (lib Buf2D) Clear(buffer *Buf2DImage, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.clearProc.Valid() {
		return false
	}

	CallStdcall2VoidRaw(uint32(lib.clearProc), buffer.address(), color)
	return true
}

func (lib Buf2D) Draw(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.drawProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.drawProc), buffer.address())
	return true
}

func (lib Buf2D) Delete(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.deleteProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.deleteProc), buffer.address())
	buffer.Data = 0
	return true
}

func (lib Buf2D) Resize(buffer *Buf2DImage, width int, height int, mode int) bool {
	if !lib.ready || buffer == nil || !lib.resizeProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.resizeProc), buffer.address(), uint32(width), uint32(height), uint32(mode))
	return true
}

func (lib Buf2D) Rotate(buffer *Buf2DImage, angle int) bool {
	if !lib.ready || buffer == nil || !lib.rotateProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.rotateProc), buffer.address(), uint32(angle))
	return true
}

func (lib Buf2D) Line(buffer *Buf2DImage, x0 int, y0 int, x1 int, y1 int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.lineProc.Valid() {
		return false
	}

	CallStdcall6Raw(uint32(lib.lineProc), buffer.address(), uint32(x0), uint32(y0), uint32(x1), uint32(y1), color)
	return true
}

func (lib Buf2D) LineSmooth(buffer *Buf2DImage, x0 int, y0 int, x1 int, y1 int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.lineSmoothProc.Valid() {
		return false
	}

	CallStdcall6Raw(uint32(lib.lineSmoothProc), buffer.address(), uint32(x0), uint32(y0), uint32(x1), uint32(y1), color)
	return true
}

func (lib Buf2D) RectBySize(buffer *Buf2DImage, x int, y int, width int, height int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.rectProc.Valid() {
		return false
	}

	CallStdcall6Raw(uint32(lib.rectProc), buffer.address(), uint32(x), uint32(y), uint32(width), uint32(height), color)
	return true
}

func (lib Buf2D) FilledRectBySize(buffer *Buf2DImage, x int, y int, width int, height int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.filledRectProc.Valid() {
		return false
	}

	CallStdcall6Raw(uint32(lib.filledRectProc), buffer.address(), uint32(x), uint32(y), uint32(width), uint32(height), color)
	return true
}

func (lib Buf2D) Circle(buffer *Buf2DImage, x int, y int, radius int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.circleProc.Valid() {
		return false
	}

	CallStdcall5Raw(uint32(lib.circleProc), buffer.address(), uint32(x), uint32(y), uint32(radius), color)
	return true
}

func (lib Buf2D) ImageHalfHeight(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.imgHDiv2Proc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.imgHDiv2Proc), buffer.address())
	return true
}

func (lib Buf2D) ImageHalfWidth(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.imgWDiv2Proc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.imgWDiv2Proc), buffer.address())
	return true
}

func (lib Buf2D) Convert24To8(buffer *Buf2DImage, mode int) bool {
	if !lib.ready || buffer == nil || !lib.conv24To8Proc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.conv24To8Proc), buffer.address(), uint32(mode))
	return true
}

func (lib Buf2D) Convert24To32(buffer *Buf2DImage, mode int) bool {
	if !lib.ready || buffer == nil || !lib.conv24To32Proc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.conv24To32Proc), buffer.address(), uint32(mode))
	return true
}

func (lib Buf2D) BitBlt(dst *Buf2DImage, x int, y int, src *Buf2DImage) bool {
	if !lib.ready || dst == nil || src == nil || !lib.bitBltProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.bitBltProc), dst.address(), uint32(x), uint32(y), src.address())
	return true
}

func (lib Buf2D) BitBltTransp(dst *Buf2DImage, x int, y int, src *Buf2DImage) bool {
	if !lib.ready || dst == nil || src == nil || !lib.bitBltTranspProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.bitBltTranspProc), dst.address(), uint32(x), uint32(y), src.address())
	return true
}

func (lib Buf2D) BitBltAlpha(dst *Buf2DImage, x int, y int, src *Buf2DImage, color uint32) bool {
	if !lib.ready || dst == nil || src == nil || !lib.bitBltAlphaProc.Valid() {
		return false
	}

	CallStdcall5Raw(uint32(lib.bitBltAlphaProc), dst.address(), uint32(x), uint32(y), src.address(), color)
	return true
}

func (lib Buf2D) CurveBezier(buffer *Buf2DImage, x0 int, y0 int, x1 int, y1 int, x2 int, y2 int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.curveBezierProc.Valid() {
		return false
	}

	p0 := packUnsignedPoint(x0, y0)
	p1 := packUnsignedPoint(x1, y1)
	p2 := packUnsignedPoint(x2, y2)
	CallStdcall5Raw(uint32(lib.curveBezierProc), buffer.address(), p0, p1, p2, color)
	return true
}

func (lib Buf2D) ConvertTextMatrix(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.convertTextProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.convertTextProc), buffer.address())
	return true
}

func (lib Buf2D) DrawText(buffer *Buf2DImage, font *Buf2DImage, text string, x int, y int, color uint32) bool {
	if !lib.ready || buffer == nil || font == nil || !lib.drawTextProc.Valid() {
		return false
	}
	if text == "" {
		return true
	}

	textPtr, textAddr := stringAddress(text)
	if textPtr == nil {
		return false
	}

	CallStdcall6Raw(uint32(lib.drawTextProc), buffer.address(), font.address(), textAddr, uint32(x), uint32(y), color)
	freeCString(textPtr)
	return true
}

func (lib Buf2D) DrawTextBytes(buffer *Buf2DImage, font *Buf2DImage, text []byte, x int, y int, color uint32) bool {
	if !lib.ready || buffer == nil || font == nil || !lib.drawTextProc.Valid() {
		return false
	}
	if len(text) == 0 {
		return true
	}

	data := make([]byte, len(text)+1)
	copy(data, text)
	CallStdcall6Raw(uint32(lib.drawTextProc), buffer.address(), font.address(), byteSliceAddress(data), uint32(x), uint32(y), color)
	_ = data
	return true
}

func (lib Buf2D) CropColor(buffer *Buf2DImage, color uint32, options Buf2DCropOption) bool {
	if !lib.ready || buffer == nil || !lib.cropColorProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.cropColorProc), buffer.address(), color, uint32(options))
	return true
}

func (lib Buf2D) OffsetH(buffer *Buf2DImage, offset int, top int, height int) bool {
	if !lib.ready || buffer == nil || !lib.offsetHProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.offsetHProc), buffer.address(), uint32(offset), uint32(top), uint32(height))
	return true
}

func (lib Buf2D) FloodFill(buffer *Buf2DImage, x int, y int, mode int, color uint32, altColor uint32) bool {
	if !lib.ready || buffer == nil || !lib.floodFillProc.Valid() {
		return false
	}

	CallStdcall6Raw(uint32(lib.floodFillProc), buffer.address(), uint32(x), uint32(y), uint32(mode), color, altColor)
	return true
}

func (lib Buf2D) SetPixel(buffer *Buf2DImage, x int, y int, color uint32) bool {
	if !lib.ready || buffer == nil || !lib.setPixelProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.setPixelProc), buffer.address(), uint32(x), uint32(y), color)
	return true
}

func (lib Buf2D) GetPixel(buffer *Buf2DImage, x int, y int) uint32 {
	if !lib.ready || buffer == nil || !lib.getPixelProc.Valid() {
		return 0
	}

	return CallStdcall3Raw(uint32(lib.getPixelProc), buffer.address(), uint32(x), uint32(y))
}

func (lib Buf2D) FlipHorizontal(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.flipHProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.flipHProc), buffer.address())
	return true
}

func (lib Buf2D) FlipVertical(buffer *Buf2DImage) bool {
	if !lib.ready || buffer == nil || !lib.flipVProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.flipVProc), buffer.address())
	return true
}

func (lib Buf2D) FilterDither(buffer *Buf2DImage, algorithm Buf2DDitherAlgorithm) bool {
	if !lib.ready || buffer == nil || !lib.filterDitherProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.filterDitherProc), buffer.address(), uint32(algorithm))
	return true
}

func (buffer *Buf2DImage) address() uint32 {
	if buffer == nil {
		return 0
	}

	return pointerValue((*byte)(unsafe.Pointer(buffer)))
}

func buf2dClampUint16(value int) uint16 {
	if value < 0 {
		return 0
	}
	if value > 0xFFFF {
		return 0xFFFF
	}
	return uint16(value)
}

func buf2dClampUint32(value int) uint32 {
	if value < 0 {
		return 0
	}
	return uint32(value)
}
