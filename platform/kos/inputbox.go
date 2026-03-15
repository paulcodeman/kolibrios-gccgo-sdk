package kos

const InputBoxDLLPath = "/sys/lib/inputbox.obj"

type InputBoxFlags uint32
type InputBoxError uint32

const (
	InputBoxString InputBoxFlags = 0
	InputBoxNumber InputBoxFlags = 1

	InputBoxMouseRelative  InputBoxFlags = 0
	InputBoxScreenRelative InputBoxFlags = 8
	InputBoxParentRelative InputBoxFlags = 16
)

const (
	InputBoxNoError InputBoxError = iota
	InputBoxNumberOverflow
	InputBoxResultTooLong
)

type InputBox struct {
	table DLLExportTable
	proc  DLLProc
	ready bool
}

func LoadInputBoxDLL() DLLExportTable {
	return LoadDLLFile(InputBoxDLLPath)
}

func LoadInputBox() (InputBox, bool) {
	return LoadInputBoxFromDLL(LoadInputBoxDLL())
}

func LoadInputBoxFromDLL(table DLLExportTable) (InputBox, bool) {
	inputBox := InputBox{
		table: table,
		proc:  LookupDLLExportAny(table, "InputBox"),
		ready: true,
	}
	if !inputBox.Valid() {
		return InputBox{}, false
	}

	return inputBox, true
}

func (inputBox InputBox) Valid() bool {
	return inputBox.table != 0 && inputBox.proc.Valid()
}

func (inputBox InputBox) ExportTable() DLLExportTable {
	return inputBox.table
}

func (inputBox InputBox) Ready() bool {
	return inputBox.ready
}

func (inputBox InputBox) PromptString(caption string, prompt string, defaultValue string, flags InputBoxFlags, bufferSize uint32) (string, InputBoxError, bool) {
	if bufferSize == 0 {
		bufferSize = 1
	}

	buffer := make([]byte, int(bufferSize))
	status, ok := inputBox.prompt(buffer, caption, prompt, defaultValue, flags&^InputBoxNumber, bufferSize, 0)
	if !ok {
		return "", 0, false
	}

	return cStringBufferToString(buffer), status, true
}

// PromptNumber keeps the default numeric value explicit as decimal text to
// avoid pulling wider integer-formatting helpers into the bootstrap runtime.
func (inputBox InputBox) PromptNumber(caption string, prompt string, defaultValue string, flags InputBoxFlags) (uint64, InputBoxError, bool) {
	var buffer [8]byte

	status, ok := inputBox.prompt(buffer[:], caption, prompt, defaultValue, flags|InputBoxNumber, uint32(len(buffer)), 0)
	if !ok {
		return 0, 0, false
	}

	return littleEndianUint64(buffer[:], 0), status, true
}

func (inputBox InputBox) prompt(buffer []byte, caption string, prompt string, defaultValue string, flags InputBoxFlags, bufferSize uint32, redrawProc uint32) (InputBoxError, bool) {
	if !inputBox.ready || !inputBox.proc.Valid() || len(buffer) == 0 {
		return 0, false
	}

	captionPtr, captionAddr := stringAddress(caption)
	promptPtr, promptAddr := stringAddress(prompt)
	defaultPtr, defaultAddr := stringAddress(defaultValue)
	if captionPtr == nil || promptPtr == nil || defaultPtr == nil {
		freeOptionalCString(captionPtr)
		freeOptionalCString(promptPtr)
		freeOptionalCString(defaultPtr)
		return 0, false
	}

	status := InputBoxError(CallStdcall7Raw(
		uint32(inputBox.proc),
		byteSliceAddress(buffer),
		captionAddr,
		promptAddr,
		defaultAddr,
		uint32(flags),
		bufferSize,
		redrawProc,
	))
	freeCString(captionPtr)
	freeCString(promptPtr)
	freeCString(defaultPtr)
	return status, true
}
