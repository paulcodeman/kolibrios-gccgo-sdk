package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"kos"
	"surface"
	surfacetinygl "surface/tinygl"
)

const (
	windowX      = 64
	windowY      = 56
	windowWidth  = 960
	windowHeight = 720

	hudUpdateNS = 250000000
)

const (
	glColorBufferBit = 0x00004000
	glDepthBufferBit = 0x00000100

	glTriangles = 0x0004

	glBack      = 0x0405
	glCullFace  = 0x0B44
	glDepthTest = 0x0B71

	glSmooth = 0x1D01

	glModelview  = 0x1700
	glProjection = 0x1701
)

const (
	colorInk      kos.Color = 0x0B1420
	colorPanel    kos.Color = 0x122234
	colorPanel2   kos.Color = 0x183049
	colorPanel3   kos.Color = 0x213D5C
	colorGrid     kos.Color = 0x294B69
	colorBlue     kos.Color = 0x4D84FF
	colorMint     kos.Color = 0x7FE7C4
	colorGold     kos.Color = 0xF8C95C
	colorRose     kos.Color = 0xFF7A93
	colorSilver   kos.Color = 0xAFC4D8
	colorWhite    kos.Color = 0xF5FBFF
	colorShadow   kos.Color = 0x000000
	colorViewport kos.Color = 0x06111B
)

type stressApp struct {
	presenter      surface.Presenter
	canvas         *surface.Buffer
	layer          surfacetinygl.Layer
	console        kos.Console
	consoleOK      bool
	headerRect     surface.Rect
	footerRect     surface.Rect
	viewFrameRect  surface.Rect
	viewRect       surface.Rect
	viewWindowRect surface.Rect
	density        int
	paused         bool
	failed         bool
	failureText    string
	runStartNS     uint64
	windowStartNS  uint64
	totalFrames    uint64
	windowFrames   uint64
	fpsCurrent     float64
	fpsAverage     float64
	fpsBest        float64
	lastHUDNS      uint64
	projection     [16]float32
}

