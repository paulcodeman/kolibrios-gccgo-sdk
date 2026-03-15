package kos

type DLLExportTable uint32
type DLLProc uint32

const ConsoleDLLPath = "/sys/lib/console.obj"

func LoadDLLFile(path string) DLLExportTable {
	return LoadDLLFileWithEncoding(path, EncodingUTF8)
}

func LoadDLLFileWithEncoding(path string, encoding StringEncoding) DLLExportTable {
	return DLLExportTable(LoadDLLWithEncoding(encoding, path))
}

func LoadDLLFileLegacy(path string) DLLExportTable {
	return DLLExportTable(LoadDLL(path))
}

func LoadConsoleDLL() DLLExportTable {
	return LoadDLLFile(ConsoleDLLPath)
}

func LoadDLLInitialized(path string) (DLLExportTable, bool) {
	table := LoadDLLFile(path)
	if table == 0 {
		return 0, false
	}
	if !InitDLLLibrary(table) {
		return 0, false
	}

	return table, true
}

func InitDLLLibrary(table DLLExportTable) bool {
	if table == 0 {
		return false
	}

	initProc := table.Lookup("lib_init")
	if !initProc.Valid() {
		return true
	}

	InitDLLLibraryRaw(uint32(initProc))
	return true
}

func LookupDLLExport(table DLLExportTable, name string) DLLProc {
	namePtr, _ := stringAddress(name)
	if table == 0 || namePtr == nil {
		return 0
	}

	proc := DLLProc(LookupDLLExportRaw(uint32(table), namePtr))
	freeCString(namePtr)
	return proc
}

func (table DLLExportTable) Lookup(name string) DLLProc {
	return LookupDLLExport(table, name)
}

func LookupDLLExportAny(table DLLExportTable, names ...string) DLLProc {
	for index := 0; index < len(names); index++ {
		proc := LookupDLLExport(table, names[index])
		if proc.Valid() {
			return proc
		}
	}

	return 0
}

func (proc DLLProc) Valid() bool {
	return proc != 0
}
