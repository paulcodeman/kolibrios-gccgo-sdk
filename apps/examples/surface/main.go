package main

import (
	"os"

	"kos"
	"surface"
)

const (
	galleryWindowX      = 56
	galleryWindowY      = 48
	galleryWindowWidth  = 640
	galleryWindowHeight = 440
)

const (
	colorBlack      kos.Color = surface.Black
	colorWhite      kos.Color = surface.White
	colorGray       kos.Color = 0x7A8793
	colorSilver     kos.Color = 0xCBD4DB
	colorInk        kos.Color = 0x10202E
	colorSlate      kos.Color = 0x172B3A
	colorPanel      kos.Color = 0x233A4D
	colorPanelDeep  kos.Color = 0x1A2E3F
	colorPanelLight kos.Color = 0x2D4A62
	colorCyan       kos.Color = 0x2EC4D6
	colorBlue       kos.Color = 0x4F7CFF
	colorMint       kos.Color = 0x7AE582
	colorGold       kos.Color = 0xFFCF5C
	colorRose       kos.Color = 0xFF7A90
)

type galleryApp struct {
	presenter surface.Presenter
	canvas    *surface.Buffer
	overlay   *surface.Buffer
	sample    *surface.Buffer
	font      *surface.Font
}

func alphaColor(color kos.Color, alpha uint8) kos.Color {
	return kos.Color(uint32(alpha)<<24 | (uint32(color) & 0xFFFFFF))
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

func uniformRadii(radius int) surface.CornerRadii {
	return surface.CornerRadii{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

func galleryFont() *surface.Font {
	for _, path := range []string{
		"assets/OpenSans-Regular.ttf",
		"../uiwindow/assets/OpenSans-Regular.ttf",
		"apps/examples/uiwindow/assets/OpenSans-Regular.ttf",
		"third_party/golang.org/x/image/font/gofont/ttfs/Go-Bold.ttf",
	} {
		if _, err := os.Stat(path); err == nil {
			if font := surface.GetFont(path, 18); font != nil {
				return font
			}
		}
	}
	return nil
}

func newGalleryApp() *galleryApp {
	presenter := surface.NewPresenter(galleryWindowX, galleryWindowY, galleryWindowWidth, galleryWindowHeight, "Surface Gallery")
	client := presenter.Client
	return &galleryApp{
		presenter: presenter,
		canvas:    surface.NewBuffer(client.Width, client.Height),
		overlay:   surface.NewBufferAlpha(196, 128),
		sample:    surface.NewBuffer(228, 96),
		font:      galleryFont(),
	}
}

func (app *galleryApp) drawPanel(rect surface.Rect, title string, gradient surface.Gradient) {
	radii := uniformRadii(14)
	app.canvas.DrawShadowRounded(rect, surface.Shadow{
		OffsetX: 0,
		OffsetY: 2,
		Blur:    3,
		Color:   colorBlack,
		Alpha:   72,
	}, radii)
	app.canvas.FillRoundedRectGradient(rect.X, rect.Y, rect.Width, rect.Height, radii, gradient)
	app.canvas.StrokeRoundedRectWidth(rect.X, rect.Y, rect.Width, rect.Height, radii, 1, alphaColor(colorWhite, 52))
	if app.font != nil {
		app.canvas.DrawTextFont(rect.X+12, rect.Y+8, colorWhite, title, app.font)
		return
	}
	app.canvas.DrawText(rect.X+12, rect.Y+10, colorWhite, title)
}

func (app *galleryApp) drawClipDemo(rect surface.Rect) {
	inner := insetRect(rect, 12, 28, 12, 12)
	app.canvas.FillRoundedRectGradientArea(inner.X, inner.Y, inner.Width, inner.Height, uniformRadii(10), surface.Gradient{
		From:      colorPanelLight,
		To:        colorPanelDeep,
		Direction: surface.GradientHorizontal,
	}, surface.Rect{X: inner.X - 42, Y: inner.Y, Width: inner.Width + 84, Height: inner.Height})
	clip := insetRect(inner, 10, 10, 10, 10)
	app.canvas.PushClip(clip)
	for i := 0; i < 14; i++ {
		y := clip.Y + i*7
		color := colorCyan
		if i%3 == 1 {
			color = colorBlue
		} else if i%3 == 2 {
			color = colorMint
		}
		app.canvas.DrawLine(clip.X-80, y+24, clip.X+clip.Width+72, y-18, alphaColor(color, 170))
	}
	app.canvas.DrawText(clip.X-8, clip.Y+16, colorWhite, "clipped text / gradient area / diagonal stripes")
	app.canvas.DrawText(clip.X+12, clip.Y+34, colorGold, "long line left <<<< clipped >>>> right")
	app.canvas.DrawText(clip.X+24, clip.Y+52, colorSilver, "PushClip + PopClip keep the draw bounds local")
	app.canvas.PopClip()
}

func (app *galleryApp) drawShapeDemo(rect surface.Rect) {
	inner := insetRect(rect, 12, 28, 12, 12)
	card := insetRect(inner, 10, 6, inner.Width/2, 10)
	app.canvas.FillRoundedRectGradient(card.X, card.Y, card.Width, card.Height, uniformRadii(16), surface.Gradient{
		From:      alphaColor(colorBlue, 230),
		To:        alphaColor(colorCyan, 230),
		Direction: surface.GradientVertical,
	})
	app.canvas.StrokeRoundedRectWidth(card.X, card.Y, card.Width, card.Height, uniformRadii(16), 2, alphaColor(colorWhite, 120))
	app.canvas.DrawShadowRounded(surface.Rect{X: card.X + 12, Y: card.Y + 12, Width: 54, Height: 34}, surface.Shadow{
		OffsetX: 2,
		OffsetY: 2,
		Blur:    4,
		Color:   colorBlack,
		Alpha:   88,
	}, uniformRadii(8))
	app.canvas.FillRoundedRect(card.X+12, card.Y+12, 54, 34, uniformRadii(8), colorMint)
	app.canvas.FillRoundedRectAlpha(card.X+80, card.Y+18, 72, 22, uniformRadii(11), colorRose, 168)
	app.canvas.DrawText(card.X+20, card.Y+58, colorWhite, "rounded fill / stroke / alpha")
	graph := insetRect(surface.Rect{X: card.X + card.Width + 12, Y: inner.Y + 4, Width: inner.Width - card.Width - 22, Height: inner.Height - 8}, 0, 0, 0, 0)
	app.canvas.FillRoundedRect(graph.X, graph.Y, graph.Width, graph.Height, uniformRadii(10), alphaColor(colorSlate, 245))
	for i := 0; i <= 4; i++ {
		y := graph.Y + 10 + i*16
		app.canvas.DrawLine(graph.X+8, y, graph.X+graph.Width-10, y, alphaColor(colorWhite, 24))
	}
	app.canvas.DrawLine(graph.X+12, graph.Y+graph.Height-12, graph.X+graph.Width-18, graph.Y+14, colorGold)
	app.canvas.DrawLine(graph.X+12, graph.Y+14, graph.X+graph.Width-18, graph.Y+graph.Height-20, alphaColor(colorCyan, 180))
	app.canvas.DrawText(graph.X+12, graph.Y+10, colorSilver, "DrawLine")
}

func (app *galleryApp) drawAlphaDemo(rect surface.Rect) {
	inner := insetRect(rect, 12, 28, 12, 12)
	app.overlay.ClearTransparent()
	app.overlay.FillRoundedRectGradient(22, 12, 116, 72, uniformRadii(18), surface.Gradient{
		From:      alphaColor(colorBlue, 168),
		To:        alphaColor(colorCyan, 232),
		Direction: surface.GradientHorizontal,
	})
	app.overlay.FillRoundedRectAlpha(68, 34, 94, 52, uniformRadii(18), colorRose, 180)
	app.overlay.StrokeRoundedRectWidth(22, 12, 116, 72, uniformRadii(18), 2, alphaColor(colorWhite, 120))
	app.overlay.StrokeRoundedRectWidth(68, 34, 94, 52, uniformRadii(18), 2, alphaColor(colorWhite, 90))
	app.overlay.DrawText(34, 24, colorWhite, "alpha")
	app.overlay.DrawText(82, 48, colorWhite, "overlay")
	app.overlay.DrawShadowRounded(surface.Rect{X: 20, Y: 90, Width: 132, Height: 20}, surface.Shadow{
		OffsetX: 0,
		OffsetY: 1,
		Blur:    3,
		Color:   colorBlack,
		Alpha:   96,
	}, uniformRadii(10))
	app.overlay.FillRoundedRectAlpha(28, 92, 132, 20, uniformRadii(10), colorMint, 100)
	app.canvas.FillRoundedRect(inner.X, inner.Y, inner.Width, inner.Height, uniformRadii(10), colorPanelDeep)
	app.canvas.BlitFrom(app.overlay, app.overlay.Bounds(), inner.X+8, inner.Y+4)
	app.canvas.DrawText(inner.X+12, inner.Y+96, colorSilver, "NewBufferAlpha + BlitFrom")
}

func (app *galleryApp) drawScrollDemo(rect surface.Rect) {
	inner := insetRect(rect, 12, 28, 12, 12)
	app.sample.Clear(colorPanelDeep)
	app.sample.FillRectGradient(0, 0, app.sample.Width(), 22, surface.Gradient{
		From:      colorPanelLight,
		To:        colorPanelDeep,
		Direction: surface.GradientHorizontal,
	})
	app.sample.DrawText(8, 6, colorWhite, "source buffer")
	for i := 0; i < 4; i++ {
		y := 26 + i*16
		color := colorSilver
		if i == 3 {
			color = colorMint
		}
		app.sample.DrawText(10, y, color, "row sample")
	}
	app.sample.ScrollRectY(surface.Rect{X: 8, Y: 24, Width: app.sample.Width() - 16, Height: 56}, -16)
	app.sample.FillRect(8, 68, app.sample.Width()-16, 16, colorGold)
	app.sample.DrawText(12, 70, colorBlack, "new tail after ScrollRectY")
	app.sample.BlitSelf(surface.Rect{X: 8, Y: 6, Width: 84, Height: 12}, 128, 6)
	app.sample.StrokeRoundedRectWidth(4, 4, app.sample.Width()-8, app.sample.Height()-8, uniformRadii(10), 1, alphaColor(colorWhite, 40))
	app.canvas.BlitFrom(app.sample, app.sample.Bounds(), inner.X+2, inner.Y+2)
	app.canvas.DrawText(inner.X+12, inner.Y+inner.Height-18, colorSilver, "BlitSelf mirrors the header block")
}

func (app *galleryApp) draw() {
	buffer := app.canvas
	bounds := buffer.Bounds()
	buffer.Clear(colorInk)
	buffer.FillRectGradient(0, 0, bounds.Width, 54, surface.Gradient{
		From:      colorPanel,
		To:        colorBlue,
		Direction: surface.GradientHorizontal,
	})
	if app.font != nil {
		buffer.DrawTextFont(18, 10, colorWhite, "surface: static gallery", app.font)
		buffer.DrawText(18, 32, colorSilver, "surface.GetFont now owns the TTF path, core stays raw")
	} else {
		buffer.DrawText(18, 14, colorWhite, "surface: static gallery")
		buffer.DrawText(18, 30, colorSilver, "rounded fill / gradient area / clip / alpha / blit / scroll")
	}

	leftTop := surface.Rect{X: 18, Y: 72, Width: 286, Height: 148}
	rightTop := surface.Rect{X: 318, Y: 72, Width: 286, Height: 148}
	leftBottom := surface.Rect{X: 18, Y: 236, Width: 286, Height: 148}
	rightBottom := surface.Rect{X: 318, Y: 236, Width: 286, Height: 148}

	app.drawPanel(leftTop, "Clip + Gradient Area", surface.Gradient{
		From:      colorPanel,
		To:        colorPanelDeep,
		Direction: surface.GradientVertical,
	})
	app.drawClipDemo(leftTop)

	app.drawPanel(rightTop, "Rounded Shapes", surface.Gradient{
		From:      colorPanel,
		To:        colorPanelLight,
		Direction: surface.GradientVertical,
	})
	app.drawShapeDemo(rightTop)

	app.drawPanel(leftBottom, "Alpha Overlay", surface.Gradient{
		From:      colorPanel,
		To:        colorPanelDeep,
		Direction: surface.GradientVertical,
	})
	app.drawAlphaDemo(leftBottom)

	app.drawPanel(rightBottom, "Blit + Scroll", surface.Gradient{
		From:      colorPanel,
		To:        colorPanelLight,
		Direction: surface.GradientVertical,
	})
	app.drawScrollDemo(rightBottom)
}

func (app *galleryApp) run() {
	app.draw()
	app.presenter.PresentFull(app.canvas)
	for {
		switch kos.WaitEvent() {
		case kos.EventRedraw:
			app.draw()
			app.presenter.PresentFull(app.canvas)
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
	app := newGalleryApp()
	app.run()
	os.Exit(0)
}
