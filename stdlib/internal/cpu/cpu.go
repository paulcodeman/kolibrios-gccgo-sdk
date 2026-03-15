package cpu

const CacheLinePadSize = 64

// DebugOptions indicates whether early GODEBUG parsing is supported.
// We keep it false by default.
var DebugOptions bool

type CacheLinePad struct{ _ [CacheLinePadSize]byte }

var CacheLineSize uintptr = CacheLinePadSize

var X86 struct {
	_            CacheLinePad
	HasAES       bool
	HasADX       bool
	HasAVX       bool
	HasAVX2      bool
	HasBMI1      bool
	HasBMI2      bool
	HasERMS      bool
	HasFMA       bool
	HasOSXSAVE   bool
	HasPCLMULQDQ bool
	HasPOPCNT    bool
	HasRDTSCP    bool
	HasSSE3      bool
	HasSSSE3     bool
	HasSSE41     bool
	HasSSE42     bool
	_            CacheLinePad
}

var ARM struct {
	_        CacheLinePad
	HasVFPv4 bool
	HasIDIVA bool
	_        CacheLinePad
}

var ARM64 struct {
	_            CacheLinePad
	HasAES       bool
	HasPMULL     bool
	HasSHA1      bool
	HasSHA2      bool
	HasCRC32     bool
	HasATOMICS   bool
	HasCPUID     bool
	IsNeoverseN1 bool
	IsZeus       bool
	_            CacheLinePad
}

var MIPS64X struct {
	_      CacheLinePad
	HasMSA bool
	_      CacheLinePad
}

var PPC64 struct {
	_        CacheLinePad
	HasDARN  bool
	HasSCV   bool
	IsPOWER8 bool
	IsPOWER9 bool
	_        CacheLinePad
}

var S390X struct {
	_         CacheLinePad
	HasZARCH  bool
	HasSTFLE  bool
	HasLDISP  bool
	HasEIMM   bool
	HasDFP    bool
	HasETF3EH bool
	HasMSA    bool
	HasAES    bool
	HasAESCBC bool
	HasAESCTR bool
	HasAESGCM bool
	HasGHASH  bool
	HasSHA1   bool
	HasSHA256 bool
	HasSHA512 bool
	HasSHA3   bool
	HasVX     bool
	HasVXE    bool
	HasKDSA   bool
	HasECDSA  bool
	HasEDDSA  bool
	_         CacheLinePad
}

// Initialize is a no-op stub for KolibriOS.
func Initialize(env string) {}
