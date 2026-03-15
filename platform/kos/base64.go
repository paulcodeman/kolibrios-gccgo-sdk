package kos

const Base64DLLPath = "/sys/lib/base64.obj"

type Base64 struct {
	table      DLLExportTable
	encodeProc DLLProc
	decodeProc DLLProc
}

func LoadBase64DLL() DLLExportTable {
	return LoadDLLFile(Base64DLLPath)
}

func LoadBase64() (Base64, bool) {
	return LoadBase64FromDLL(LoadBase64DLL())
}

func LoadBase64FromDLL(table DLLExportTable) (Base64, bool) {
	base64 := Base64{
		table:      table,
		encodeProc: LookupDLLExportAny(table, "base64_encode"),
		decodeProc: LookupDLLExportAny(table, "base64_decode"),
	}
	if !base64.Valid() {
		return Base64{}, false
	}

	return base64, true
}

func (base64 Base64) Valid() bool {
	return base64.table != 0 && base64.encodeProc.Valid() && base64.decodeProc.Valid()
}

func (base64 Base64) ExportTable() DLLExportTable {
	return base64.table
}

func (base64 Base64) Encode(dst []byte, src []byte) (int, bool) {
	return base64.transform(base64.encodeProc, dst, src)
}

func (base64 Base64) Decode(dst []byte, src []byte) (int, bool) {
	return base64.transform(base64.decodeProc, dst, src)
}

func (base64 Base64) transform(proc DLLProc, dst []byte, src []byte) (int, bool) {
	if !proc.Valid() {
		return 0, false
	}

	tempSize := len(dst) + 1
	if tempSize == 0 {
		tempSize = 1
	}

	temp := make([]byte, tempSize)
	length := int(CallStdcall3Raw(uint32(proc), byteSliceAddress(src), byteSliceAddress(temp), uint32(len(src))))
	if length < 0 || length > len(dst) {
		return 0, false
	}

	copy(dst, temp[:length])
	return length, true
}
