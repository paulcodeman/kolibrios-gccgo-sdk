package kos

const ConsoleDefaultDimension = ^uint32(0)
const ConsoleDLLStart = 1

type Console struct {
	table           DLLExportTable
	startProc       DLLProc
	initProc        DLLProc
	writeStringProc DLLProc
	exitProc        DLLProc
	setTitleProc    DLLProc
	getchProc       DLLProc
	getsProc        DLLProc
	keyHitProc      DLLProc
	version         uint32
}

var activeConsole Console

func LoadConsole() (Console, bool) {
	return LoadConsoleFromDLL(LoadConsoleDLL())
}

func LoadConsoleFromDLL(table DLLExportTable) (Console, bool) {
	console := Console{
		table:           table,
		startProc:       table.Lookup("START"),
		initProc:        table.Lookup("con_init"),
		writeStringProc: table.Lookup("con_write_string"),
		exitProc:        table.Lookup("con_exit"),
		setTitleProc:    table.Lookup("con_set_title"),
		getchProc:       table.Lookup("con_getch"),
		getsProc:        table.Lookup("con_gets"),
		keyHitProc:      table.Lookup("con_kbhit"),
		version:         uint32(table.Lookup("version")),
	}
	if !console.Valid() {
		return Console{}, false
	}
	console.start()

	return console, true
}

func OpenConsole(title string) (Console, bool) {
	console, ok := LoadConsole()
	if !ok {
		return Console{}, false
	}
	if !console.InitDefault(title) {
		return Console{}, false
	}

	return console, true
}

func (console Console) ExportTable() DLLExportTable {
	return console.table
}

func (console Console) Valid() bool {
	return console.table != 0 &&
		console.initProc.Valid() &&
		console.writeStringProc.Valid() &&
		console.exitProc.Valid()
}

func (console Console) SupportsTitle() bool {
	return console.setTitleProc.Valid()
}

func (console Console) Version() uint32 {
	return console.version
}

func (console Console) SupportsInput() bool {
	return console.getchProc.Valid()
}

func (console Console) SupportsLineInput() bool {
	return console.getsProc.Valid()
}

func (console Console) start() {
	if console.startProc.Valid() {
		CallStdcall1VoidRaw(uint32(console.startProc), ConsoleDLLStart)
	}
}

func (console Console) Init(windowWidth uint32, windowHeight uint32, scrollWidth uint32, scrollHeight uint32, title string) bool {
	titlePtr, titleAddr := stringAddress(title)
	if !console.Valid() || titlePtr == nil {
		return false
	}

	CallStdcall5VoidRaw(uint32(console.initProc), windowWidth, windowHeight, scrollWidth, scrollHeight, titleAddr)
	freeCString(titlePtr)
	registerActiveConsole(console)
	return true
}

func (console Console) InitDefault(title string) bool {
	return console.Init(
		ConsoleDefaultDimension,
		ConsoleDefaultDimension,
		ConsoleDefaultDimension,
		ConsoleDefaultDimension,
		title,
	)
}

func (console Console) SetTitle(title string) bool {
	titlePtr, titleAddr := stringAddress(title)
	if !console.SupportsTitle() || titlePtr == nil {
		return false
	}

	CallStdcall1VoidRaw(uint32(console.setTitleProc), titleAddr)
	freeCString(titlePtr)
	return true
}

func (console Console) WriteString(text string) bool {
	textPtr, textAddr := stringAddress(text)
	if !console.Valid() || textPtr == nil {
		return false
	}

	CallStdcall2VoidRaw(uint32(console.writeStringProc), textAddr, uint32(len(text)))
	freeCString(textPtr)
	return true
}

func (console Console) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if console.WriteString(string(data)) {
		return len(data), nil
	}

	return 0, &consoleError{text: "console write failed"}
}

func (console Console) ReadLine(buffer []byte) (int, error) {
	if !console.SupportsLineInput() || len(buffer) < 2 {
		return 0, &consoleError{text: "console line read failed"}
	}
	if CallStdcall2Raw(uint32(console.getsProc), byteSliceAddress(buffer), uint32(len(buffer))) == 0 {
		return 0, &consoleError{text: "console line read failed"}
	}

	return consoleLineLength(buffer), nil
}

func HasActiveConsole() bool {
	return ConsoleBridgeReadyRaw() != 0
}

func WriteActiveConsole(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if !HasActiveConsole() {
		return 0, &consoleError{text: "active console unavailable"}
	}
	if ConsoleBridgeWriteRaw(byteSliceAddress(data), uint32(len(data))) == 0 {
		return 0, &consoleError{text: "console write failed"}
	}

	return len(data), nil
}

func ReadActiveConsoleLine(buffer []byte) (int, error) {
	if len(buffer) < 2 {
		return 0, &consoleError{text: "active console line read failed"}
	}
	if !HasActiveConsole() {
		return 0, &consoleError{text: "active console unavailable"}
	}
	if ConsoleBridgeReadLineRaw(byteSliceAddress(buffer), uint32(len(buffer))) == 0 {
		return 0, &consoleError{text: "active console line read failed"}
	}

	return consoleLineLength(buffer), nil
}

func (console Console) KeyHit() bool {
	return console.keyHitProc.Valid() && CallStdcall0Raw(uint32(console.keyHitProc)) != 0
}

func (console Console) Getch() int {
	if !console.SupportsInput() {
		return 0
	}

	return int(int32(CallStdcall0Raw(uint32(console.getchProc))))
}

func (console Console) Close() error {
	if !console.Valid() {
		return &consoleError{text: "console close failed"}
	}

	console.Exit(true)
	return nil
}

func (console Console) Exit(closeWindow bool) {
	if !console.exitProc.Valid() {
		return
	}

	CallStdcall1VoidRaw(uint32(console.exitProc), boolToUint32(closeWindow))
	unregisterActiveConsole(console)
}

func boolToUint32(value bool) uint32 {
	if value {
		return 1
	}

	return 0
}

type consoleError struct {
	text string
}

func (err *consoleError) Error() string {
	return err.text
}

func consoleLineLength(buffer []byte) int {
	for index := 0; index < len(buffer); index++ {
		if buffer[index] == 0 {
			return index
		}
	}

	return len(buffer)
}

func registerActiveConsole(console Console) {
	activeConsole = console
	ConsoleBridgeSetRaw(uint32(console.table), uint32(console.writeStringProc), uint32(console.exitProc), uint32(console.getsProc))
}

func unregisterActiveConsole(console Console) {
	ConsoleBridgeClearRaw(uint32(console.table))
	if activeConsole.table == console.table {
		activeConsole = Console{}
	}
}

func closeActiveConsole(closeWindow bool) {
	if !HasActiveConsole() {
		return
	}

	ConsoleBridgeCloseRaw(boolToUint32(closeWindow))
	activeConsole = Console{}
}
