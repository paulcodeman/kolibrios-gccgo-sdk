package main

import (
	"kos"
	"net"
)

func check(ok bool, why string) {
	if !ok {
		kos.DebugString(why)
		panic(why)
	}
}

func main() {
	console, ok := kos.OpenConsole("OpenCode interface test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_INTERFACE_START")
	for _, s := range []string{"00:00:5e:00:53:01", "00-00-5e-00-53-01", "0000.5e00.5301"} {
		mac, err := net.ParseMAC(s)
		check(err == nil && mac.String() == "00:00:5e:00:53:01", "MAC parsing")
	}
	_, err := net.ParseMAC("00:00:5e:00:53:xx")
	check(err != nil, "invalid MAC")
	interfaces, err := net.Interfaces()
	check(err == nil && len(interfaces) > 0, "Interfaces")
	regs := kos.SyscallRegs{EAX: 74, EBX: 255}
	kos.SyscallRaw(&regs)
	check(int(regs.EAX) == len(interfaces), "native interface count")
	found := false
	for _, ifi := range interfaces {
		kos.DebugString(ifi.Name)
		byIndex, err := net.InterfaceByIndex(ifi.Index)
		check(err == nil && byIndex.Name == ifi.Name, "InterfaceByIndex")
		byName, err := net.InterfaceByName(ifi.Name)
		check(err == nil && byName.Index == ifi.Index, "InterfaceByName")
		addresses, err := ifi.Addrs()
		check(err == nil, "interface addresses")
		if ifi.Flags&net.FlagLoopback != 0 {
			check(ifi.Flags&net.FlagUp != 0, "loopback up")
			for _, addr := range addresses {
				kos.DebugString(addr.String())
				if addr.String() == "127.0.0.1/8" {
					found = true
				}
			}
		}
	}
	check(found, "native loopback IPv4 and mask")
	all, err := net.InterfaceAddrs()
	check(err == nil && len(all) > 0, "InterfaceAddrs")
	_, err = net.InterfaceByIndex(0)
	check(err != nil, "invalid index")
	_, err = net.InterfaceByName("no-such-device")
	check(err != nil, "missing device")
	kos.DebugString("OPENCODE_INTERFACE_PASS")
	console.WriteString("OPENCODE_INTERFACE_PASS\n")
}
