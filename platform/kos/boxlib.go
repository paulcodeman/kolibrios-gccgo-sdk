package kos

const BoxLibDLLPath = "/sys/lib/box_lib.obj"

const (
	editBoxRawSize     = 76
	scrollBarRawSize   = 84
	progressBarRawSize = 44

	editBoxOffsetWidth            = 0
	editBoxOffsetLeft             = 4
	editBoxOffsetTop              = 8
	editBoxOffsetColor            = 12
	editBoxOffsetShiftColor       = 16
	editBoxOffsetFocusBorderColor = 20
	editBoxOffsetBlurBorderColor  = 24
	editBoxOffsetTextColor        = 28
	editBoxOffsetMax              = 32
	editBoxOffsetText             = 36
	editBoxOffsetMouseVariable    = 40
	editBoxOffsetFlags            = 44
	editBoxOffsetSize             = 48
	editBoxOffsetPosition         = 52
	editBoxOffsetOffset           = 56
	editBoxOffsetCursorX          = 60
	editBoxOffsetCursorY          = 62
	editBoxOffsetShift            = 64
	editBoxOffsetShiftOld         = 66
	editBoxOffsetHeight           = 68
	editBoxOffsetCharWidth        = 72

	scrollBarOffsetXSize      = 0
	scrollBarOffsetXPos       = 2
	scrollBarOffsetYSize      = 4
	scrollBarOffsetYPos       = 6
	scrollBarOffsetButtonSize = 8
	scrollBarOffsetType       = 12
	scrollBarOffsetMaxArea    = 16
	scrollBarOffsetCurArea    = 20
	scrollBarOffsetPosition   = 24
	scrollBarOffsetBackColor  = 28
	scrollBarOffsetFrontColor = 32
	scrollBarOffsetLineColor  = 36
	scrollBarOffsetRedraw     = 40
	scrollBarOffsetAllRedraw  = 76
	scrollBarOffsetArOffset   = 80

	progressBarOffsetValue         = 0
	progressBarOffsetLeft          = 4
	progressBarOffsetTop           = 8
	progressBarOffsetWidth         = 12
	progressBarOffsetHeight        = 16
	progressBarOffsetStyle         = 20
	progressBarOffsetMin           = 24
	progressBarOffsetMax           = 28
	progressBarOffsetBackColor     = 32
	progressBarOffsetProgressColor = 36
	progressBarOffsetFrameColor    = 40
)

const (
	EditBoxFlagPassword    = 1
	EditBoxFlagFocus       = 1 << 1
	EditBoxFlagShift       = 1 << 2
	EditBoxFlagShiftOn     = 1 << 3
	EditBoxFlagShiftBuffer = 1 << 4
	EditBoxFlagInsert      = 1 << 7
	EditBoxFlagMouseOn     = 1 << 8
	EditBoxFlagCtrlOn      = 1 << 9
	EditBoxFlagAltOn       = 1 << 10
	EditBoxFlagDisabled    = 1 << 11
	EditBoxFlagAlwaysFocus = 1 << 14
	EditBoxFlagFigureOnly  = 1 << 15
)

type BoxLib struct {
	table             DLLExportTable
	editDrawProc      DLLProc
	editKeySafeProc   DLLProc
	editMouseProc     DLLProc
	editSetTextProc   DLLProc
	scrollVDrawProc   DLLProc
	scrollVMouseProc  DLLProc
	scrollHDrawProc   DLLProc
	scrollHMouseProc  DLLProc
	progressDrawProc  DLLProc
	progressStepProc  DLLProc
	version           uint32
	editVersion       uint32
	scrollbarVersion  uint32
	progressAvailable bool
	ready             bool
}

type EditBox struct {
	raw        [editBoxRawSize]byte
	textBuffer []byte
	mouseState [4]byte
}

type ScrollBar struct {
	raw [scrollBarRawSize]byte
}

type ProgressBar struct {
	raw [progressBarRawSize]byte
}

func LoadBoxLibDLL() DLLExportTable {
	return LoadDLLFile(BoxLibDLLPath)
}

func LoadBoxLib() (BoxLib, bool) {
	return LoadBoxLibFromDLL(LoadBoxLibDLL())
}

