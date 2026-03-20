package rand

import (
	"os"
	"sync"
	"time"
	"unsafe"

	"kos"
)

const fallbackRandSeed = uint64(0x9e3779b97f4a7c15)

type fallbackReader struct {
	mu    sync.Mutex
	state uint64
}

func init() {
	// Temporary KolibriOS fallback until a real CSPRNG source is wired in.
	Reader = newFallbackReader()
}

func newFallbackReader() *fallbackReader {
	seed := uint64(time.Now().UnixNano())
	seed ^= kos.UptimeNanoseconds()
	seed ^= uint64(os.Getpid()) << 32
	seed ^= uint64(uintptr(unsafe.Pointer(&seed)))
	if seed == 0 {
		seed = fallbackRandSeed
	}

	return &fallbackReader{state: seed}
}

func (r *fallbackReader) Read(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := 0; i < len(b); {
		word := r.next()
		for j := 0; j < 8 && i < len(b); j++ {
			b[i] = byte(word)
			word >>= 8
			i++
		}
	}

	return len(b), nil
}

func (r *fallbackReader) next() uint64 {
	r.state += fallbackRandSeed

	z := r.state
	z = (z ^ (z >> 30)) * uint64(0xbf58476d1ce4e5b9)
	z = (z ^ (z >> 27)) * uint64(0x94d049bb133111eb)
	return z ^ (z >> 31)
}
