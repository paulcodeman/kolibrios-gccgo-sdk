package kos

const NetworkDLLPath = "/sys/lib/network.obj"

const (
	NetworkFamilyUnspec = 0
	NetworkFamilyIPv4   = 2
	NetworkFamilyIPv6   = 10
)

const (
	NetworkSockStream = 1
	NetworkSockDgram  = 2
)

const (
	NetworkEAIAddrFamily = 1
	NetworkEAIAgain      = 2
	NetworkEAIBadFlags   = 3
	NetworkEAIFail       = 4
	NetworkEAIFamily     = 5
	NetworkEAIMemory     = 6
	NetworkEAINoname     = 8
	NetworkEAIService    = 9
	NetworkEAISocktype   = 10
	NetworkEAIBadHints   = 12
	NetworkEAIProtocol   = 13
	NetworkEAIOverflow   = 14
)

type Network struct {
	table            DLLExportTable
	inetAddrProc     DLLProc
	inetNtoaProc     DLLProc
	getAddrInfoProc  DLLProc
	freeAddrInfoProc DLLProc
	version          uint32
}

type NetworkAddrInfo struct {
	Flags     uint32
	Family    uint32
	SockType  uint32
	Protocol  uint32
	AddrLen   uint32
	Address   uint32
	AddressIP string
	Port      uint16
	CanonName string
}

type networkError struct {
	op   string
	code int
}

func (err *networkError) Error() string {
	if err == nil {
		return ""
	}

	name := networkErrorName(err.code)
	if name == "" {
		return err.op + ": unexpected failure"
	}

	return err.op + ": " + name
}

func LoadNetworkDLL() DLLExportTable {
	return LoadDLLFile(NetworkDLLPath)
}

func LoadNetwork() (Network, bool) {
	return LoadNetworkFromDLL(LoadNetworkDLL())
}

func LoadNetworkFromDLL(table DLLExportTable) (Network, bool) {
	network := Network{
		table:            table,
		inetAddrProc:     table.Lookup("inet_addr"),
		inetNtoaProc:     table.Lookup("inet_ntoa"),
		getAddrInfoProc:  table.Lookup("getaddrinfo"),
		freeAddrInfoProc: table.Lookup("freeaddrinfo"),
		version:          uint32(table.Lookup("version")),
	}
	if !network.Valid() {
		return Network{}, false
	}
	if !InitDLLLibrary(table) {
		return Network{}, false
	}

	return network, true
}

func (network Network) Valid() bool {
	return network.table != 0 &&
		network.inetAddrProc.Valid() &&
		network.inetNtoaProc.Valid() &&
		network.getAddrInfoProc.Valid() &&
		network.freeAddrInfoProc.Valid()
}

func (network Network) ExportTable() DLLExportTable {
	return network.table
}

func (network Network) Version() uint32 {
	return network.version
}

func (network Network) InetAddr(host string) uint32 {
	hostPtr, hostAddr := stringAddress(host)
	if !network.inetAddrProc.Valid() || hostPtr == nil {
		return ^uint32(0)
	}

	addr := CallStdcall1Raw(uint32(network.inetAddrProc), hostAddr)
	freeCString(hostPtr)
	return addr
}

func (network Network) InetNtoa(addr uint32) string {
	if !network.inetNtoaProc.Valid() {
		return ""
	}

	ptr := CallStdcall1Raw(uint32(network.inetNtoaProc), addr)
	if ptr == 0 {
		return ""
	}

	return CStringToStringRaw(ptr)
}

func (network Network) LookupHost(host string) ([]string, error) {
	items, err := network.GetAddrInfo(host, "")
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(items))
	for index := 0; index < len(items); index++ {
		if items[index].AddressIP == "" {
			continue
		}
		if stringSliceContains(results, items[index].AddressIP) {
			continue
		}
		results = append(results, items[index].AddressIP)
	}

	return results, nil
}