func LoadBoxLibFromDLL(table DLLExportTable) (BoxLib, bool) {
	lib := BoxLib{
		table:             table,
		editDrawProc:      LookupDLLExportAny(table, "edit_box_draw"),
		editKeySafeProc:   LookupDLLExportAny(table, "edit_box_key_safe"),
		editMouseProc:     LookupDLLExportAny(table, "edit_box_mouse"),
		editSetTextProc:   LookupDLLExportAny(table, "edit_box_set_text"),
		scrollVDrawProc:   LookupDLLExportAny(table, "scrollbar_v_draw"),
		scrollVMouseProc:  LookupDLLExportAny(table, "scrollbar_v_mouse"),
		scrollHDrawProc:   LookupDLLExportAny(table, "scrollbar_h_draw"),
		scrollHMouseProc:  LookupDLLExportAny(table, "scrollbar_h_mouse"),
		progressDrawProc:  LookupDLLExportAny(table, "progressbar_draw"),
		progressStepProc:  LookupDLLExportAny(table, "progressbar_progress"),
		version:           uint32(LookupDLLExportAny(table, "version")),
		editVersion:       uint32(LookupDLLExportAny(table, "version_ed")),
		scrollbarVersion:  uint32(LookupDLLExportAny(table, "version_scrollbar")),
		progressAvailable: LookupDLLExportAny(table, "progressbar_draw").Valid() && LookupDLLExportAny(table, "progressbar_progress").Valid(),
		ready:             true,
	}
	initProc := LookupDLLExportAny(table, "lib_init")
	if !lib.Valid() {
		return BoxLib{}, false
	}
	if initProc.Valid() {
		InitDLLLibraryRaw(uint32(initProc))
	}

	return lib, true
}

func NewEditBox(x int, y int, width int, maxChars int, text string) *EditBox {
	if maxChars < 1 {
		maxChars = 1
	}

	box := &EditBox{
		textBuffer: make([]byte, maxChars+1),
	}
	box.setUint32(editBoxOffsetWidth, uint32(width))
	box.setUint32(editBoxOffsetLeft, uint32(x))
	box.setUint32(editBoxOffsetTop, uint32(y))
	box.setUint32(editBoxOffsetColor, 0x00FFFFFF)
	box.setUint32(editBoxOffsetShiftColor, 0x00A4C4E4)
	box.setUint32(editBoxOffsetFocusBorderColor, 0x00A0A0A0)
	box.setUint32(editBoxOffsetBlurBorderColor, 0x00707070)
	box.setUint32(editBoxOffsetTextColor, 0x10000000)
	box.setUint32(editBoxOffsetMax, uint32(maxChars))
	box.setUint32(editBoxOffsetText, byteSliceAddress(box.textBuffer))
	box.setUint32(editBoxOffsetMouseVariable, byteSliceAddress(box.mouseState[:]))
	box.setUint32(editBoxOffsetFlags, EditBoxFlagAlwaysFocus|EditBoxFlagFocus)
	box.SetText(text)
	box.setUint32(editBoxOffsetPosition, uint32(box.Size()))
	return box
}

func NewVerticalScrollBar(x int, y int, width int, height int, maxArea int, curArea int, position int) *ScrollBar {
	bar := &ScrollBar{}
	bar.setUint16(scrollBarOffsetXSize, uint16(width))
	bar.setUint16(scrollBarOffsetXPos, uint16(x))
	bar.setUint16(scrollBarOffsetYSize, uint16(height))
	bar.setUint16(scrollBarOffsetYPos, uint16(y))
	bar.setUint32(scrollBarOffsetButtonSize, 16)
	bar.setUint32(scrollBarOffsetType, 1)
	bar.setUint32(scrollBarOffsetBackColor, 0x00E8E8E8)
	bar.setUint32(scrollBarOffsetFrontColor, 0x006699CC)
	bar.setUint32(scrollBarOffsetLineColor, 0x00404040)
	bar.setUint32(scrollBarOffsetArOffset, 0)
	bar.SetRange(maxArea, curArea)
	bar.SetPosition(position)
	return bar
}

func NewHorizontalScrollBar(x int, y int, width int, height int, maxArea int, curArea int, position int) *ScrollBar {
	return NewVerticalScrollBar(x, y, width, height, maxArea, curArea, position)
}

