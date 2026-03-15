package kos

const HTTPDLLPath = "/sys/lib/http.obj"

const (
	HTTPFlagHTTP11         HTTPFlags = 1 << 0
	HTTPFlagGotHeader      HTTPFlags = 1 << 1
	HTTPFlagGotAllData     HTTPFlags = 1 << 2
	HTTPFlagContentLength  HTTPFlags = 1 << 3
	HTTPFlagChunked        HTTPFlags = 1 << 4
	HTTPFlagConnected      HTTPFlags = 1 << 5
	HTTPFlagKeepAlive      HTTPFlags = 1 << 8
	HTTPFlagStream         HTTPFlags = 1 << 9
	HTTPFlagReuseBuffer    HTTPFlags = 1 << 10
	HTTPFlagBlock          HTTPFlags = 1 << 11
	HTTPFlagRing           HTTPFlags = 1 << 12
	HTTPFlagInvalidHeader  HTTPFlags = 1 << 16
	HTTPFlagNoRAM          HTTPFlags = 1 << 17
	HTTPFlagSocketError    HTTPFlags = 1 << 18
	HTTPFlagTimeoutError   HTTPFlags = 1 << 19
	HTTPFlagTransferFailed HTTPFlags = 1 << 20
	HTTPFlagNeedMoreSpace  HTTPFlags = 1 << 21
)

type HTTPFlags uint32
type HTTPTransfer uint32

type HTTP struct {
	table            DLLExportTable
	getProc          DLLProc
	headProc         DLLProc
	postProc         DLLProc
	sendProc         DLLProc
	receiveProc      DLLProc
	disconnectProc   DLLProc
	freeProc         DLLProc
	escapeProc       DLLProc
	unescapeProc     DLLProc
	findHeaderProc   DLLProc
	version          uint32
	ready            bool
}

func (flags HTTPFlags) Has(mask HTTPFlags) bool {
	return flags&mask != 0
}

func LoadHTTPDLL() DLLExportTable {
	return LoadDLLFile(HTTPDLLPath)
}

func LoadHTTP() (HTTP, bool) {
	return LoadHTTPFromDLL(LoadHTTPDLL())
}

func LoadHTTPFromDLL(table DLLExportTable) (HTTP, bool) {
	http := HTTP{
		table:          table,
		getProc:        LookupDLLExportAny(table, "get", "http_get"),
		headProc:       LookupDLLExportAny(table, "head", "http_head"),
		postProc:       LookupDLLExportAny(table, "post", "http_post"),
		sendProc:       LookupDLLExportAny(table, "send", "http_send"),
		receiveProc:    LookupDLLExportAny(table, "receive", "http_receive", "process", "http_process"),
		disconnectProc: LookupDLLExportAny(table, "disconnect", "http_disconnect", "stop", "http_stop"),
		freeProc:       LookupDLLExportAny(table, "free", "http_free"),
		escapeProc:     LookupDLLExportAny(table, "escape", "http_escape", "uri_escape"),
		unescapeProc:   LookupDLLExportAny(table, "unescape", "http_unescape", "uri_unescape"),
		findHeaderProc: LookupDLLExportAny(table, "find_header_field", "http_find_header_field"),
		version:        uint32(LookupDLLExportAny(table, "version", "http_version")),
		ready:          true,
	}
	initProc := LookupDLLExportAny(table, "lib_init", "http_lib_init")
	if !http.Valid() {
		return HTTP{}, false
	}
	if initProc.Valid() {
		InitDLLLibraryRaw(uint32(initProc))
	}

	return http, true
}

func (http HTTP) Valid() bool {
	return http.table != 0 &&
		http.getProc.Valid() &&
		http.headProc.Valid() &&
		http.sendProc.Valid() &&
		http.receiveProc.Valid() &&
		http.disconnectProc.Valid() &&
		http.freeProc.Valid() &&
		http.escapeProc.Valid() &&
		http.unescapeProc.Valid()
}

func (http HTTP) ExportTable() DLLExportTable {
	return http.table
}

func (http HTTP) Version() uint32 {
	return http.version
}

func (http HTTP) Ready() bool {
	return http.ready
}

func (http HTTP) Escape(value string) (string, bool) {
	return http.transform(http.escapeProc, value)
}

func (http HTTP) Unescape(value string) (string, bool) {
	return http.transform(http.unescapeProc, value)
}

func (http HTTP) Get(url string, previous HTTPTransfer, flags HTTPFlags, addHeader string) (HTTPTransfer, bool) {
	return http.start(http.getProc, url, previous, flags, addHeader)
}

func (http HTTP) Head(url string, previous HTTPTransfer, flags HTTPFlags, addHeader string) (HTTPTransfer, bool) {
	return http.start(http.headProc, url, previous, flags, addHeader)
}

