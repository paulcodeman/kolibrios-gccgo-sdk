//go:build kolibrios && gccgo

package net

import (
	"kos"
	"runtime"
	"syscall"
	"unsafe"
)

// The public network ABI is described by sysfuncs.txt, calls 74 and 76.
// Slot numbers can contain holes after device removal (network/stack.inc).
// Go indexes are one-based; native slots are zero-based and limited to 16.
// MTU remains zero because the public ABI has no MTU query. Kernel pointers
// returned by 74/4 are deliberately not dereferenced from user space.
func interfaceTable(ifindex int) ([]Interface, error) {
	var interfaces []Interface
	for slot := 0; slot < 16; slot++ {
		if ifindex != 0 && ifindex != slot+1 {
			continue
		}
		regs := kos.SyscallRegs{EAX: 74, EBX: uint32(slot << 8)}
		kos.SyscallRaw(&regs)
		if regs.EAX == ^uint32(0) {
			continue
		}
		kind := regs.EAX
		var name [64]byte
		regs = kos.SyscallRegs{EAX: 74, EBX: uint32(slot<<8 | 1), ECX: uint32(uintptr(unsafe.Pointer(&name[0])))}
		kos.SyscallRaw(&regs)
		runtime.KeepAlive(&name)
		if regs.EAX == ^uint32(0) {
			continue // Device removed during enumeration.
		}
		n := 0
		for n < len(name) && name[n] != 0 {
			n++
		}
		// Registered devices are administratively enabled. FlagUp does not
		// imply physical carrier; the kernel reports carrier separately.
		ifi := Interface{Index: slot + 1, Name: string(name[:n]), Flags: FlagUp}
		switch kind {
		case 0: // NET_DEVICE_LOOPBACK
			ifi.Flags |= FlagLoopback
		case 1: // NET_DEVICE_ETH
			ifi.Flags |= FlagBroadcast
			regs = kos.SyscallRegs{EAX: 76, EBX: uint32(slot << 8)}
			kos.SyscallRaw(&regs)
			// ethernet.inc read_mac returns the first two bytes in BX,
			// followed by four bytes in EAX, each in little-endian order.
			if regs.EAX != ^uint32(0) {
				ifi.HardwareAddr = HardwareAddr{byte(regs.EBX), byte(regs.EBX >> 8), byte(regs.EAX), byte(regs.EAX >> 8), byte(regs.EAX >> 16), byte(regs.EAX >> 24)}
			}
		case 2: // NET_DEVICE_SLIP
			ifi.Flags |= FlagPointToPoint
		}
		interfaces = append(interfaces, ifi)
	}
	return interfaces, nil
}

func interfaceAddrTable(ifi *Interface) ([]Addr, error) {
	index := 0
	if ifi != nil {
		index = ifi.Index
		if index <= 0 {
			return nil, errInvalidInterfaceIndex
		}
	}
	interfaces, err := interfaceTable(index)
	if err != nil {
		return nil, err
	}
	if ifi != nil && len(interfaces) == 0 {
		return nil, errNoSuchInterface
	}
	var addresses []Addr
	for _, entry := range interfaces {
		slot := uint32(entry.Index - 1)
		regs := kos.SyscallRegs{EAX: 76, EBX: 1<<16 | slot<<8 | 2}
		kos.SyscallRaw(&regs)
		ip := regs.EAX
		if ip == ^uint32(0) || ip == 0 {
			continue
		}
		regs = kos.SyscallRegs{EAX: 76, EBX: 1<<16 | slot<<8 | 6}
		kos.SyscallRaw(&regs)
		mask := regs.EAX
		if mask == ^uint32(0) && entry.Flags&FlagLoopback == 0 {
			// A /32 mask is also all ones, so confirm the slot still exists.
			current, _ := interfaceTable(entry.Index)
			if len(current) == 0 {
				continue
			}
		}
		addresses = append(addresses, &IPNet{IP: IP{byte(ip), byte(ip >> 8), byte(ip >> 16), byte(ip >> 24)}, Mask: IPMask{byte(mask), byte(mask >> 8), byte(mask >> 16), byte(mask >> 24)}})
	}
	return addresses, nil
}

func interfaceMulticastAddrTable(ifi *Interface) ([]Addr, error) {
	return nil, syscall.ENOTSUP
}
