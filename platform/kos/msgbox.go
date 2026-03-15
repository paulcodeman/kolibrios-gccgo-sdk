package kos

const MsgBoxDLLPath = "/sys/lib/msgbox.obj"

const (
	msgBoxTextsOffset    = 2
	msgBoxTextsSize      = 2048
	msgBoxStackSize      = 1024
	msgBoxTopStackOffset = msgBoxTextsOffset + msgBoxTextsSize + msgBoxStackSize
	msgBoxDataSize       = msgBoxTopStackOffset + 4
	MsgBoxMaxButtons     = 8
)

type MsgBox struct {
	table      DLLExportTable
	createProc DLLProc
	reinitProc DLLProc
	ready      bool
}

type MsgBoxData struct {
	raw [msgBoxDataSize]byte
}

func LoadMsgBoxDLL() DLLExportTable {
	return LoadDLLFile(MsgBoxDLLPath)
}

func LoadMsgBox() (MsgBox, bool) {
	return LoadMsgBoxFromDLL(LoadMsgBoxDLL())
}

func LoadMsgBoxFromDLL(table DLLExportTable) (MsgBox, bool) {
	msgbox := MsgBox{
		table:      table,
		createProc: LookupDLLExportAny(table, "mb_create"),
		reinitProc: LookupDLLExportAny(table, "mb_reinit"),
		ready:      true,
	}
	if !msgbox.Valid() {
		return MsgBox{}, false
	}

	return msgbox, true
}

func NewMsgBox(title string, text string, defaultButton uint8, buttons ...string) (MsgBoxData, bool) {
	var box MsgBoxData

	if len(buttons) > MsgBoxMaxButtons {
		return MsgBoxData{}, false
	}

	box.raw[0] = defaultButton
	cursor := msgBoxTextsOffset
	var ok bool
	cursor, ok = box.appendText(cursor, title)
	if !ok {
		return MsgBoxData{}, false
	}
	cursor, ok = box.appendText(cursor, text)
	if !ok {
		return MsgBoxData{}, false
	}

	for index := 0; index < len(buttons); index++ {
		cursor, ok = box.appendText(cursor, buttons[index])
		if !ok {
			return MsgBoxData{}, false
		}
	}

	return box, true
}

func (msgbox MsgBox) Valid() bool {
	return msgbox.table != 0 &&
		msgbox.createProc.Valid() &&
		msgbox.reinitProc.Valid()
}

func (msgbox MsgBox) ExportTable() DLLExportTable {
	return msgbox.table
}

func (msgbox MsgBox) Ready() bool {
	return msgbox.ready
}

func (msgbox MsgBox) Start(box *MsgBoxData) bool {
	if !msgbox.ready || box == nil || !msgbox.createProc.Valid() {
		return false
	}

	CallStdcall2VoidRaw(uint32(msgbox.createProc), box.address(), box.stackTopAddress())
	return true
}

func (msgbox MsgBox) Reinit(box *MsgBoxData) bool {
	if !msgbox.ready || box == nil || !msgbox.reinitProc.Valid() {
		return false
	}

	CallStdcall1VoidRaw(uint32(msgbox.reinitProc), box.address())
	return true
}

func (box *MsgBoxData) Result() uint8 {
	if box == nil {
		return 0
	}

	return box.raw[0]
}

func (box *MsgBoxData) SetDefaultButton(button uint8) {
	if box != nil {
		box.raw[0] = button
	}
}

func (box *MsgBoxData) address() uint32 {
	if box == nil {
		return 0
	}

	return byteSliceAddress(box.raw[:])
}

func (box *MsgBoxData) stackTopAddress() uint32 {
	addr := box.address()
	if addr == 0 {
		return 0
	}

	return addr + msgBoxTopStackOffset
}

func (box *MsgBoxData) appendText(cursor int, value string) (int, bool) {
	limit := msgBoxTextsOffset + msgBoxTextsSize
	end := cursor + len(value) + 1
	if box == nil || cursor < msgBoxTextsOffset || end > limit {
		return cursor, false
	}

	copy(box.raw[cursor:end-1], value)
	box.raw[end-1] = 0
	return end, true
}