func (http HTTP) Post(url string, previous HTTPTransfer, flags HTTPFlags, addHeader string, contentType string, contentLength uint32) (HTTPTransfer, bool) {
	if !http.ready || !http.postProc.Valid() {
		return 0, false
	}

	urlPtr, urlAddr := stringAddress(url)
	headerPtr, headerAddr := optionalCStringAddress(addHeader)
	contentTypePtr, contentTypeAddr := stringAddress(contentType)
	if urlPtr == nil || contentTypePtr == nil {
		freeOptionalCString(headerPtr)
		freeOptionalCString(contentTypePtr)
		return 0, false
	}

	transfer := HTTPTransfer(CallStdcall6Raw(
		uint32(http.postProc),
		urlAddr,
		uint32(previous),
		uint32(flags),
		headerAddr,
		contentTypeAddr,
		contentLength,
	))
	freeCString(urlPtr)
	freeOptionalCString(headerPtr)
	freeCString(contentTypePtr)
	return transfer, transfer.Valid()
}

func (http HTTP) Send(transfer HTTPTransfer, data []byte) int {
	if !http.ready || !transfer.Valid() || !http.sendProc.Valid() {
		return -1
	}
	if len(data) == 0 {
		return 0
	}

	return int(int32(CallStdcall3Raw(uint32(http.sendProc), uint32(transfer), byteSliceAddress(data), uint32(len(data)))))
}

func (http HTTP) Receive(transfer HTTPTransfer) int {
	if !http.ready || !transfer.Valid() || !http.receiveProc.Valid() {
		return 0
	}

	return int(int32(CallStdcall1Raw(uint32(http.receiveProc), uint32(transfer))))
}

func (http HTTP) Disconnect(transfer HTTPTransfer) {
	if http.ready && transfer.Valid() && http.disconnectProc.Valid() {
		CallStdcall1VoidRaw(uint32(http.disconnectProc), uint32(transfer))
	}
}

func (http HTTP) Free(transfer HTTPTransfer) {
	if http.ready && transfer.Valid() && http.freeProc.Valid() {
		CallStdcall1VoidRaw(uint32(http.freeProc), uint32(transfer))
	}
}

func (http HTTP) start(proc DLLProc, url string, previous HTTPTransfer, flags HTTPFlags, addHeader string) (HTTPTransfer, bool) {
	if !http.ready {
		return 0, false
	}

	urlPtr, urlAddr := stringAddress(url)
	headerPtr, headerAddr := optionalCStringAddress(addHeader)
	if !proc.Valid() || urlPtr == nil {
		freeOptionalCString(headerPtr)
		return 0, false
	}

	transfer := HTTPTransfer(CallStdcall4Raw(uint32(proc), urlAddr, uint32(previous), uint32(flags), headerAddr))
	freeCString(urlPtr)
	freeOptionalCString(headerPtr)
	return transfer, transfer.Valid()
}

func (http HTTP) transform(proc DLLProc, value string) (string, bool) {
	inputPtr, inputAddr := stringAddress(value)
	if !proc.Valid() || inputPtr == nil {
		return "", false
	}

	resultPtr := CallStdcall2Raw(uint32(proc), inputAddr, uint32(len(value)))
	freeCString(inputPtr)
	if resultPtr == 0 {
		return "", false
	}

	result := CStringToStringRaw(resultPtr)
	HeapFreeRaw(resultPtr)
	return result, true
}

func (transfer HTTPTransfer) Valid() bool {
	return transfer != 0
}

func (transfer HTTPTransfer) Flags() HTTPFlags {
	return HTTPFlags(ReadUint32Raw(uint32(transfer), 4))
}

func (transfer HTTPTransfer) Status() uint32 {
	return ReadUint32Raw(uint32(transfer), 24)
}

func (transfer HTTPTransfer) HeaderLength() uint32 {
	return ReadUint32Raw(uint32(transfer), 28)
}

func (transfer HTTPTransfer) ContentPointer() uint32 {
	return ReadUint32Raw(uint32(transfer), 32)
}

func (transfer HTTPTransfer) ContentLength() uint32 {
	return ReadUint32Raw(uint32(transfer), 36)
}

func (transfer HTTPTransfer) ContentReceived() uint32 {
	return ReadUint32Raw(uint32(transfer), 40)
}

func (transfer HTTPTransfer) HeaderBytes() []byte {
	size := transfer.HeaderLength()
	if transfer == 0 || size == 0 {
		return []byte{}
	}

	return CopyBytesRaw(uint32(transfer)+44, size)
}

func (transfer HTTPTransfer) HeaderString() string {
	return string(transfer.HeaderBytes())
}

func (transfer HTTPTransfer) ContentBytes() []byte {
	ptr := transfer.ContentPointer()
	size := transfer.ContentReceived()
	if transfer == 0 || ptr == 0 || size == 0 {
		return []byte{}
	}

	return CopyBytesRaw(ptr, size)
}
