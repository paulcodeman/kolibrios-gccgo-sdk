package main

import "kos"

func configureUserNetwork() bool {
	// sysfuncs.txt 74/0: native device type, and 76/IPv4/3,5,7,9:
	// address, DNS, subnet and gateway. Values are network bytes in memory.
	for slot := uint32(0); slot < 16; slot++ {
		regs := kos.SyscallRegs{EAX: 74, EBX: slot << 8}
		kos.SyscallRaw(&regs)
		if regs.EAX != 1 { // NET_DEVICE_ETH
			continue
		}
		for _, setting := range [...]struct{ operation, value uint32 }{
			{3, 0x0f02000a}, // 10.0.2.15
			{5, 0x0302000a}, // 10.0.2.3
			{7, 0x00ffffff}, // 255.255.255.0
			{9, 0x0202000a}, // 10.0.2.2
		} {
			regs = kos.SyscallRegs{EAX: 76, EBX: 1<<16 | slot<<8 | setting.operation, ECX: setting.value}
			kos.SyscallRaw(&regs)
			if regs.EAX == ^uint32(0) {
				return false
			}
		}
		kos.DebugString("OPENCODE_TEST_USER_NETWORK_READY\n")
		return true
	}
	return false
}

func main() {
	serviceStarted := false
	networkConfigured := false
	for i := 0; i < 100; i++ {
		if !networkConfigured {
			if _, status := kos.GetPathInfo("/hd0/1/NET.USER"); status == kos.FileSystemOK {
				if !configureUserNetwork() {
					kos.SleepSeconds(1)
					continue
				}
				networkConfigured = true
			}
		}
		if _, status := kos.GetPathInfo("/hd0/1/SERVICE.KEX"); status == kos.FileSystemOK {
			if !serviceStarted {
				if pid, status := kos.StartApplication("/hd0/1/SERVICE.KEX", "", false); status == kos.FileSystemOK && pid > 0 {
					serviceStarted = true
				}
			}
			if _, status := kos.GetPathInfo("/hd0/1/SERVICE.READY"); status != kos.FileSystemOK {
				kos.SleepSeconds(1)
				continue
			}
		}
		arguments := ""
		if data, status := kos.ReadAllFile("/hd0/1/SMOKE.ARGS"); status == kos.FileSystemOK {
			arguments = string(data)
		}
		executable := "/hd0/1/SMOKE.KEX"
		if data, status := kos.ReadAllFile("/hd0/1/SMOKE.PATH"); status == kos.FileSystemOK {
			executable = string(data)
		}
		if pid, status := kos.StartApplication(executable, arguments, false); status == kos.FileSystemOK && pid > 0 {
			return
		}
		kos.SleepSeconds(1)
	}
	kos.DebugString("OpenCode test executable not found on /hd0/1")
}