func (network Network) GetAddrInfo(host string, service string) ([]NetworkAddrInfo, error) {
	var resultBuffer [4]byte

	hostPtr, hostAddr := optionalCStringAddress(host)
	servicePtr, serviceAddr := optionalCStringAddress(service)
	if !network.getAddrInfoProc.Valid() || hostPtr == nil && host != "" || servicePtr == nil && service != "" {
		freeOptionalCString(hostPtr)
		freeOptionalCString(servicePtr)
		return nil, &networkError{op: "getaddrinfo", code: NetworkEAIFail}
	}

	code := int(CallStdcall4Raw(
		uint32(network.getAddrInfoProc),
		hostAddr,
		serviceAddr,
		0,
		pointerValue(&resultBuffer[0]),
	))
	freeOptionalCString(hostPtr)
	freeOptionalCString(servicePtr)
	if code != 0 {
		return nil, &networkError{op: "getaddrinfo", code: code}
	}

	resultHead := networkLittleEndianUint32(resultBuffer[:])
	results := network.copyAddrInfoList(resultHead)
	if resultHead != 0 {
		CallStdcall1VoidRaw(uint32(network.freeAddrInfoProc), resultHead)
	}

	return results, nil
}

func networkErrorName(code int) string {
	switch code {
	case NetworkEAIAddrFamily:
		return "EAI_ADDRFAMILY"
	case NetworkEAIAgain:
		return "EAI_AGAIN"
	case NetworkEAIBadFlags:
		return "EAI_BADFLAGS"
	case NetworkEAIFail:
		return "EAI_FAIL"
	case NetworkEAIFamily:
		return "EAI_FAMILY"
	case NetworkEAIMemory:
		return "EAI_MEMORY"
	case NetworkEAINoname:
		return "EAI_NONAME"
	case NetworkEAIService:
		return "EAI_SERVICE"
	case NetworkEAISocktype:
		return "EAI_SOCKTYPE"
	case NetworkEAIBadHints:
		return "EAI_BADHINTS"
	case NetworkEAIProtocol:
		return "EAI_PROTOCOL"
	case NetworkEAIOverflow:
		return "EAI_OVERFLOW"
	}

	return ""
}

func stringSliceContains(values []string, target string) bool {
	for index := 0; index < len(values); index++ {
		if values[index] == target {
			return true
		}
	}

	return false
}

func swap16(value uint16) uint16 {
	return (value << 8) | (value >> 8)
}

func (network Network) copyAddrInfoList(head uint32) []NetworkAddrInfo {
	results := []NetworkAddrInfo{}

	for cursor := head; cursor != 0; cursor = ReadUint32Raw(cursor, 28) {
		info := NetworkAddrInfo{
			Flags:    ReadUint32Raw(cursor, 0),
			Family:   ReadUint32Raw(cursor, 4),
			SockType: ReadUint32Raw(cursor, 8),
			Protocol: ReadUint32Raw(cursor, 12),
			AddrLen:  ReadUint32Raw(cursor, 16),
		}

		canonPtr := ReadUint32Raw(cursor, 20)
		if canonPtr != 0 {
			info.CanonName = CStringToStringRaw(canonPtr)
		}

		addrPtr := ReadUint32Raw(cursor, 24)
		if addrPtr != 0 {
			familyAndPort := ReadUint32Raw(addrPtr, 0)
			info.Port = swap16(uint16(familyAndPort >> 16))
			info.Address = ReadUint32Raw(addrPtr, 4)
			info.AddressIP = network.InetNtoa(info.Address)
		}

		results = append(results, info)
	}

	return results
}

func networkLittleEndianUint32(buffer []byte) uint32 {
	if len(buffer) < 4 {
		return 0
	}

	return uint32(buffer[0]) |
		uint32(buffer[1])<<8 |
		uint32(buffer[2])<<16 |
		uint32(buffer[3])<<24
}

func optionalCStringAddress(value string) (*byte, uint32) {
	if value == "" {
		return nil, 0
	}

	return stringAddress(value)
}

func freeOptionalCString(ptr *byte) {
	if ptr != nil {
		freeCString(ptr)
	}
}