func NewProgressBar(x int, y int, width int, height int, minValue int, maxValue int, value int) *ProgressBar {
	bar := &ProgressBar{}
	bar.setUint32(progressBarOffsetLeft, uint32(x))
	bar.setUint32(progressBarOffsetTop, uint32(y))
	bar.setUint32(progressBarOffsetWidth, uint32(width))
	bar.setUint32(progressBarOffsetHeight, uint32(height))
	bar.setUint32(progressBarOffsetStyle, 0)
	bar.setUint32(progressBarOffsetMin, uint32(minValue))
	bar.setUint32(progressBarOffsetMax, uint32(maxValue))
	bar.setUint32(progressBarOffsetBackColor, 0x00EAEAEA)
	bar.setUint32(progressBarOffsetProgressColor, 0x0033AA55)
	bar.setUint32(progressBarOffsetFrameColor, 0x00404040)
	bar.SetValue(value)
	return bar
}

func (lib BoxLib) Valid() bool {
	return lib.table != 0 &&
		lib.editDrawProc.Valid() &&
		lib.editKeySafeProc.Valid() &&
		lib.editMouseProc.Valid() &&
		lib.editSetTextProc.Valid() &&
		lib.scrollVDrawProc.Valid() &&
		lib.scrollVMouseProc.Valid() &&
		lib.scrollHDrawProc.Valid() &&
		lib.scrollHMouseProc.Valid() &&
		lib.progressDrawProc.Valid() &&
		lib.progressStepProc.Valid()
}

func (lib BoxLib) ExportTable() DLLExportTable {
	return lib.table
}

func (lib BoxLib) Version() uint32 {
	return lib.version
}

func (lib BoxLib) EditVersion() uint32 {
	return lib.editVersion
}

func (lib BoxLib) ScrollBarVersion() uint32 {
	return lib.scrollbarVersion
}

func (lib BoxLib) Ready() bool {
	return lib.ready
}

func (lib BoxLib) DrawEditBox(box *EditBox) bool {
	if !lib.ready || box == nil || !lib.editDrawProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.editDrawProc), box.address())
	return true
}

func (lib BoxLib) HandleEditBoxKey(box *EditBox, rawKey uint32) bool {
	if !lib.ready || box == nil || !lib.editKeySafeProc.Valid() {
		return false
	}

	CallStdcall2VoidRaw(uint32(lib.editKeySafeProc), box.address(), rawKey)
	return true
}

func (lib BoxLib) HandleEditBoxMouse(box *EditBox) bool {
	if !lib.ready || box == nil || !lib.editMouseProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.editMouseProc), box.address())
	return true
}

func (lib BoxLib) SetEditBoxText(box *EditBox, value string) bool {
	var textPtr *byte
	var textAddr uint32

	if !lib.ready || box == nil || !lib.editSetTextProc.Valid() {
		return false
	}

	textPtr, textAddr = stringAddress(value)
	if textPtr == nil {
		return false
	}

	CallStdcall2VoidRaw(uint32(lib.editSetTextProc), box.address(), textAddr)
	freeCString(textPtr)
	return true
}

func (lib BoxLib) DrawVerticalScrollBar(bar *ScrollBar) bool {
	if !lib.ready || bar == nil || !lib.scrollVDrawProc.Valid() {
		return false
	}

	bar.prepareDraw()
	CallStdcall1VoidRaw(uint32(lib.scrollVDrawProc), bar.address())
	return true
}

func (lib BoxLib) HandleVerticalScrollBarMouse(bar *ScrollBar) bool {
	if !lib.ready || bar == nil || !lib.scrollVMouseProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.scrollVMouseProc), bar.address())
	return true
}

func (lib BoxLib) DrawHorizontalScrollBar(bar *ScrollBar) bool {
	if !lib.ready || bar == nil || !lib.scrollHDrawProc.Valid() {
		return false
	}

	bar.prepareDraw()
	CallStdcall1VoidRaw(uint32(lib.scrollHDrawProc), bar.address())
	return true
}

func (lib BoxLib) HandleHorizontalScrollBarMouse(bar *ScrollBar) bool {
	if !lib.ready || bar == nil || !lib.scrollHMouseProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.scrollHMouseProc), bar.address())
	return true
}

func (lib BoxLib) DrawProgressBar(bar *ProgressBar) bool {
	if !lib.ready || bar == nil || !lib.progressDrawProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.progressDrawProc), bar.address())
	return true
}

func (lib BoxLib) AdvanceProgressBar(bar *ProgressBar) bool {
	if !lib.ready || bar == nil || !lib.progressStepProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(lib.progressStepProc), bar.address())
	return true
}

