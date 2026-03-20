package ssh

import (
	"fmt"

	"kos"
)

const debugKolibriSSH = false

func debugSSHf(format string, args ...interface{}) {
	if !debugKolibriSSH {
		return
	}
	prefix := "ssh debug"
	if tid, ok := kos.CurrentThreadID(); ok {
		prefix = fmt.Sprintf("%s [tid=0x%X", prefix, tid)
		if slot, ok := kos.CurrentThreadSlotIndex(); ok {
			prefix = fmt.Sprintf("%s slot=%d", prefix, slot)
		}
		prefix += "]"
	}
	message := fmt.Sprintf(prefix+": "+format, args...)
	fmt.Println(message)
	kos.DebugString(message)
}
