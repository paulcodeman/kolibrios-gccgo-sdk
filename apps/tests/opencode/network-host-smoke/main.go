package main

import (
	"io"
	"kos"
	"net"
	"net/http"
	"strings"
	"time"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode host model network path")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	found := false
	for _, device := range interfaces {
		addresses, err := device.Addrs()
		if err != nil {
			panic(err)
		}
		for _, address := range addresses {
			if address.String() == "10.0.2.15/24" {
				found = true
			}
		}
	}
	if !found {
		panic("QEMU user IPv4 configuration missing")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get("http://10.0.2.2:18082/check.txt")
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 || strings.TrimSpace(string(body)) != "KolibriOS host-network PASS" {
		panic("host HTTP response mismatch")
	}
	console.WriteString("Native RTL8139, IPv4 interface and host HTTP model path PASS\n")
	kos.DebugString("OPENCODE_HOST_NETWORK_PASS")
}