func (box *EditBox) SetFlags(flags uint32) {
	if box != nil {
		box.setUint32(editBoxOffsetFlags, flags)
	}
}

func (box *EditBox) Flags() uint32 {
	if box == nil {
		return 0
	}

	return littleEndianUint32(box.raw[:], editBoxOffsetFlags)
}

func (box *EditBox) SetText(value string) {
	if box == nil || len(box.textBuffer) == 0 {
		return
	}

	for index := 0; index < len(box.textBuffer); index++ {
		box.textBuffer[index] = 0
	}
	limit := len(value)
	if limit > len(box.textBuffer)-1 {
		limit = len(box.textBuffer) - 1
	}
	copy(box.textBuffer[:limit], value[:limit])
	box.setUint32(editBoxOffsetSize, uint32(limit))
	if box.Position() > limit {
		box.setUint32(editBoxOffsetPosition, uint32(limit))
	}
}

func (box *EditBox) Text() string {
	if box == nil {
		return ""
	}

	return cStringBufferToString(box.textBuffer)
}

func (box *EditBox) Size() int {
	if box == nil {
		return 0
	}

	return int(littleEndianUint32(box.raw[:], editBoxOffsetSize))
}

func (box *EditBox) Position() int {
	if box == nil {
		return 0
	}

	return int(littleEndianUint32(box.raw[:], editBoxOffsetPosition))
}

func (bar *ScrollBar) SetRange(maxArea int, curArea int) {
	if bar == nil {
		return
	}
	if maxArea < 0 {
		maxArea = 0
	}
	if curArea < 0 {
		curArea = 0
	}
	if curArea > maxArea {
		curArea = maxArea
	}

	bar.setUint32(scrollBarOffsetMaxArea, uint32(maxArea))
	bar.setUint32(scrollBarOffsetCurArea, uint32(curArea))
}

func (bar *ScrollBar) SetPosition(position int) {
	if bar == nil {
		return
	}

	maxPosition := int(littleEndianUint32(bar.raw[:], scrollBarOffsetMaxArea)) - int(littleEndianUint32(bar.raw[:], scrollBarOffsetCurArea))
	if maxPosition < 0 {
		maxPosition = 0
	}
	if position < 0 {
		position = 0
	}
	if position > maxPosition {
		position = maxPosition
	}

	bar.setUint32(scrollBarOffsetPosition, uint32(position))
}

func (bar *ScrollBar) Position() int {
	if bar == nil {
		return 0
	}

	return int(littleEndianUint32(bar.raw[:], scrollBarOffsetPosition))
}

func (bar *ProgressBar) SetValue(value int) {
	var minValue int
	var maxValue int

	if bar == nil {
		return
	}

	minValue = int(littleEndianUint32(bar.raw[:], progressBarOffsetMin))
	maxValue = int(littleEndianUint32(bar.raw[:], progressBarOffsetMax))
	if maxValue < minValue {
		maxValue = minValue
	}
	if value < minValue {
		value = minValue
	}
	if value > maxValue {
		value = maxValue
	}

	bar.setUint32(progressBarOffsetValue, uint32(value))
}

func (bar *ProgressBar) Value() int {
	if bar == nil {
		return 0
	}

	return int(littleEndianUint32(bar.raw[:], progressBarOffsetValue))
}

func (box *EditBox) address() uint32 {
	return byteSliceAddress(box.raw[:])
}

func (bar *ScrollBar) address() uint32 {
	return byteSliceAddress(bar.raw[:])
}

func (bar *ProgressBar) address() uint32 {
	return byteSliceAddress(bar.raw[:])
}

func (box *EditBox) setUint32(offset int, value uint32) {
	putUint32LE(box.raw[:], offset, value)
}

func (bar *ScrollBar) setUint16(offset int, value uint16) {
	putUint16LE(bar.raw[:], offset, value)
}

func (bar *ScrollBar) setUint32(offset int, value uint32) {
	putUint32LE(bar.raw[:], offset, value)
}

func (bar *ScrollBar) prepareDraw() {
	bar.setUint32(scrollBarOffsetAllRedraw, 1)
	bar.setUint32(scrollBarOffsetRedraw, 1)
}

func (bar *ProgressBar) setUint32(offset int, value uint32) {
	putUint32LE(bar.raw[:], offset, value)
}
