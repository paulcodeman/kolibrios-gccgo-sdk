package kos

const INIDLLPath = "/sys/lib/libini.obj"

const iniDefaultValueBufferSize = 4096

type INI struct {
	table         DLLExportTable
	getStringProc DLLProc
	setStringProc DLLProc
	getIntProc    DLLProc
	setIntProc    DLLProc
	ready         bool
}

type iniDocument struct {
	sections []iniSection
}

type iniSection struct {
	name    string
	entries []iniEntry
}

type iniEntry struct {
	key   string
	value string
}

func LoadINIDLL() DLLExportTable {
	return LoadDLLFile(INIDLLPath)
}

func LoadINI() (INI, bool) {
	return LoadINIFromDLL(LoadINIDLL())
}

func LoadINIFromDLL(table DLLExportTable) (INI, bool) {
	ini := INI{
		table:         table,
		getStringProc: LookupDLLExportAny(table, "ini_get_str"),
		setStringProc: LookupDLLExportAny(table, "ini_set_str"),
		getIntProc:    LookupDLLExportAny(table, "ini_get_int"),
		setIntProc:    LookupDLLExportAny(table, "ini_set_int"),
		ready:         true,
	}
	initProc := LookupDLLExportAny(table, "lib_init")
	if !ini.Valid() {
		return INI{}, false
	}
	if initProc.Valid() && InitDLLLibraryRaw(uint32(initProc)) != 0 {
		return INI{}, false
	}

	return ini, true
}

func (ini INI) Valid() bool {
	return ini.table != 0 &&
		ini.getStringProc.Valid() &&
		ini.setStringProc.Valid() &&
		ini.getIntProc.Valid() &&
		ini.setIntProc.Valid()
}

func (ini INI) ExportTable() DLLExportTable {
	return ini.table
}

func (ini INI) Ready() bool {
	return ini.ready
}

func (ini INI) GetString(path string, section string, key string, defaultValue string) (string, bool) {
	return ini.GetStringSized(path, section, key, defaultValue, iniDefaultValueBufferSize)
}

func (ini INI) GetStringSized(path string, section string, key string, defaultValue string, bufferSize uint32) (string, bool) {
	if !ini.ready {
		return defaultValue, false
	}
	if bufferSize == 0 {
		bufferSize = 1
	}

	document, ok := ini.loadDocument(path)
	if !ok {
		return clampINIString(defaultValue, bufferSize), false
	}

	value, found := document.lookup(section, key)
	if !found {
		return clampINIString(defaultValue, bufferSize), false
	}

	return clampINIString(value, bufferSize), true
}

func (ini INI) SetString(path string, section string, key string, value string) bool {
	if !ini.ready {
		return false
	}

	document, _ := ini.loadDocument(path)
	document.set(section, key, value)
	encoded := document.encode()
	written, status := CreateOrRewriteFile(path, encoded)
	return status == FileSystemOK && written == uint32(len(encoded))
}

func (ini INI) GetInt(path string, section string, key string, defaultValue int32) int32 {
	if !ini.ready {
		return defaultValue
	}

	document, ok := ini.loadDocument(path)
	if !ok {
		return defaultValue
	}

	value, found := document.lookup(section, key)
	if !found {
		return defaultValue
	}

	parsed, parsedOK := parseINIInt32(value)
	if !parsedOK {
		return defaultValue
	}

	return parsed
}

func (ini INI) SetInt(path string, section string, key string, value int32) bool {
	return ini.SetString(path, section, key, formatINIInt32(value))
}

func (ini INI) loadDocument(path string) (iniDocument, bool) {
	data, status := ReadAllFile(path)
	if status != FileSystemOK && status != FileSystemEOF {
		return iniDocument{}, false
	}

	return parseINIDocument(data), true
}

func parseINIDocument(data []byte) iniDocument {
	document := iniDocument{}
	currentSection := ""

	start := 0
	for start <= len(data) {
		end := start
		for end < len(data) && data[end] != '\n' {
			end++
		}

		line := trimINISpace(bytesToINIString(data[start:end]))
		if line == "" || line[0] == ';' {
			if end >= len(data) {
				break
			}
			start = end + 1
			continue
		}
		if line[0] == '[' {
			sectionEnd := indexByteInString(line, ']')
			if sectionEnd <= 1 {
				if end >= len(data) {
					break
				}
				start = end + 1
				continue
			}

			currentSection = trimINISpace(line[1:sectionEnd])
			document.ensureSection(currentSection)
		} else {
			split := indexByteInString(line, '=')
			if split > 0 {
				key := trimINISpace(line[:split])
				value := trimINISpace(line[split+1:])
				if key != "" {
					document.set(currentSection, key, value)
				}
			}
		}

		if end >= len(data) {
			break
		}
		start = end + 1
	}

	return document
}

