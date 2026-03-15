package main

import (
	"fmt"
	"net"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	networkButtonExit    kos.ButtonID = 1
	networkButtonRefresh kos.ButtonID = 2

	networkWindowTitle  = "KolibriOS Network Demo"
	networkWindowX      = 220
	networkWindowY      = 138
	networkWindowWidth  = 860
	networkWindowHeight = 316
)

type App struct {
	summary      string
	lookupLine   string
	hostPortLine string
	dllLine      string
	infoLine     string
	ok           bool
	refreshBtn   *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(networkButtonRefresh, "Refresh", 28, 264)
	refresh.SetWidth(116)

	app := App{
		refreshBtn: refresh,
	}
	app.refreshProbe()
	return app
}

func (app *App) Run() {
	for {
		switch kos.WaitEvent() {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case networkButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case networkButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(networkButtonExit, "Exit", 170, 264)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(networkWindowX, networkWindowY, networkWindowWidth, networkWindowHeight, networkWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary net package: import \"net\"")
	kos.DrawText(28, 92, ui.Aqua, app.lookupLine)
	kos.DrawText(28, 114, ui.Lime, app.hostPortLine)
	kos.DrawText(28, 136, ui.Yellow, app.dllLine)
	kos.DrawText(28, 158, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	network, ok := kos.LoadNetwork()
	if !ok {
		app.fail("network.obj unavailable", "Info: failed to load "+kos.NetworkDLLPath)
		return
	}

	loopback := network.InetAddr("127.0.0.1")
	loopbackText := network.InetNtoa(loopback)
	if loopbackText != "127.0.0.1" {
		app.fail("inet_addr / inet_ntoa mismatch", "Info: expected loopback round-trip")
		return
	}

	hosts, err := net.LookupHost("127.0.0.1")
	if err != nil {
		app.fail("LookupHost failed", "Info: "+err.Error())
		return
	}
	if len(hosts) != 1 || hosts[0] != "127.0.0.1" {
		app.fail("LookupHost mismatch", "Info: expected single 127.0.0.1 result")
		return
	}

	joinedIPv4 := net.JoinHostPort("127.0.0.1", "80")
	hostIPv4, portIPv4, splitIPv4Err := net.SplitHostPort(joinedIPv4)
	joinedIPv6 := net.JoinHostPort("2001:db8::1", "443")
	hostIPv6, portIPv6, splitIPv6Err := net.SplitHostPort(joinedIPv6)
	if splitIPv4Err != nil || splitIPv6Err != nil {
		app.fail("SplitHostPort failed", "Info: unexpected host:port split error")
		return
	}
	if hostIPv4 != "127.0.0.1" || portIPv4 != "80" || joinedIPv6 != "[2001:db8::1]:443" || hostIPv6 != "2001:db8::1" || portIPv6 != "443" {
		app.fail("host:port mismatch", "Info: join/split behavior differs from Go contract")
		return
	}

	app.ok = true
	app.summary = "network probe ok / ordinary import net resolved"
	app.lookupLine = fmt.Sprintf("LookupHost: 127.0.0.1 -> %v / inet 0x%x -> %s", hosts, loopback, loopbackText)
	app.hostPortLine = fmt.Sprintf("Join/Split: %s -> %s/%s / %s -> %s/%s", joinedIPv4, hostIPv4, portIPv4, joinedIPv6, hostIPv6, portIPv6)
	app.dllLine = fmt.Sprintf("DLL: %s / table 0x%x / ver 0x%x", kos.NetworkDLLPath, uint32(network.ExportTable()), network.Version())
	app.infoLine = "Info: getaddrinfo is active; this demo keeps the lookup offline by using a numeric host literal"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "network probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
