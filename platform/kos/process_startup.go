package kos

// ProcessStartup bridges attributes absent from KolibriOS's native loader.
// It is an SDK startup format, not an additional kernel syscall ABI.
type ProcessStartup struct {
	Args          []string
	Env           []string
	Dir           string
	StatusPath    string
	StandardFiles []string
}

const processStartupPrefix = "@KGP1:"

var loadedProcessStartup *ProcessStartup
var processStartupLoaded bool
var processExitReported bool

// The os adapter installs a closer so explicitly closed and shared standard
// streams use the same reference-counted lifetime as File.Close.
var ChildProcessCloseStreams func()

func processWord(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
}

func processWordAt(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}

func EncodeProcessStartup(startup *ProcessStartup) []byte {
	data := []byte("KGP1")
	appendString := func(value string) {
		data = append(data, processWord(uint32(len(value)))...)
		data = append(data, []byte(value)...)
	}
	appendString(startup.Dir)
	appendString(startup.StatusPath)
	for _, list := range [][]string{startup.Args, startup.Env, startup.StandardFiles} {
		data = append(data, processWord(uint32(len(list)))...)
		for _, value := range list {
			appendString(value)
		}
	}
	return data
}

func DecodeProcessStartup(data []byte) (*ProcessStartup, bool) {
	if len(data) < 4 || len(data) > 1024*1024 || string(data[:4]) != "KGP1" {
		return nil, false
	}
	position := 4
	word := func() (uint32, bool) {
		if len(data)-position < 4 {
			return 0, false
		}
		result := processWordAt(data[position:])
		position += 4
		return result, true
	}
	readString := func() (string, bool) {
		n, ok := word()
		if !ok || uint32(len(data)-position) < n {
			return "", false
		}
		result := string(data[position : position+int(n)])
		position += int(n)
		for i := 0; i < len(result); i++ {
			if result[i] == 0 {
				return "", false
			}
		}
		return result, true
	}
	startup := new(ProcessStartup)
	var ok bool
	if startup.Dir, ok = readString(); !ok {
		return nil, false
	}
	if startup.StatusPath, ok = readString(); !ok {
		return nil, false
	}
	for _, output := range []*[]string{&startup.Args, &startup.Env, &startup.StandardFiles} {
		count, ok := word()
		if !ok || count > uint32((len(data)-position)/4) {
			return nil, false
		}
		*output = make([]string, int(count))
		for i := range *output {
			value, ok := readString()
			if !ok {
				return nil, false
			}
			(*output)[i] = value
		}
	}
	if position != len(data) || startup.StatusPath == "" {
		return nil, false
	}
	return startup, true
}

func ProcessStartupArgument(path string) string {
	const digits = "0123456789abcdef"
	data := make([]byte, len(processStartupPrefix)+len(path)*2)
	copy(data, processStartupPrefix)
	for i := 0; i < len(path); i++ {
		data[len(processStartupPrefix)+i*2] = digits[path[i]>>4]
		data[len(processStartupPrefix)+i*2+1] = digits[path[i]&15]
	}
	return string(data)
}

func CurrentProcessStartup() *ProcessStartup {
	if processStartupLoaded {
		return loadedProcessStartup
	}
	processStartupLoaded = true
	argument := LoaderParameters()
	if len(argument) < len(processStartupPrefix) || argument[:len(processStartupPrefix)] != processStartupPrefix {
		return nil
	}
	hex := argument[len(processStartupPrefix):]
	if len(hex)%2 != 0 || len(hex) > 1800 {
		return nil
	}
	path := make([]byte, len(hex)/2)
	digit := func(value byte) int {
		if value >= '0' && value <= '9' {
			return int(value - '0')
		}
		if value >= 'a' && value <= 'f' {
			return int(value-'a') + 10
		}
		return -1
	}
	for i := range path {
		a, b := digit(hex[i*2]), digit(hex[i*2+1])
		if a < 0 || b < 0 {
			return nil
		}
		path[i] = byte(a*16 + b)
	}
	data, status := ReadAllFile(string(path))
	if status != FileSystemOK {
		return nil
	}
	startup, ok := DecodeProcessStartup(data)
	if !ok {
		return nil
	}
	loadedProcessStartup = startup
	initializeChildProcessControl(startup)
	return startup
}

func ChildProcessExit(code int) {
	if processExitReported {
		return
	}
	startup := CurrentProcessStartup()
	if startup == nil {
		return
	}
	if ChildProcessCloseStreams != nil {
		ChildProcessCloseStreams()
	} else {
		for i, specification := range startup.StandardFiles {
			descriptor, ok := LocalSocketStartupDescriptor(specification)
			if !ok {
				continue
			}
			duplicate := false
			for _, previous := range startup.StandardFiles[:i] {
				if previous == specification {
					duplicate = true
					break
				}
			}
			if !duplicate {
				CloseLocalSocket(descriptor)
			}
		}
	}
	status := append([]byte("KGS1"), processWord(uint32(int32(code)))...)
	// Main-return and os.Exit both arrive before process thread teardown.
	written, result := WriteFile(startup.StatusPath, status, 0)
	if result == FileSystemOK && written == uint32(len(status)) {
		processExitReported = true
	}
}
