package main

import (
	"fmt"
	"runtime"

	"kos"
)

const (
	windowTitle  = "KolibriOS Preemption Check"
	windowX      = 280
	windowY      = 180
	windowWidth  = 720
	windowHeight = 248

	colorBlack  kos.Color = 0x000000
	colorSilver kos.Color = 0xC0C0C0
	colorWhite  kos.Color = 0xFFFFFF
	colorAqua   kos.Color = 0x00FFFF
	colorLime   kos.Color = 0x00FF00
	colorYellow kos.Color = 0xFFFF00
	colorRed    kos.Color = 0xFF0000
)

type heartbeatUpdate struct {
	count int
	at    uint32
}

type app struct {
	prevGOMAXPROCS int
	nowGOMAXPROCS  int
	startedAt      uint32
	lastHeartbeat  uint32
	heartbeatCount int
	finalCount     int
	finished       bool
	heartbeatCh    chan heartbeatUpdate
	resultCh       chan int
	dirty          bool
}

func newApp() *app {
	probe := &app{
		heartbeatCh: make(chan heartbeatUpdate, 64),
		resultCh:    make(chan int, 1),
		dirty:       true,
	}

	probe.prevGOMAXPROCS = runtime.GOMAXPROCS(1)
	probe.nowGOMAXPROCS = runtime.GOMAXPROCS(0)
	probe.startedAt = kos.UptimeCentiseconds()
	probe.startProbe()
	return probe
}

func (app *app) startProbe() {
	done := make(chan struct{})

	go func() {
		start := kos.UptimeCentiseconds()
		for {
			if kos.UptimeCentiseconds()-start >= 300 {
				break
			}
		}
		close(done)
	}()

	go func() {
		count := 0
		defer close(app.heartbeatCh)
		defer close(app.resultCh)

		for {
			select {
			case <-done:
				app.resultCh <- count
				return
			default:
				count++
				update := heartbeatUpdate{
					count: count,
					at:    kos.UptimeCentiseconds(),
				}
				select {
				case app.heartbeatCh <- update:
				default:
				}
				kos.SleepCentiseconds(10)
			}
		}
	}()
}

func (app *app) pollProbe() {
	changed := false

	for {
		select {
		case update, ok := <-app.heartbeatCh:
			if !ok {
				app.heartbeatCh = nil
				continue
			}
			app.heartbeatCount = update.count
			app.lastHeartbeat = update.at
			changed = true
		case count, ok := <-app.resultCh:
			if !ok {
				app.resultCh = nil
				continue
			}
			app.finalCount = count
			app.finished = true
			changed = true
		default:
			if changed {
				app.dirty = true
			}
			return
		}
	}
}

func (app *app) run() {
	app.redraw()
	app.dirty = false

	for {
		app.pollProbe()

		switch kos.WaitEventFor(5) {
		case kos.EventNone:
			app.pollProbe()
			if app.dirty {
				app.redraw()
				app.dirty = false
			}
		case kos.EventRedraw:
			app.redraw()
			app.dirty = false
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		}
	}
}

func (app *app) redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(windowX, windowY, windowWidth, windowHeight, windowTitle)
	kos.DrawText(28, 44, colorWhite, fmt.Sprintf("GOMAXPROCS: prev=%d now=%d", app.prevGOMAXPROCS, app.nowGOMAXPROCS))
	kos.DrawText(28, 64, colorSilver, "Single-P probe: one goroutine spins for 3 seconds without Gosched.")
	kos.DrawText(28, 84, colorAqua, "Expected on preemptive runtime: heartbeat keeps incrementing during the busy loop.")
	kos.DrawText(28, 104, colorYellow, "Expected on cooperative runtime: heartbeat stays at 0 until the busy loop ends.")
	kos.DrawText(28, 130, colorWhite, fmt.Sprintf("Started at: %d cs / now: %d cs", app.startedAt, kos.UptimeCentiseconds()))
	kos.DrawText(28, 148, colorSilver, fmt.Sprintf("Last heartbeat at: %d cs", app.lastHeartbeat))
	kos.DrawText(28, 166, colorAqua, fmt.Sprintf("Heartbeat count: %d", app.heartbeatCount))
	kos.DrawText(28, 184, app.statusColor(), app.statusLine())
	kos.DrawText(28, 208, colorBlack, "Close the window to exit.")
	kos.EndRedraw()
}

func (app *app) statusColor() kos.Color {
	if !app.finished {
		return colorSilver
	}
	if app.finalCount > 0 {
		return colorLime
	}
	return colorRed
}

func (app *app) statusLine() string {
	if !app.finished {
		return "Probe running."
	}
	if app.finalCount > 0 {
		return fmt.Sprintf("Probe finished: heartbeat count=%d -> scheduler is preempting the single P.", app.finalCount)
	}
	return "Probe finished: heartbeat count=0 -> scheduler is still cooperative on the single P."
}

func main() {
	newApp().run()
}