func (document *iniDocument) ensureSection(name string) *iniSection {
	for index := 0; index < len(document.sections); index++ {
		if document.sections[index].name == name {
			return &document.sections[index]
		}
	}

	document.sections = append(document.sections, iniSection{name: name})
	return &document.sections[len(document.sections)-1]
}

func (document *iniDocument) set(section string, key string, value string) {
	target := document.ensureSection(section)
	for index := 0; index < len(target.entries); index++ {
		if target.entries[index].key == key {
			target.entries[index].value = value
			return
		}
	}

	target.entries = append(target.entries, iniEntry{key: key, value: value})
}

func (document iniDocument) lookup(section string, key string) (string, bool) {
	for sectionIndex := 0; sectionIndex < len(document.sections); sectionIndex++ {
		if document.sections[sectionIndex].name != section {
			continue
		}

		for entryIndex := 0; entryIndex < len(document.sections[sectionIndex].entries); entryIndex++ {
			entry := document.sections[sectionIndex].entries[entryIndex]
			if entry.key == key {
				return entry.value, true
			}
		}

		return "", false
	}

	return "", false
}

func (document iniDocument) encode() []byte {
	if len(document.sections) == 0 {
		return []byte{}
	}

	buffer := make([]byte, 0, 128)
	for sectionIndex := 0; sectionIndex < len(document.sections); sectionIndex++ {
		section := document.sections[sectionIndex]
		if sectionIndex != 0 {
			buffer = append(buffer, '\r', '\n')
		}
		if section.name != "" {
			buffer = append(buffer, '[')
			buffer = append(buffer, section.name...)
			buffer = append(buffer, ']', '\r', '\n')
		}
		for entryIndex := 0; entryIndex < len(section.entries); entryIndex++ {
			buffer = append(buffer, section.entries[entryIndex].key...)
			buffer = append(buffer, '=')
			buffer = append(buffer, section.entries[entryIndex].value...)
			buffer = append(buffer, '\r', '\n')
		}
	}

	return buffer
}

func trimINISpace(value string) string {
	start := 0
	for start < len(value) && isINISpace(value[start]) {
		start++
	}

	end := len(value)
	for end > start && isINISpace(value[end-1]) {
		end--
	}

	return value[start:end]
}

func bytesToINIString(value []byte) string {
	end := len(value)
	if end > 0 && value[end-1] == '\r' {
		end--
	}

	return string(value[:end])
}

func indexByteInString(value string, needle byte) int {
	for index := 0; index < len(value); index++ {
		if value[index] == needle {
			return index
		}
	}

	return -1
}

func isINISpace(value byte) bool {
	return value == ' ' || value == '\t'
}

func clampINIString(value string, bufferSize uint32) string {
	if bufferSize <= 1 {
		return ""
	}
	maxLen := int(bufferSize - 1)
	if len(value) <= maxLen {
		return value
	}

	return value[:maxLen]
}

func parseINIInt32(value string) (int32, bool) {
	trimmed := trimINISpace(value)
	if trimmed == "" {
		return 0, false
	}

	sign := int32(1)
	index := 0
	if trimmed[0] == '-' {
		sign = -1
		index = 1
	} else if trimmed[0] == '+' {
		index = 1
	}
	if index >= len(trimmed) {
		return 0, false
	}

	var result int32
	for index < len(trimmed) {
		ch := trimmed[index]
		if ch < '0' || ch > '9' {
			return 0, false
		}
		result = result*10 + int32(ch-'0')
		index++
	}

	return result * sign, true
}

func formatINIInt32(value int32) string {
	if value == 0 {
		return "0"
	}
	if value < 0 {
		return "-" + formatINIUint32(uint32(-value))
	}

	return formatINIUint32(uint32(value))
}

func formatINIUint32(value uint32) string {
	if value < 10 {
		return string([]byte{'0' + byte(value)})
	}

	return formatINIUint32(value/10) + string([]byte{'0' + byte(value%10)})
}

func cStringBufferToString(buffer []byte) string {
	limit := 0
	for limit < len(buffer) && buffer[limit] != 0 {
		limit++
	}

	return string(buffer[:limit])
}