func uniformRadii(radius int) surface.CornerRadii {
	return surface.CornerRadii{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

func insetRect(rect surface.Rect, inset int) surface.Rect {
	width := rect.Width - inset*2
	height := rect.Height - inset*2
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return surface.Rect{
		X:      rect.X + inset,
		Y:      rect.Y + inset,
		Width:  width,
		Height: height,
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampDensity(value int) int {
	if value < 1 {
		return 1
	}
	if value > 8 {
		return 8
	}
	return value
}

func readDensity() int {
	if len(os.Args) < 2 {
		return 1
	}
	value, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return 1
	}
	return clampDensity(value)
}

func newStressApp(density int) *stressApp {
	presenter := surface.NewPresenter(windowX, windowY, windowWidth, windowHeight, "TinyGL Stress")
	client := presenter.Client
	headerRect := surface.Rect{X: 0, Y: 0, Width: client.Width, Height: 62}
	footerRect := surface.Rect{X: 0, Y: client.Height - 104, Width: client.Width, Height: 104}
	viewFrameRect := surface.Rect{
		X:      16,
		Y:      headerRect.Height + 12,
		Width:  client.Width - 32,
		Height: client.Height - headerRect.Height - footerRect.Height - 28,
	}
	viewRect := insetRect(viewFrameRect, 14)
	app := &stressApp{
		presenter:  presenter,
		canvas:     surface.NewBuffer(client.Width, client.Height),
		headerRect: headerRect,
		footerRect: footerRect,
		viewFrameRect: surface.Rect{
			X:      viewFrameRect.X,
			Y:      viewFrameRect.Y,
			Width:  viewFrameRect.Width,
			Height: viewFrameRect.Height,
		},
		viewRect: viewRect,
		viewWindowRect: surface.Rect{
			X:      presenter.Client.X + viewRect.X,
			Y:      presenter.Client.Y + viewRect.Y,
			Width:  viewRect.Width,
			Height: viewRect.Height,
		},
		density: clampDensity(density),
	}
	console, ok := kos.OpenConsole("TinyGL Stress")
	if ok {
		app.console = console
		app.consoleOK = true
		app.console.WriteString("TinyGL Stress\n")
		app.console.WriteString("Controls: +/- density, Space pause, R reset, Esc exit\n")
	}
	app.resetStats(kos.UptimeNanoseconds())
	return app
}

func (app *stressApp) sceneGrid() (int, int, int) {
	return 6*app.density + 2, 4*app.density + 2, 3*app.density + 2
}

func (app *stressApp) sceneMetrics() (int, int) {
	cols, rows, layers := app.sceneGrid()
	cubes := cols * rows * layers
	return cubes, cubes * 12
}

func (app *stressApp) resetStats(now uint64) {
	app.runStartNS = now
	app.windowStartNS = now
	app.totalFrames = 0
	app.windowFrames = 0
	app.fpsCurrent = 0
	app.fpsAverage = 0
	app.fpsBest = 0
	app.lastHUDNS = 0
	app.updateConsoleTitle()
}

func (app *stressApp) updateConsoleTitle() {
	if !app.consoleOK || !app.console.SupportsTitle() {
		return
	}
	cubes, triangles := app.sceneMetrics()
	status := "running"
	if app.paused {
		status = "paused"
	}
	if app.failed {
		status = "failed"
	}
	title := fmt.Sprintf(
		"TinyGL Stress / %s / %.1f fps / cubes=%d / tris=%d / density=%d",
		status,
		app.fpsCurrent,
		cubes,
		triangles,
		app.density,
	)
	app.console.SetTitle(title)
}

func (app *stressApp) drawChrome() {
	bounds := app.canvas.Bounds()
	app.canvas.Clear(colorInk)
	app.canvas.FillRectGradient(0, 0, bounds.Width, app.headerRect.Height, surface.Gradient{
		From:      colorPanel3,
		To:        colorBlue,
		Direction: surface.GradientHorizontal,
	})
	app.canvas.DrawShadowRounded(app.viewFrameRect, surface.Shadow{
		OffsetX: 0,
		OffsetY: 2,
		Blur:    4,
		Color:   colorShadow,
		Alpha:   88,
	}, uniformRadii(16))
	app.canvas.FillRoundedRectGradient(app.viewFrameRect.X, app.viewFrameRect.Y, app.viewFrameRect.Width, app.viewFrameRect.Height, uniformRadii(16), surface.Gradient{
		From:      colorPanel2,
		To:        colorPanel,
		Direction: surface.GradientVertical,
	})
	app.canvas.StrokeRoundedRectWidth(app.viewFrameRect.X, app.viewFrameRect.Y, app.viewFrameRect.Width, app.viewFrameRect.Height, uniformRadii(16), 1, 0x48FFFFFF)
	app.canvas.FillRect(app.viewRect.X, app.viewRect.Y, app.viewRect.Width, app.viewRect.Height, colorViewport)
	app.drawHeader()
	app.drawFooter()
}

func (app *stressApp) drawHeader() {
	app.canvas.FillRectGradient(app.headerRect.X, app.headerRect.Y, app.headerRect.Width, app.headerRect.Height, surface.Gradient{
		From:      colorPanel3,
		To:        colorBlue,
		Direction: surface.GradientHorizontal,
	})
	cubes, triangles := app.sceneMetrics()
	mode := "running"
	if app.paused {
		mode = "paused"
	}
	if app.failed {
		mode = "tinygl unavailable"
	}
	app.canvas.DrawText(18, 16, colorWhite, "tinygl: direct surface/tinygl stress test")
	app.canvas.DrawText(18, 34, colorSilver, fmt.Sprintf("mode=%s / density=%d / cubes=%d / triangles/frame=%d / viewport=%dx%d", mode, app.density, cubes, triangles, app.viewRect.Width, app.viewRect.Height))
}

func (app *stressApp) drawFooter() {
	app.canvas.FillRect(app.footerRect.X, app.footerRect.Y, app.footerRect.Width, app.footerRect.Height, colorInk)
	panel := insetRect(app.footerRect, 10)
	app.canvas.FillRoundedRectGradient(panel.X, panel.Y, panel.Width, panel.Height, uniformRadii(12), surface.Gradient{
		From:      colorPanel2,
		To:        colorPanel,
		Direction: surface.GradientVertical,
	})
	app.canvas.StrokeRoundedRectWidth(panel.X, panel.Y, panel.Width, panel.Height, uniformRadii(12), 1, 0x30FFFFFF)

	status := "running"
	if app.paused {
		status = "paused"
	}
	if app.failed {
		status = app.failureText
	}
	app.canvas.DrawText(panel.X+14, panel.Y+14, colorWhite, fmt.Sprintf("fps %.1f / avg %.1f / best %.1f / frames %d", app.fpsCurrent, app.fpsAverage, app.fpsBest, app.totalFrames))
	app.canvas.DrawText(panel.X+14, panel.Y+34, colorMint, fmt.Sprintf("controls: +/- density, space pause, r reset, esc exit / status: %s", status))
	app.canvas.DrawText(panel.X+14, panel.Y+54, colorGold, "path: direct surface/tinygl layer without ui")
}

func (app *stressApp) drawFailureView() {
	app.canvas.FillRect(app.viewRect.X, app.viewRect.Y, app.viewRect.Width, app.viewRect.Height, colorViewport)
	app.canvas.DrawText(app.viewRect.X+18, app.viewRect.Y+22, colorRose, "TinyGL render path is unavailable.")
	app.canvas.DrawText(app.viewRect.X+18, app.viewRect.Y+42, colorSilver, "Check /sys/lib/tinygl.obj and current video mode.")
	app.canvas.DrawText(app.viewRect.X+18, app.viewRect.Y+62, colorSilver, "The app itself uses surface/tinygl directly and does not depend on ui.")
}

func (app *stressApp) perspectiveMatrix(width int, height int) []float32 {
	if width <= 0 || height <= 0 {
		copy(app.projection[:], []float32{
			1, 0, 0, 0,
			0, 1, 0, 0,
			0, 0, -1, -1,
			0, 0, -0.2, 0,
		})
		return app.projection[:]
	}
	aspect := float32(width) / float32(height)
	fovRadians := float32(55.0 * math.Pi / 180.0)
	near := float32(0.5)
	far := float32(120.0)
	f := float32(1.0 / math.Tan(float64(fovRadians*0.5)))
	app.projection[0] = f / aspect
	app.projection[1] = 0
	app.projection[2] = 0
	app.projection[3] = 0
	app.projection[4] = 0
	app.projection[5] = f
	app.projection[6] = 0
	app.projection[7] = 0
	app.projection[8] = 0
	app.projection[9] = 0
	app.projection[10] = (far + near) / (near - far)
	app.projection[11] = -1
	app.projection[12] = 0
	app.projection[13] = 0
	app.projection[14] = (2 * far * near) / (near - far)
	app.projection[15] = 0
	return app.projection[:]
}

func faceColor(gl *kos.TinyGL, r uint8, g uint8, b uint8) {
	gl.Color3ub(r, g, b)
}

func drawCube(gl *kos.TinyGL, size float32, seed int) {
	s := size * 0.5
	tint := uint8((seed * 11) & 31)
	faceColor(gl, 0xFF-tint, 0x6A+tint, 0x88+tint/2)
	gl.Vertex3f(-s, -s, s)
	gl.Vertex3f(s, -s, s)
	gl.Vertex3f(s, s, s)
	gl.Vertex3f(-s, -s, s)
	gl.Vertex3f(s, s, s)
	gl.Vertex3f(-s, s, s)

	faceColor(gl, 0x76+tint/2, 0xD9, 0xFF-tint/2)
	gl.Vertex3f(s, -s, -s)
	gl.Vertex3f(-s, -s, -s)
	gl.Vertex3f(-s, s, -s)
	gl.Vertex3f(s, -s, -s)
	gl.Vertex3f(-s, s, -s)
	gl.Vertex3f(s, s, -s)

	faceColor(gl, 0x8A+tint, 0xF0-tint/3, 0xBC)
	gl.Vertex3f(-s, -s, -s)
	gl.Vertex3f(-s, -s, s)
	gl.Vertex3f(-s, s, s)
	gl.Vertex3f(-s, -s, -s)
	gl.Vertex3f(-s, s, s)
	gl.Vertex3f(-s, s, -s)

	faceColor(gl, 0xFF, 0xD0-tint/4, 0x70+tint)
	gl.Vertex3f(s, -s, s)
	gl.Vertex3f(s, -s, -s)
	gl.Vertex3f(s, s, -s)
	gl.Vertex3f(s, -s, s)
	gl.Vertex3f(s, s, -s)
	gl.Vertex3f(s, s, s)

	faceColor(gl, 0xF7, 0xF9, 0xFF)
	gl.Vertex3f(-s, s, s)
	gl.Vertex3f(s, s, s)
	gl.Vertex3f(s, s, -s)
	gl.Vertex3f(-s, s, s)
	gl.Vertex3f(s, s, -s)
	gl.Vertex3f(-s, s, -s)

	faceColor(gl, 0x58+tint/2, 0x7E+tint/2, 0xD8)
	gl.Vertex3f(-s, -s, -s)
	gl.Vertex3f(s, -s, -s)
	gl.Vertex3f(s, -s, s)
	gl.Vertex3f(-s, -s, -s)
	gl.Vertex3f(s, -s, s)
	gl.Vertex3f(-s, -s, s)
}

func (app *stressApp) drawScene(gl *kos.TinyGL, now uint64) {
	cols, rows, layers := app.sceneGrid()
	phase := float32(now-app.runStartNS) / 1000000000.0
	spacing := float32(1.55)
	size := float32(0.62)
	centerX := float32(cols-1) * 0.5
	centerY := float32(rows-1) * 0.5
	centerZ := float32(layers-1) * 0.5
	camera := float32(10.0 + float32(maxInt(cols, rows))*0.75 + float32(layers)*0.85)

	gl.ClearColor(0.04, 0.08, 0.12, 1.0)
	gl.Clear(glColorBufferBit | glDepthBufferBit)
	gl.Enable(glDepthTest)
	gl.Enable(glCullFace)
	gl.CullFace(glBack)
	gl.ShadeModel(glSmooth)

	gl.MatrixMode(glProjection)
	gl.LoadMatrix(app.perspectiveMatrix(app.viewRect.Width, app.viewRect.Height))
	gl.MatrixMode(glModelview)
	gl.LoadIdentity()
	gl.Translatef(0, 0, -camera)
	gl.Rotatef(phase*19.0, 0.45, 1.0, 0.0)
	gl.Rotatef(phase*13.0, 1.0, 0.2, 0.0)

	for z := 0; z < layers; z++ {
		for y := 0; y < rows; y++ {
			for x := 0; x < cols; x++ {
				seed := x*17 + y*29 + z*13
				px := (float32(x) - centerX) * spacing
				py := (float32(y) - centerY) * spacing
				pz := (float32(z) - centerZ) * spacing

				gl.PushMatrix()
				gl.Translatef(px, py, pz)
				gl.Rotatef(phase*90.0+float32(seed%360), 1.0, 0.35, 0.25)
				gl.Rotatef(phase*47.0+float32((seed*3)%360), 0.2, 1.0, 0.4)
				gl.Begin(glTriangles)
				drawCube(gl, size, seed)
				gl.End()
				gl.PopMatrix()
			}
		}
	}
	gl.Flush()
}

func (app *stressApp) renderTinyGL(now uint64) bool {
	if app.failed {
		return false
	}
	if app.layer.Render(app.viewWindowRect, func(gl *kos.TinyGL, ctx *kos.TinyGLContext) {
		app.drawScene(gl, now)
	}) {
		return true
	}
	app.failed = true
	app.failureText = "tinygl load/context init failed"
	app.drawFailureView()
	app.drawFooter()
	app.presenter.PresentRect(app.canvas, app.viewFrameRect)
	app.presenter.PresentRect(app.canvas, app.footerRect)
	app.updateConsoleTitle()
	return false
}

func (app *stressApp) updateFPS(now uint64) {
	delta := now - app.windowStartNS
	if delta < hudUpdateNS {
		return
	}
	app.fpsCurrent = float64(app.windowFrames) * 1000000000.0 / float64(delta)
	totalDelta := now - app.runStartNS
	if totalDelta > 0 {
		app.fpsAverage = float64(app.totalFrames) * 1000000000.0 / float64(totalDelta)
	}
	if app.fpsCurrent > app.fpsBest {
		app.fpsBest = app.fpsCurrent
	}
	app.windowStartNS = now
	app.windowFrames = 0
	app.drawHeader()
	app.drawFooter()
	app.presenter.PresentRect(app.canvas, app.headerRect)
	app.presenter.PresentRect(app.canvas, app.footerRect)
	app.lastHUDNS = now
	app.updateConsoleTitle()
}

func (app *stressApp) redrawFull(now uint64) {
	app.drawChrome()
	if app.failed {
		app.drawFailureView()
		app.presenter.PresentFull(app.canvas)
		return
	}
	app.presenter.PresentFull(app.canvas)
	app.renderTinyGL(now)
}

func (app *stressApp) step() {
	now := kos.UptimeNanoseconds()
	if app.renderTinyGL(now) {
		app.totalFrames++
		app.windowFrames++
		app.updateFPS(now)
	}
}

func (app *stressApp) adjustDensity(delta int) {
	next := clampDensity(app.density + delta)
	if next == app.density {
		return
	}
	app.density = next
	now := kos.UptimeNanoseconds()
	app.failed = false
	app.failureText = ""
	app.resetStats(now)
	app.redrawFull(now)
}

func (app *stressApp) togglePause() {
	app.paused = !app.paused
	now := kos.UptimeNanoseconds()
	app.windowStartNS = now
	app.windowFrames = 0
	app.drawHeader()
	app.drawFooter()
	app.presenter.PresentRect(app.canvas, app.headerRect)
	app.presenter.PresentRect(app.canvas, app.footerRect)
	app.updateConsoleTitle()
}

func (app *stressApp) handleKey(key kos.KeyEvent) bool {
	switch {
	case key.Code == 27 || key.ScanCode == 1:
		return true
	case key.Code == ' ':
		app.togglePause()
	case key.Code == 'r' || key.Code == 'R':
		now := kos.UptimeNanoseconds()
		app.failed = false
		app.failureText = ""
		app.resetStats(now)
		app.redrawFull(now)
	case key.Code == '+' || key.Code == '=':
		app.adjustDensity(1)
	case key.Code == '-' || key.Code == '_':
		app.adjustDensity(-1)
	}
	return false
}

func (app *stressApp) run() {
	app.redrawFull(kos.UptimeNanoseconds())
	for {
		var event kos.EventType
		if app.paused || app.failed {
			event = kos.WaitEventFor(2)
		} else {
			event = kos.PollEvent()
		}
		switch event {
		case kos.EventNone:
			if !app.paused && !app.failed {
				app.step()
			}
		case kos.EventRedraw:
			app.redrawFull(kos.UptimeNanoseconds())
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		case kos.EventKey:
			if app.handleKey(kos.ReadKey()) {
				return
			}
		}
	}
}

func main() {
	app := newStressApp(readDensity())
	app.run()
	os.Exit(0)
}
