package kos

const ArchiverDLLPath = "/sys/lib/archiver.obj"

type Archiver struct {
	table             DLLExportTable
	deflateUnpackProc DLLProc
	version           uint32
}

func LoadArchiverDLL() DLLExportTable {
	return LoadDLLFile(ArchiverDLLPath)
}

func LoadArchiver() (Archiver, bool) {
	return LoadArchiverFromDLL(LoadArchiverDLL())
}

func LoadArchiverFromDLL(table DLLExportTable) (Archiver, bool) {
	lib := Archiver{
		table:             table,
		deflateUnpackProc: LookupDLLExportAny(table, "deflate_unpack"),
		version:           uint32(LookupDLLExportAny(table, "version")),
	}
	if !lib.Valid() {
		return Archiver{}, false
	}

	return lib, true
}

func (lib Archiver) Valid() bool {
	return lib.table != 0 &&
		lib.deflateUnpackProc.Valid()
}

func (lib Archiver) ExportTable() DLLExportTable {
	return lib.table
}

func (lib Archiver) Version() uint32 {
	return lib.version
}

func (lib Archiver) DeflateUnpack(data []byte) ([]byte, bool) {
	var packedLength [4]byte
	var unpackedPtr uint32
	var unpackedLength uint32
	var unpacked []byte

	if !lib.Valid() || len(data) == 0 {
		return nil, false
	}

	putUint32LE(packedLength[:], 0, uint32(len(data)))
	unpackedPtr = CallStdcall2Raw(uint32(lib.deflateUnpackProc), byteSliceAddress(data), byteSliceAddress(packedLength[:]))
	if unpackedPtr == 0 {
		return nil, false
	}

	unpackedLength = littleEndianUint32(packedLength[:], 0)
	unpacked = CopyBytesRaw(unpackedPtr, unpackedLength)
	HeapFreeRaw(unpackedPtr)
	return unpacked, true
}
