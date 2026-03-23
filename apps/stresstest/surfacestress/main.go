package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"kos"
	"surface"
)

const (
	stressWindowX      = 84
	stressWindowY      = 60
	stressWindowWidth  = 860
	stressWindowHeight = 560
	stageFramesBase    = 12
)

const (
	colorBlack      kos.Color = surface.Black
	colorWhite      kos.Color = surface.White
	colorInk        kos.Color = 0x0B1620
	colorPanel      kos.Color = 0x152636
	colorPanel2     kos.Color = 0x1D3347
	colorPanel3     kos.Color = 0x27475F
	colorSlate      kos.Color = 0x365C78
	colorSilver     kos.Color = 0xC8D2D8
	colorMuted      kos.Color = 0x8DA2B2
	colorBlue       kos.Color = 0x4F7CFF
	colorCyan       kos.Color = 0x2EC4D6
	colorMint       kos.Color = 0x7AE582
	colorGold       kos.Color = 0xFFCF5C
	colorRose       kos.Color = 0xFF7A90
	colorRed        kos.Color = 0xF25F5C
	colorLine       kos.Color = 0x273D52
	colorLineBright kos.Color = 0x436785
)

type benchResult struct {
	name       string
	group      string
	iterations int
	operations int
	totalNS    uint64
	nsPerOp    uint64
}

type stageResult struct {
	name       string
	frames     int
	totalNS    uint64
	nsPerFrame uint64
}

type benchCase struct {
	name       string
	group      string
	iterations int
	operations int
	setup      func()
	fn         func(iter int)
}

type stressApp struct {
	presenter  surface.Presenter
	canvas     *surface.Buffer
	work       *surface.Buffer
	alphaWork  *surface.Buffer
	sample     *surface.Buffer
	alphaStamp *surface.Buffer
	font       *surface.Font
	fontLabel  string
	console    kos.Console
	consoleOK  bool
	scale      int
	runID      int
	status     string
	micro      []benchResult
	cpuStages  []stageResult
	present    stageResult
}

func alphaColor(color kos.Color, alpha uint8) kos.Color {
	return kos.Color(uint32(alpha)<<24 | (uint32(color) & 0xFFFFFF))
}

