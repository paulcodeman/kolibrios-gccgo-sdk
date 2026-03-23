package main

import (
	"os"
	"strconv"

	"kos"
	"surface"
)

const (
	waveWindowX      = 72
	waveWindowY      = 72
	waveWindowWidth  = 680
	waveWindowHeight = 360
)

const (
	waveBlack   kos.Color = 0x000000
	waveWhite   kos.Color = 0xFFFFFF
	waveGray    kos.Color = 0x8EA0AD
	waveInk     kos.Color = 0x0D1721
	wavePanel   kos.Color = 0x162534
	wavePanel2  kos.Color = 0x1E3247
	waveGrid    kos.Color = 0x29455F
	waveBlue    kos.Color = 0x4F7CFF
	waveMint    kos.Color = 0x7AE582
	waveGold    kos.Color = 0xFFCF5C
	waveRose    kos.Color = 0xFF7A90
)

type waveApp struct {
	presenter  surface.Presenter
	canvas     *surface.Buffer
	graph      *surface.Buffer
	graphRect  surface.Rect
	statusRect surface.Rect
	ticks      int
	lastA      int
	lastB      int
}

func uniformWaveRadii(radius int) surface.CornerRadii {
	return surface.CornerRadii{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

func insetWave(rect surface.Rect, left int, top int, right int, bottom int) surface.Rect {
	width := rect.Width - left - right
	height := rect.Height - top - bottom
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return surface.Rect{
		X:      rect.X + left,
		Y:      rect.Y + top,
		Width:  width,
		Height: height,
	}
}

func triWave(step int, amplitude int) int {
	if amplitude <= 1 {
		return 0
	}
	period := amplitude * 2
	value := step % period
	if value < 0 {
		value += period
	}
	if value >= amplitude {
		value = period - value - 1
	}
	return value
}

func newWaveApp() *waveApp {
	presenter := surface.NewPresenter(waveWindowX, waveWindowY, waveWindowWidth, waveWindowHeight, "Surface Wave")
	client := presenter.Client
	graphRect := surface.Rect{X: 18, Y: 74, Width: client.Width - 36, Height: 184}
	statusRect := surface.Rect{X: 18, Y: 274, Width: client.Width - 36, Height: 34}
	app := &waveApp{
		presenter:  presenter,
		canvas:     surface.NewBuffer(client.Width, client.Height),
		graph:      surface.NewBuffer(graphRect.Width, graphRect.Height),
		graphRect:  graphRect,
		statusRect: statusRect,
	}
	app.lastA = graphRect.Height / 2
	app.lastB = graphRect.Height / 2
	app.resetGraph()
	app.renderFrame()
	app.blitGraph()
	app.drawStatus()
	return app
}

func (app *waveApp) resetGraph() {
	app.graph.Clear(wavePanel)
	for x := 0; x < app.graph.Width(); x += 32 {
		app.graph.DrawLine(x, 0, x, app.graph.Height()-1, waveGrid)
	}
	for y := 16; y < app.graph.Height(); y += 32 {
		app.graph.DrawLine(0, y, app.graph.Width()-1, y, waveGrid)
	}
	mid := app.graph.Height() / 2
	app.graph.DrawLine(0, mid, app.graph.Width()-1, mid, waveGray)
}

func (app *waveApp) renderFrame() {
	bounds := app.canvas.Bounds()
	app.canvas.Clear(waveInk)
	app.canvas.FillRectGradient(0, 0, bounds.Width, 56, surface.Gradient{
		From:      wavePanel2,
		To:        waveBlue,
		Direction: surface.GradientHorizontal,
	})
	app.canvas.DrawText(18, 16, waveWhite, "surface: animated partial redraw")
	app.canvas.DrawText(18, 32, waveGray, "BlitSelf scrolls the trace, PresentRect updates only the graph + status")
	panel := insetWave(surface.Rect{X: 10, Y: 60, Width: bounds.Width - 20, Height: bounds.Height - 70}, 0, 0, 0, 0)
	app.canvas.DrawShadowRounded(panel, surface.Shadow{
		OffsetX: 0,
		OffsetY: 2,
		Blur:    3,
		Color:   waveBlack,
		Alpha:   68,
	}, uniformWaveRadii(14))
	app.canvas.FillRoundedRectGradient(panel.X, panel.Y, panel.Width, panel.Height, uniformWaveRadii(14), surface.Gradient{
		From:      wavePanel2,
		To:        wavePanel,
		Direction: surface.GradientVertical,
	})
	app.canvas.StrokeRoundedRectWidth(panel.X, panel.Y, panel.Width, panel.Height, uniformWaveRadii(14), 1, 0x40FFFFFF)
	app.canvas.DrawText(app.graphRect.X, app.graphRect.Y-18, waveWhite, "trace A / trace B / scroll-by-copy")
}

func (app *waveApp) blitGraph() {
	app.canvas.BlitFrom(app.graph, app.graph.Bounds(), app.graphRect.X, app.graphRect.Y)
}

func (app *waveApp) drawStatus() {
	app.canvas.FillRoundedRect(app.statusRect.X, app.statusRect.Y, app.statusRect.Width, app.statusRect.Height, uniformWaveRadii(10), wavePanel)
	app.canvas.DrawText(app.statusRect.X+12, app.statusRect.Y+10, waveWhite, "ticks="+strconv.Itoa(app.ticks))
	app.canvas.DrawText(app.statusRect.X+132, app.statusRect.Y+10, waveMint, "A="+strconv.Itoa(app.lastA))
	app.canvas.DrawText(app.statusRect.X+232, app.statusRect.Y+10, waveGold, "B="+strconv.Itoa(app.lastB))
	app.canvas.DrawText(app.statusRect.X+332, app.statusRect.Y+10, waveGray, "PresentRect(graph) + PresentRect(status)")
}

func (app *waveApp) step() {
	app.ticks++
	width := app.graph.Width()
	height := app.graph.Height()
	app.graph.BlitSelf(surface.Rect{X: 1, Y: 0, Width: width - 1, Height: height}, 0, 0)
	app.graph.FillRect(width-1, 0, 1, height, wavePanel)
	if app.ticks%32 == 0 {
		app.graph.DrawLine(width-1, 0, width-1, height-1, waveGrid)
	}
	mid := height / 2
	nextA := 18 + triWave(app.ticks*3, height-36)
	nextB := 18 + triWave(app.ticks*2+height/3, height-36)
	app.graph.DrawLine(width-2, app.lastA, width-1, nextA, waveMint)
	app.graph.DrawLine(width-2, app.lastB, width-1, nextB, waveGold)
	app.graph.SetPixel(width-1, mid, waveGray)
	app.graph.SetPixelAlpha(width-1, nextA, waveWhite, 220)
	app.graph.SetPixelAlpha(width-1, nextB, waveRose, 180)
	app.lastA = nextA
	app.lastB = nextB
	app.blitGraph()
	app.drawStatus()
	app.presenter.PresentRect(app.canvas, app.graphRect)
	app.presenter.PresentRect(app.canvas, app.statusRect)
}

func (app *waveApp) redrawFull() {
	app.renderFrame()
	app.blitGraph()
	app.drawStatus()
	app.presenter.PresentFull(app.canvas)
}

func (app *waveApp) run() {
	app.redrawFull()
	for {
		switch kos.WaitEventFor(4) {
		case kos.EventNone:
			app.step()
		case kos.EventRedraw:
			app.redrawFull()
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		case kos.EventKey:
			_ = kos.ReadKey()
		}
	}
}

func main() {
	app := newWaveApp()
	app.run()
	os.Exit(0)
}
