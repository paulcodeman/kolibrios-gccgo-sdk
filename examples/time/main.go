package main

import (
	"os"
	"time"
	"ui/elements"

	"kos"
	"ui"
)

const (
	timeprobeButtonExit    kos.ButtonID = 1
	timeprobeButtonSleep   kos.ButtonID = 2
	timeprobeButtonRefresh kos.ButtonID = 3

	timeprobeWindowX      = 320
	timeprobeWindowY      = 180
	timeprobeWindowWidth  = 690
	timeprobeWindowHeight = 302
	timeprobeWindowTitle  = "KolibriOS Time Demo"
)

type App struct {
	now          time.Time
	utcNow       time.Time
	fixedNow     time.Time
	uptime       uint32
	uptimeNS     uint64
	timeoutTicks uint32
	sleepDelta   time.Duration
	unixStable   bool
	locationOK   bool
	lastEvent    string
	sleep        *ui.Element
	refresh      *ui.Element
}

func NewApp() App {
	sleep := elements.ButtonAt(timeprobeButtonSleep, "Sleep 0.5s", 28, 248)
	sleep.SetWidth(142)

	refresh := elements.ButtonAt(timeprobeButtonRefresh, "Refresh", 190, 248)
	refresh.SetWidth(112)

	app := App{
		sleep:   sleep,
		refresh: refresh,
	}
	app.refreshTimeState()
	app.lastEvent = "startup refresh"

	return app
}

func (app *App) Run() {
	for {
		switch kos.WaitEventFor(50) {
		case kos.EventNone:
			app.timeoutTicks++
			app.refreshTimeState()
			app.lastEvent = "wait timeout / auto refresh"
			app.Redraw()
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
	case timeprobeButtonSleep:
		before := time.Now()
		time.Sleep(500 * time.Millisecond)
		app.sleepDelta = time.Since(before)
		app.refreshTimeState()
		app.lastEvent = "sleep delta / " + formatDurationMilliseconds(app.sleepDelta)
		app.Redraw()
	case timeprobeButtonRefresh:
		app.refreshTimeState()
		app.lastEvent = "manual refresh"
		app.Redraw()
	case timeprobeButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(timeprobeButtonExit, "Exit", 322, 248)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(timeprobeWindowX, timeprobeWindowY, timeprobeWindowWidth, timeprobeWindowHeight, timeprobeWindowTitle)
	kos.DrawText(28, 44, ui.White, "time.Now(): "+formatTimeStamp(app.now)+" / loc "+app.now.Location().String())
	kos.DrawText(28, 64, ui.Silver, "UTC(): "+formatTimeStamp(app.utcNow)+" / loc "+app.utcNow.Location().String())
	kos.DrawText(28, 84, ui.Aqua, "FixedZone(+03): "+formatTimeStamp(app.fixedNow))
	kos.DrawText(28, 104, ui.Lime, "Unix roundtrip: "+formatBoolWord(app.unixStable)+" / location parity "+formatBoolWord(app.locationOK))
	kos.DrawText(28, 124, ui.Yellow, "Sleep 0.5s delta: "+formatDurationMilliseconds(app.sleepDelta))
	kos.DrawText(28, 144, ui.White, "Uptime: "+formatUint32(app.uptime)+" cs / "+formatCentisecondsAsSeconds(app.uptime))
	kos.DrawText(28, 164, ui.Silver, "High precision uptime: "+formatHex64(app.uptimeNS))
	kos.DrawText(28, 184, ui.Aqua, "Wall clock source: syscalls 29 + 3 / UTC and fixed offsets now supported")
	kos.DrawText(28, 204, ui.Lime, "Monotonic source: syscall 26.10 / wait timeouts "+formatUint32(app.timeoutTicks))
	kos.DrawText(28, 224, ui.Yellow, "Last event: "+app.lastEvent)
	app.sleep.Draw()
	app.refresh.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshTimeState() {
	fixed := time.FixedZone("UTC+03", 3*60*60)

	app.now = time.Now()
	app.utcNow = app.now.UTC()
	app.fixedNow = app.now.In(fixed)
	app.uptime = kos.UptimeCentiseconds()
	app.uptimeNS = kos.UptimeNanoseconds()
	app.unixStable = time.Unix(app.now.Unix(), int64(app.now.Nanosecond())).Equal(app.now)
	app.locationOK = app.utcNow.Location() == time.UTC &&
		app.fixedNow.Location() == fixed &&
		app.fixedNow.Unix() == app.now.Unix() &&
		app.fixedNow.UTC().Equal(app.utcNow)
}

func main() {
	app := NewApp()
	app.Run()
}