func uniformRadii(radius int) surface.CornerRadii {
	return surface.CornerRadii{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

func insetRect(rect surface.Rect, left int, top int, right int, bottom int) surface.Rect {
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

func (app *stressApp) stageGraphRect() surface.Rect {
	return surface.Rect{X: 18, Y: 242, Width: app.canvas.Width() - 36, Height: 128}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func readScale() int {
	scale := 1
	if len(os.Args) > 1 {
		if parsed, err := strconv.Atoi(os.Args[1]); err == nil && parsed > 0 {
			scale = parsed
		}
	}
	return scale
}

func loadStressFont() (*surface.Font, string) {
	candidates := []struct {
		path  string
		label string
	}{
		{path: "assets/OpenSans-Regular.ttf", label: "OpenSans 18"},
		{path: "../../examples/uiwindow/assets/OpenSans-Regular.ttf", label: "OpenSans 18"},
		{path: "apps/examples/uiwindow/assets/OpenSans-Regular.ttf", label: "OpenSans 18"},
		{path: "third_party/golang.org/x/image/font/gofont/ttfs/Go-Bold.ttf", label: "GoBold 18"},
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate.path); err != nil {
			continue
		}
		if font := surface.GetFont(candidate.path, 18); font != nil {
			return font, candidate.label
		}
	}
	return nil, "bitmap fallback"
}

func newStressApp(scale int) *stressApp {
	presenter := surface.NewPresenter(stressWindowX, stressWindowY, stressWindowWidth, stressWindowHeight, "Surface Stress")
	client := presenter.Client
	console, consoleOK := kos.OpenConsole("Surface Stress")
	font, fontLabel := loadStressFont()
	app := &stressApp{
		presenter:  presenter,
		canvas:     surface.NewBuffer(client.Width, client.Height),
		work:       surface.NewBuffer(640, 320),
		alphaWork:  surface.NewBufferAlpha(640, 320),
		sample:     surface.NewBuffer(240, 128),
		alphaStamp: surface.NewBufferAlpha(240, 128),
		font:       font,
		fontLabel:  fontLabel,
		console:    console,
		consoleOK:  consoleOK,
		scale:      scale,
	}
	if app.consoleOK && app.console.SupportsTitle() {
		app.console.SetTitle("Surface Stress / ready")
	}
	app.prepareSample()
	app.prepareAlphaStamp()
	return app
}

func (app *stressApp) logf(format string, args ...interface{}) {
	if !app.consoleOK {
		return
	}
	fmt.Printf(format, args...)
}

func (app *stressApp) setTitles(state string) {
	title := "Surface Stress"
	if state != "" {
		title += " / " + state
	}
	app.presenter.SetTitle(title)
	if app.consoleOK && app.console.SupportsTitle() {
		app.console.SetTitle(title)
	}
}

func (app *stressApp) warmTextCaches() {
	app.work.Clear(colorPanel)
	app.work.DrawText(12, 18, colorWhite, "surface stress warmup")
	if app.font != nil {
		app.work.DrawTextFont(12, 42, colorWhite, "surface stress warmup", app.font)
	}
}

func (app *stressApp) prepareSample() {
	app.sample.Clear(colorPanel2)
	app.sample.FillRectGradient(0, 0, app.sample.Width(), 24, surface.Gradient{
		From:      colorSlate,
		To:        colorPanel2,
		Direction: surface.GradientHorizontal,
	})
	app.sample.DrawText(10, 8, colorWhite, "sample source")
	for row := 0; row < 5; row++ {
		y := 34 + row*16
		color := colorSilver
		if row%3 == 1 {
			color = colorMint
		} else if row%3 == 2 {
			color = colorGold
		}
		app.sample.DrawText(12, y, color, "row "+strconv.Itoa(row)+" / blit / scroll / copy")
	}
	app.sample.StrokeRoundedRectWidth(4, 4, app.sample.Width()-8, app.sample.Height()-8, uniformRadii(12), 1, alphaColor(colorWhite, 44))
}

func (app *stressApp) prepareAlphaStamp() {
	app.alphaStamp.ClearTransparent()
	app.alphaStamp.FillRoundedRectGradientAlpha(10, 10, app.alphaStamp.Width()-20, app.alphaStamp.Height()-20, uniformRadii(18), surface.Gradient{
		From:      colorBlue,
		To:        colorCyan,
		Direction: surface.GradientHorizontal,
	}, 196)
	app.alphaStamp.FillRoundedRectAlpha(44, 34, 124, 42, uniformRadii(20), colorRose, 168)
	app.alphaStamp.DrawText(26, 28, colorWhite, "alpha stamp")
	app.alphaStamp.DrawText(56, 52, colorWhite, "blend path")
}

func (app *stressApp) prepareWorkPattern() {
	app.work.Clear(colorPanel)
	app.work.FillRectGradient(0, 0, app.work.Width(), 34, surface.Gradient{
		From:      colorPanel3,
		To:        colorPanel,
		Direction: surface.GradientHorizontal,
	})
	for x := 0; x < app.work.Width(); x += 32 {
		app.work.DrawLine(x, 34, x, app.work.Height()-1, colorLine)
	}
	for y := 34; y < app.work.Height(); y += 24 {
		app.work.DrawLine(0, y, app.work.Width()-1, y, colorLine)
	}
	app.work.DrawText(12, 10, colorWhite, "work pattern")
	for row := 0; row < 8; row++ {
		y := 50 + row*28
		app.work.DrawText(16, y, colorSilver, "line "+strconv.Itoa(row)+" / 0123456789 / surface")
	}
}

func (app *stressApp) preparePresentScene() {
	app.canvas.Clear(colorInk)
	bounds := app.canvas.Bounds()
	app.canvas.FillRectGradient(0, 0, bounds.Width, 64, surface.Gradient{
		From:      colorPanel2,
		To:        colorBlue,
		Direction: surface.GradientHorizontal,
	})
	app.canvas.DrawText(18, 16, colorWhite, "surface present benchmark")
	app.canvas.DrawText(18, 34, colorMuted, "present/full and present/rect are measured separately")
	panel := surface.Rect{X: 18, Y: 84, Width: bounds.Width - 36, Height: bounds.Height - 102}
	app.canvas.FillRoundedRectGradient(panel.X, panel.Y, panel.Width, panel.Height, uniformRadii(18), surface.Gradient{
		From:      colorPanel2,
		To:        colorPanel,
		Direction: surface.GradientVertical,
	})
	app.canvas.StrokeRoundedRectWidth(panel.X, panel.Y, panel.Width, panel.Height, uniformRadii(18), 1, alphaColor(colorWhite, 38))
	app.canvas.BlitFrom(app.sample, app.sample.Bounds(), panel.X+18, panel.Y+22)
	app.canvas.BlitFrom(app.alphaStamp, app.alphaStamp.Bounds(), panel.X+290, panel.Y+26)
	app.canvas.FillRoundedRectAlpha(panel.X+22, panel.Y+180, 220, 56, uniformRadii(18), colorMint, 96)
	app.canvas.DrawLine(panel.X+278, panel.Y+170, panel.X+panel.Width-30, panel.Y+46, colorGold)
	app.canvas.DrawLine(panel.X+278, panel.Y+52, panel.X+panel.Width-30, panel.Y+190, alphaColor(colorCyan, 200))
	app.canvas.DrawText(panel.X+28, panel.Y+196, colorWhite, "present target")
}

func (app *stressApp) runCase(spec benchCase) benchResult {
	if spec.setup != nil {
		spec.setup()
	}
	if spec.fn != nil {
		spec.fn(0)
	}
	start := kos.UptimeNanoseconds()
	for iter := 0; iter < spec.iterations; iter++ {
		spec.fn(iter)
	}
	elapsed := kos.UptimeNanoseconds() - start
	operations := spec.operations
	if operations <= 0 {
		operations = spec.iterations
	}
	if operations <= 0 {
		operations = 1
	}
	return benchResult{
		name:       spec.name,
		group:      spec.group,
		iterations: spec.iterations,
		operations: operations,
		totalNS:    elapsed,
		nsPerOp:    elapsed / uint64(operations),
	}
}

func (app *stressApp) runMicroBenchmarks() []benchResult {
	app.warmTextCaches()
	panelRect := surface.Rect{X: 18, Y: 18, Width: app.work.Width() - 36, Height: app.work.Height() - 36}
	rounded := uniformRadii(18)
	innerRounded := uniformRadii(24)
	lineCount := 48
	pixelOps := 512
	bitmapLines := 9
	fontLines := 8
	results := make([]benchResult, 0, 18)
	cases := []benchCase{
		{
			name:       "clear/full",
			group:      "fill",
			iterations: 180 * app.scale,
			fn: func(iter int) {
				app.work.Clear(colorPanel)
			},
		},
		{
			name:       "fill/rect",
			group:      "fill",
			iterations: 200 * app.scale,
			fn: func(iter int) {
				app.work.FillRect(20, 18, app.work.Width()-40, app.work.Height()-36, colorPanel3)
			},
		},
		{
			name:       "fill/rect-alpha",
			group:      "alpha",
			iterations: 180 * app.scale,
			setup:      app.prepareWorkPattern,
			fn: func(iter int) {
				app.work.FillRectAlpha(30, 24, app.work.Width()-60, app.work.Height()-48, colorBlue, 168)
			},
		},
		{
			name:       "fill/rounded",
			group:      "rounded",
			iterations: 140 * app.scale,
			fn: func(iter int) {
				app.work.FillRoundedRect(panelRect.X, panelRect.Y, panelRect.Width, panelRect.Height, rounded, colorPanel3)
			},
		},
		{
			name:       "gradient/h",
			group:      "gradient",
			iterations: 140 * app.scale,
			fn: func(iter int) {
				app.work.FillRectGradient(0, 0, app.work.Width(), app.work.Height(), surface.Gradient{
					From:      colorPanel2,
					To:        colorBlue,
					Direction: surface.GradientHorizontal,
				})
			},
		},
		{
			name:       "gradient/v",
			group:      "gradient",
			iterations: 140 * app.scale,
			fn: func(iter int) {
				app.work.FillRectGradient(0, 0, app.work.Width(), app.work.Height(), surface.Gradient{
					From:      colorPanel2,
					To:        colorMint,
					Direction: surface.GradientVertical,
				})
			},
		},
		{
			name:       "gradient/rounded-alpha",
			group:      "gradient",
			iterations: 110 * app.scale,
			setup:      app.prepareWorkPattern,
			fn: func(iter int) {
				app.work.FillRoundedRectGradientAreaAlpha(26, 20, app.work.Width()-52, app.work.Height()-40, innerRounded, surface.Gradient{
					From:      colorBlue,
					To:        colorRose,
					Direction: surface.GradientHorizontal,
				}, surface.Rect{X: 0, Y: 0, Width: app.work.Width() + 96, Height: app.work.Height()}, 172)
			},
		},
		{
			name:       "shadow+fill",
			group:      "shadow",
			iterations: 96 * app.scale,
			fn: func(iter int) {
				app.work.Clear(colorPanel)
				app.work.DrawShadowRounded(panelRect, surface.Shadow{
					OffsetX: 0,
					OffsetY: 2,
					Blur:    4,
					Color:   colorBlack,
					Alpha:   88,
				}, rounded)
				app.work.FillRoundedRect(panelRect.X, panelRect.Y, panelRect.Width, panelRect.Height, rounded, colorPanel3)
			},
		},
		{
			name:       "lines/fan",
			group:      "line",
			iterations: 72 * app.scale,
			operations: 72 * app.scale * lineCount,
			fn: func(iter int) {
				app.work.Clear(colorPanel)
				for line := 0; line < lineCount; line++ {
					y := 12 + line*6
					color := colorCyan
					if line%3 == 1 {
						color = colorGold
					} else if line%3 == 2 {
						color = colorMint
					}
					app.work.DrawLine(0, y, app.work.Width()-1, app.work.Height()-y-1, color)
				}
			},
		},
		{
			name:       "pixels/alpha",
			group:      "pixel",
			iterations: 84 * app.scale,
			operations: 84 * app.scale * pixelOps,
			setup:      app.prepareWorkPattern,
			fn: func(iter int) {
				for pixel := 0; pixel < pixelOps; pixel++ {
					x := 12 + ((pixel*11)+iter*7)%(app.work.Width()-24)
					y := 18 + ((pixel*7)+iter*13)%(app.work.Height()-36)
					alpha := uint8(96 + (pixel & 127))
					color := colorRose
					if pixel&1 == 0 {
						color = colorWhite
					}
					app.work.SetPixelAlpha(x, y, color, alpha)
				}
			},
		},
		{
			name:       "text/bitmap",
			group:      "text",
			iterations: 84 * app.scale,
			operations: 84 * app.scale * bitmapLines,
			fn: func(iter int) {
				app.work.FillRect(0, 0, app.work.Width(), app.work.Height(), colorPanel)
				for row := 0; row < bitmapLines; row++ {
					y := 12 + row*30
					app.work.DrawText(16, y, colorSilver, "bitmap text / glyph path / 0123456789 / surface bench")
				}
			},
		},
		{
			name:       "blit/from",
			group:      "blit",
			iterations: 220 * app.scale,
			fn: func(iter int) {
				x := 12 + (iter*17)%(app.work.Width()-app.sample.Width()-12)
				y := 18 + (iter*9)%(app.work.Height()-app.sample.Height()-18)
				app.work.BlitFrom(app.sample, app.sample.Bounds(), x, y)
			},
		},
		{
			name:       "blit/alpha",
			group:      "blit",
			iterations: 180 * app.scale,
			setup:      app.prepareWorkPattern,
			fn: func(iter int) {
				x := 24 + (iter*13)%(app.work.Width()-app.alphaStamp.Width()-24)
				y := 30 + (iter*11)%(app.work.Height()-app.alphaStamp.Height()-30)
				app.work.BlitFrom(app.alphaStamp, app.alphaStamp.Bounds(), x, y)
			},
		},
		{
			name:       "blit/self",
			group:      "blit",
			iterations: 190 * app.scale,
			setup:      app.prepareWorkPattern,
			fn: func(iter int) {
				if iter%32 == 0 {
					app.prepareWorkPattern()
				}
				app.work.BlitSelf(surface.Rect{X: 6, Y: 34, Width: app.work.Width() - 12, Height: app.work.Height() - 40}, 0, 28)
			},
		},
		{
			name:       "scroll/y",
			group:      "scroll",
			iterations: 180 * app.scale,
			setup:      app.prepareWorkPattern,
			fn: func(iter int) {
				if iter%40 == 0 {
					app.prepareWorkPattern()
				}
				app.work.ScrollRectY(surface.Rect{X: 8, Y: 36, Width: app.work.Width() - 16, Height: app.work.Height() - 44}, -8)
				app.work.FillRect(8, app.work.Height()-16, app.work.Width()-16, 8, colorGold)
			},
		},
	}
	if app.font != nil {
		cases = append(cases, benchCase{
			name:       "text/ttf-warm",
			group:      "text",
			iterations: 52 * app.scale,
			operations: 52 * app.scale * fontLines,
			fn: func(iter int) {
				app.work.FillRect(0, 0, app.work.Width(), app.work.Height(), colorPanel)
				for row := 0; row < fontLines; row++ {
					y := 12 + row*32
					app.work.DrawTextFont(16, y, colorWhite, "cached TTF draw / surface stress / abcXYZ012345", app.font)
				}
			},
		})
	}
	presentRectIterations := 34 * app.scale
	presentClientIterations := 16 * app.scale
	presentFullIterations := 8 * app.scale
	cases = append(cases,
		benchCase{
			name:       "present/rect",
			group:      "present",
			iterations: presentRectIterations,
			setup:      app.preparePresentScene,
			fn: func(iter int) {
				app.presenter.PresentRect(app.canvas, surface.Rect{X: 18, Y: 84, Width: app.canvas.Width() - 36, Height: app.canvas.Height() - 102})
			},
		},
		benchCase{
			name:       "present/client",
			group:      "present",
			iterations: presentClientIterations,
			setup:      app.preparePresentScene,
			fn: func(iter int) {
				app.presenter.PresentClient(app.canvas)
			},
		},
		benchCase{
			name:       "present/full",
			group:      "present",
			iterations: presentFullIterations,
			setup:      app.preparePresentScene,
			fn: func(iter int) {
				app.presenter.PresentFull(app.canvas)
			},
		},
	)
	for _, spec := range cases {
		results = append(results, app.runCase(spec))
	}
	sort.Slice(results, func(i int, j int) bool {
		if results[i].nsPerOp == results[j].nsPerOp {
			return results[i].totalNS > results[j].totalNS
		}
		return results[i].nsPerOp > results[j].nsPerOp
	})
	return results
}

func (app *stressApp) drawStageClear(frame int) {
	app.canvas.Clear(colorInk)
}

func (app *stressApp) drawStageHeader(frame int) {
	bounds := app.canvas.Bounds()
	app.canvas.FillRectGradient(0, 0, bounds.Width, 56, surface.Gradient{
		From:      colorPanel2,
		To:        colorBlue,
		Direction: surface.GradientHorizontal,
	})
	app.canvas.DrawText(18, 16, colorWhite, "surface frame-stage profile")
	app.canvas.DrawText(18, 34, colorMuted, "clear / header / rounded / shadow-fill / shadow-mask / vectors / text / blit / present")
}

func (app *stressApp) drawStagePanelsFill(frame int) {
	cards := []surface.Rect{
		{X: 18, Y: 78, Width: 246, Height: 144},
		{X: 282, Y: 78, Width: 246, Height: 144},
		{X: 546, Y: 78, Width: app.canvas.Width() - 564, Height: 144},
	}
	for index, rect := range cards {
		gradient := surface.Gradient{
			From:      colorPanel2,
			To:        colorPanel3,
			Direction: surface.GradientVertical,
		}
		if index == 1 {
			gradient.From = colorBlue
			gradient.To = colorPanel2
		} else if index == 2 {
			gradient.From = colorPanel3
			gradient.To = colorCyan
		}
		app.canvas.FillRoundedRectGradient(rect.X, rect.Y, rect.Width, rect.Height, uniformRadii(18), gradient)
	}
}

func (app *stressApp) drawStagePanelsStroke(frame int) {
	cards := []surface.Rect{
		{X: 18, Y: 78, Width: 246, Height: 144},
		{X: 282, Y: 78, Width: 246, Height: 144},
		{X: 546, Y: 78, Width: app.canvas.Width() - 564, Height: 144},
	}
	for _, rect := range cards {
		app.canvas.StrokeRoundedRectWidth(rect.X, rect.Y, rect.Width, rect.Height, uniformRadii(18), 1, alphaColor(colorWhite, 42))
	}
}

func (app *stressApp) drawStageShadowBase(frame int) {
	graph := app.stageGraphRect()
	app.canvas.FillRoundedRect(graph.X, graph.Y, graph.Width, graph.Height, uniformRadii(16), colorPanel)
}

func (app *stressApp) drawStageShadowMask(frame int) {
	graph := app.stageGraphRect()
	app.canvas.DrawShadowRounded(graph, surface.Shadow{
		OffsetX: 0,
		OffsetY: 2,
		Blur:    3,
		Color:   colorBlack,
		Alpha:   76,
	}, uniformRadii(16))
}

func (app *stressApp) drawStageVectorLines(frame int) {
	graph := app.stageGraphRect()
	for x := graph.X + 10; x < graph.X+graph.Width; x += 28 {
		app.canvas.DrawLine(x, graph.Y+10, x, graph.Y+graph.Height-10, colorLine)
	}
	for y := graph.Y + 12; y < graph.Y+graph.Height; y += 22 {
		app.canvas.DrawLine(graph.X+10, y, graph.X+graph.Width-10, y, colorLine)
	}
	height := graph.Height - 28
	for step := 0; step < 36; step++ {
		x0 := graph.X + 12 + step*20
		x1 := x0 + 20
		y0 := graph.Y + 14 + ((step*11)+(frame*7))%height
		y1 := graph.Y + 14 + (((step+1)*11)+(frame*7))%height
		y2 := graph.Y + 14 + ((step*17)+(frame*5))%height
		y3 := graph.Y + 14 + (((step+1)*17)+(frame*5))%height
		app.canvas.DrawLine(x0, y0, x1, y1, colorMint)
		app.canvas.DrawLine(x0, y2, x1, y3, colorGold)
	}
}

func (app *stressApp) drawStageText(frame int) {
	textRect := surface.Rect{X: 18, Y: 386, Width: app.canvas.Width() - 36, Height: 88}
	app.canvas.FillRoundedRect(textRect.X, textRect.Y, textRect.Width, textRect.Height, uniformRadii(14), colorPanel2)
	for row := 0; row < 3; row++ {
		y := textRect.Y + 12 + row*20
		app.canvas.DrawText(textRect.X+16, y, colorSilver, "bitmap text path / cached defaults / surface frame stage")
	}
	if app.font != nil {
		app.canvas.DrawTextFont(textRect.X+16, textRect.Y+68, colorWhite, "TTF steady-state path / cached glyphs / stage profile", app.font)
		return
	}
	app.canvas.DrawText(textRect.X+16, textRect.Y+68, colorMuted, "TTF path unavailable in this environment")
}

func (app *stressApp) drawStageBlits(frame int) {
	app.sample.ScrollRectY(surface.Rect{X: 8, Y: 34, Width: app.sample.Width() - 16, Height: 78}, -16)
	app.sample.FillRect(8, app.sample.Height()-18, app.sample.Width()-16, 16, colorGold)
	app.sample.DrawText(12, app.sample.Height()-16, colorBlack, "tail after scroll")
	app.sample.BlitSelf(surface.Rect{X: 8, Y: 8, Width: 96, Height: 14}, 128, 8)
	app.canvas.BlitFrom(app.sample, app.sample.Bounds(), 28, 94)
	app.canvas.BlitFrom(app.alphaStamp, app.alphaStamp.Bounds(), app.canvas.Width()-app.alphaStamp.Width()-28, 98)
}

func (app *stressApp) warmStageCaches() {
	app.canvas.Clear(colorInk)
	app.drawStageHeader(0)
	app.drawStagePanelsFill(0)
	app.drawStagePanelsStroke(0)
	app.drawStageShadowBase(0)
	app.drawStageShadowMask(0)
	app.drawStageText(0)
	app.prepareSample()
	app.drawStageBlits(0)
	app.drawStageClear(0)
}

func (app *stressApp) runStageProfile() ([]stageResult, stageResult) {
	type stageSpec struct {
		name string
		fn   func(frame int)
	}
	frames := stageFramesBase * app.scale
	if frames < stageFramesBase {
		frames = stageFramesBase
	}
	app.warmStageCaches()
	specs := []stageSpec{
		{name: "clear/full", fn: app.drawStageClear},
		{name: "header-static", fn: app.drawStageHeader},
		{name: "rounded-fill", fn: app.drawStagePanelsFill},
		{name: "rounded-stroke", fn: app.drawStagePanelsStroke},
		{name: "shadow-fill-base", fn: app.drawStageShadowBase},
		{name: "shadow-mask", fn: app.drawStageShadowMask},
		{name: "vector-lines", fn: app.drawStageVectorLines},
		{name: "text-blocks", fn: app.drawStageText},
		{name: "blit+scroll", fn: app.drawStageBlits},
	}
	totals := make(map[string]uint64, len(specs)+1)
	for frame := 0; frame < frames; frame++ {
		for _, spec := range specs {
			start := kos.UptimeNanoseconds()
			spec.fn(frame)
			totals[spec.name] += kos.UptimeNanoseconds() - start
		}
		start := kos.UptimeNanoseconds()
		app.presenter.PresentFull(app.canvas)
		totals["present-full"] += kos.UptimeNanoseconds() - start
	}
	results := make([]stageResult, 0, len(specs))
	for _, spec := range specs {
		total := totals[spec.name]
		results = append(results, stageResult{
			name:       spec.name,
			frames:     frames,
			totalNS:    total,
			nsPerFrame: total / uint64(frames),
		})
	}
	sort.Slice(results, func(i int, j int) bool {
		if results[i].nsPerFrame == results[j].nsPerFrame {
			return results[i].totalNS > results[j].totalNS
		}
		return results[i].nsPerFrame > results[j].nsPerFrame
	})
	return results, stageResult{
		name:       "present-full",
		frames:     frames,
		totalNS:    totals["present-full"],
		nsPerFrame: totals["present-full"] / uint64(frames),
	}
}

func totalMicroNS(results []benchResult) uint64 {
	var total uint64
	for _, result := range results {
		total += result.totalNS
	}
	return total
}

func totalStageNS(results []stageResult) uint64 {
	var total uint64
	for _, result := range results {
		total += result.totalNS
	}
	return total
}

func formatDurationShort(ns uint64) string {
	switch {
	case ns >= 1000000:
		return fmt.Sprintf("%7.2fms", float64(ns)/1000000.0)
	case ns >= 1000:
		return fmt.Sprintf("%7.2fus", float64(ns)/1000.0)
	default:
		return fmt.Sprintf("%7dns", ns)
	}
}

func formatMilliseconds(ns uint64) string {
	return fmt.Sprintf("%7.2fms", float64(ns)/1000000.0)
}

func formatStageShare(value uint64, total uint64) string {
	if total == 0 {
		return "  0.0%"
	}
	return fmt.Sprintf("%5.1f%%", float64(value)*100.0/float64(total))
}

func (app *stressApp) printReport() {
	if len(app.micro) == 0 || len(app.cpuStages) == 0 {
		return
	}
	app.logf("\nsurface stress run=%d scale=%d client=%dx%d font=%s\n",
		app.runID,
		app.scale,
		app.canvas.Width(),
		app.canvas.Height(),
		app.fontLabel,
	)
	app.logf("microbench ranked by time/op\n")
	app.logf("%-22s %-10s %8s %12s %12s\n", "name", "group", "ops", "total_ns", "time_per_op")
	for _, result := range app.micro {
		app.logf("%-22s %-10s %8d %12d %12d\n",
			result.name,
			result.group,
			result.operations,
			result.totalNS,
			result.nsPerOp,
		)
	}
	stageTotal := totalStageNS(app.cpuStages)
	app.logf("\ncpu frame stages ranked by ns/frame\n")
	app.logf("%-22s %8s %12s %10s\n", "name", "frames", "ns_per_frame", "share")
	for _, result := range app.cpuStages {
		app.logf("%-22s %8d %12d %10s\n",
			result.name,
			result.frames,
			result.nsPerFrame,
			formatStageShare(result.totalNS, stageTotal),
		)
	}
	app.logf("\npresent/full: frames=%d ns_per_frame=%d total_ns=%d\n",
		app.present.frames,
		app.present.nsPerFrame,
		app.present.totalNS,
	)
	app.logf("\n")
}

func (app *stressApp) drawPanel(rect surface.Rect, title string, subtitle string) {
	app.canvas.DrawShadowRounded(rect, surface.Shadow{
		OffsetX: 0,
		OffsetY: 2,
		Blur:    3,
		Color:   colorBlack,
		Alpha:   76,
	}, uniformRadii(16))
	app.canvas.FillRoundedRectGradient(rect.X, rect.Y, rect.Width, rect.Height, uniformRadii(16), surface.Gradient{
		From:      colorPanel2,
		To:        colorPanel,
		Direction: surface.GradientVertical,
	})
	app.canvas.StrokeRoundedRectWidth(rect.X, rect.Y, rect.Width, rect.Height, uniformRadii(16), 1, alphaColor(colorWhite, 38))
	if app.font != nil {
		app.canvas.DrawTextFont(rect.X+12, rect.Y+8, colorWhite, title, app.font)
		if subtitle != "" {
			app.canvas.DrawText(rect.X+12, rect.Y+32, colorMuted, subtitle)
		}
		return
	}
	app.canvas.DrawText(rect.X+12, rect.Y+10, colorWhite, title)
	if subtitle != "" {
		app.canvas.DrawText(rect.X+12, rect.Y+26, colorMuted, subtitle)
	}
}

func (app *stressApp) renderDashboard() {
	bounds := app.canvas.Bounds()
	app.canvas.Clear(colorInk)
	app.canvas.FillRectGradient(0, 0, bounds.Width, 56, surface.Gradient{
		From:      colorPanel2,
		To:        colorBlue,
		Direction: surface.GradientHorizontal,
	})
	if app.font != nil {
		app.canvas.DrawTextFont(18, 10, colorWhite, "surface: stress + timing profile", app.font)
	} else {
		app.canvas.DrawText(18, 14, colorWhite, "surface: stress + timing profile")
	}
	app.canvas.DrawText(18, 32, colorSilver, app.status)
	leftRect := surface.Rect{X: 18, Y: 74, Width: 452, Height: bounds.Height - 92}
	rightTop := surface.Rect{X: 488, Y: 74, Width: bounds.Width - 506, Height: 218}
	rightBottom := surface.Rect{X: 488, Y: 308, Width: bounds.Width - 506, Height: bounds.Height - 326}
	app.drawPanel(leftRect, "Slowest API Paths", "ranked by time/op")
	app.drawPanel(rightTop, "Slowest CPU Frame Stages", "offscreen only / time/frame")
	app.drawPanel(rightBottom, "Notes", "")
	app.canvas.DrawText(leftRect.X+16, leftRect.Y+52, colorMuted, "name               time/op    total")
	rowY := leftRect.Y + 72
	for index, result := range app.micro {
		if index >= 12 {
			break
		}
		color := colorSilver
		if index == 0 {
			color = colorRose
		} else if index == 1 {
			color = colorGold
		} else if index == 2 {
			color = colorMint
		}
		line := fmt.Sprintf("%-18s %9s %9s", result.name, formatDurationShort(result.nsPerOp), formatMilliseconds(result.totalNS))
		app.canvas.DrawText(leftRect.X+16, rowY, color, line)
		rowY += 18
	}
	stageTotal := totalStageNS(app.cpuStages)
	app.canvas.DrawText(rightTop.X+16, rightTop.Y+52, colorMuted, "stage                ms/frame   share")
	rowY = rightTop.Y + 72
	for index, result := range app.cpuStages {
		if index >= 8 {
			break
		}
		color := colorSilver
		if index == 0 {
			color = colorRose
		}
		line := fmt.Sprintf("%-18s %9s %8s", result.name, formatMilliseconds(result.nsPerFrame), formatStageShare(result.totalNS, stageTotal))
		app.canvas.DrawText(rightTop.X+16, rowY, color, line)
		rowY += 18
	}
	noteY := rightBottom.Y + 18
	cpuFrameTotal := totalStageNS(app.cpuStages)
	fullFrameTotal := cpuFrameTotal + app.present.totalNS
	notes := []string{
		"Press R, Space or Enter to rerun.",
		"Press Esc or close button to exit.",
		"Microbench total: " + formatMilliseconds(totalMicroNS(app.micro)),
		"CPU frame total/frame: " + formatMilliseconds(cpuFrameTotal/uint64(maxInt(1, app.present.frames))),
		"Present full/frame: " + formatMilliseconds(app.present.nsPerFrame),
		"CPU+present/frame: " + formatMilliseconds(fullFrameTotal/uint64(maxInt(1, app.present.frames))),
		"Font path: " + app.fontLabel,
		"Scale: " + strconv.Itoa(app.scale) + "x",
		"CPU stage profile excludes PresentFull.",
		"CPU stage profile is measured after cache warmup.",
		"present/full uses the real window blit path.",
		"text/ttf-warm excludes font file load and first glyph rasterization.",
	}
	for _, note := range notes {
		app.canvas.DrawText(rightBottom.X+16, noteY, colorSilver, note)
		noteY += 18
	}
	if len(app.micro) > 0 {
		top := app.micro[0]
		app.canvas.DrawText(rightBottom.X+16, noteY+6, colorWhite, "Top API suspect: "+top.name+" / "+top.group)
	}
	if len(app.cpuStages) > 0 {
		top := app.cpuStages[0]
		app.canvas.DrawText(rightBottom.X+16, noteY+24, colorWhite, "Top CPU frame suspect: "+top.name)
	}
	if app.present.nsPerFrame > 0 {
		app.canvas.DrawText(rightBottom.X+16, noteY+42, colorWhite, "Present path: "+formatMilliseconds(app.present.nsPerFrame)+" / frame")
	}
}

func (app *stressApp) rerun() {
	app.runID++
	app.setTitles("running")
	app.status = "run " + strconv.Itoa(app.runID) + ": measuring offscreen paths and frame stages..."
	app.renderDashboard()
	app.presenter.PresentFull(app.canvas)
	app.micro = app.runMicroBenchmarks()
	app.cpuStages, app.present = app.runStageProfile()
	topMicro := ""
	topCPU := ""
	if len(app.micro) > 0 {
		topMicro = app.micro[0].name
	}
	if len(app.cpuStages) > 0 {
		topCPU = app.cpuStages[0].name
	}
	app.status = "run " + strconv.Itoa(app.runID) + ": top api=" + topMicro + " / top cpu-frame=" + topCPU + " / present=" + app.present.name
	app.printReport()
	app.renderDashboard()
	app.setTitles("done")
	app.presenter.PresentFull(app.canvas)
}

func (app *stressApp) run() {
	app.rerun()
	for {
		switch kos.WaitEventFor(4) {
		case kos.EventRedraw:
			app.renderDashboard()
			app.presenter.PresentFull(app.canvas)
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		case kos.EventKey:
			key := kos.ReadKey()
			switch {
			case key.Code == 27 || key.ScanCode == 1:
				return
			case key.Code == 13 || key.Code == 32 || key.Code == 'r' || key.Code == 'R':
				app.rerun()
			}
		}
	}
}

func main() {
	app := newStressApp(readScale())
	app.run()
	os.Exit(0)
}
