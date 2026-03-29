package core

import (
	"bytes"
	"image"
	gifimg "image/gif"
	jpegimg "image/jpeg"
	pngimg "image/png"
	"os"
	"sync"

	"kos"
)

const (
	windowStyleCaptionBit            uint32    = 0x10000000
	windowTitleIconGap                         = 6
	windowTitleFallbackLeftInset               = 6
	windowTitleFallbackRightInset              = 54
	windowTitleFallbackVerticalInset           = 2
	windowTitleFallbackTextColor     kos.Color = 0xFFFFFF
)

type presenterTitleImage struct {
	Width  int
	Height int
	Pixels []uint32
	Opaque bool
}

var presenterTitleImageCache = struct {
	sync.Mutex
	entries map[string]*presenterTitleImage
}{
	entries: map[string]*presenterTitleImage{},
}

func (presenter Presenter) customTitleBarEnabled() bool {
	if presenter.TitleIconPath == "" {
		return false
	}
	if presenter.windowStyle() == kos.WindowStyleBorderless {
		return false
	}
	return presenter.Width > 0 && presenter.Height > 0 && kos.SkinHeight() > 0
}

func (presenter Presenter) systemTitle() string {
	if presenter.customTitleBarEnabled() {
		return ""
	}
	return presenter.Title
}

func (presenter Presenter) presentWindowStyle() uint32 {
	style := presenter.windowStyle()
	if presenter.customTitleBarEnabled() {
		return style &^ windowStyleCaptionBit
	}
	return style
}

func (presenter Presenter) drawCustomTitleBar() {
	if !presenter.customTitleBarEnabled() {
		return
	}
	titleRect := presenter.customTitleRect()
	if titleRect.Empty() {
		return
	}

	textX := titleRect.X
	if titleRect.Height > 0 {
		iconSize := titleRect.Height
		if iconSize > titleRect.Width {
			iconSize = titleRect.Width
		}
		if iconSize > 0 {
			iconRect := Rect{
				X:      textX,
				Y:      titleRect.Y + (titleRect.Height-iconSize)/2,
				Width:  iconSize,
				Height: iconSize,
			}
			presenter.drawCustomTitleIcon(iconRect)
			textX = iconRect.X + iconRect.Width + windowTitleIconGap
		}
	}

	maxTextWidth := titleRect.X + titleRect.Width - textX
	title := fitPresenterTitle(presenter.Title, maxTextWidth)
	if title == "" {
		return
	}

	textY := titleRect.Y
	if titleRect.Height > DefaultFontHeight {
		textY += (titleRect.Height - DefaultFontHeight) / 2
	}
	kos.DrawText(textX, textY, presenter.titleTextColor(), title)
}

func (presenter Presenter) customTitleRect() Rect {
	if presenter.Width <= 0 {
		return Rect{}
	}
	skinHeight := kos.SkinHeight()
	if skinHeight <= 0 {
		return Rect{}
	}
	margins := kos.WindowSkinMargins()
	left := margins.Left
	top := margins.Top
	right := presenter.Width - margins.Right
	bottom := skinHeight - margins.Bottom
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right > presenter.Width {
		right = presenter.Width
	}
	if bottom > skinHeight {
		bottom = skinHeight
	}
	if right <= left || bottom <= top {
		left = windowTitleFallbackLeftInset
		top = windowTitleFallbackVerticalInset
		right = presenter.Width - windowTitleFallbackRightInset
		bottom = skinHeight - windowTitleFallbackVerticalInset
	}
	if right <= left || bottom <= top {
		return Rect{}
	}
	return Rect{
		X:      left,
		Y:      top,
		Width:  right - left,
		Height: bottom - top,
	}
}

func (presenter Presenter) titleTextColor() kos.Color {
	colors := kos.StandardWindowColors()
	if colors.GrabText != 0 {
		return colors.GrabText
	}
	return windowTitleFallbackTextColor
}

func (presenter Presenter) drawCustomTitleIcon(rect Rect) bool {
	if rect.Empty() || presenter.TitleIconPath == "" {
		return false
	}
	image := presenterTitleImageForPath(presenter.TitleIconPath)
	if image == nil || image.Width <= 0 || image.Height <= 0 || len(image.Pixels) < image.Width*image.Height {
		return false
	}
	background := presenter.captureWindowRect(rect)
	if background == nil {
		return false
	}
	background.DrawImagePixelsRect(Rect{Width: rect.Width, Height: rect.Height}, image.Width, image.Height, image.Pixels, image.Opaque)
	background.BlitToWindow(rect.X, rect.Y)
	return true
}

func (presenter Presenter) captureWindowRect(rect Rect) *Buffer {
	if rect.Empty() {
		return nil
	}
	windowX, windowY := presenter.windowScreenPosition()
	raw := make([]byte, rect.Width*rect.Height*3)
	if !kos.ReadScreenArea(raw, rect.Width, rect.Height, windowX+rect.X, windowY+rect.Y) {
		return nil
	}
	buffer := NewBuffer(rect.Width, rect.Height)
	if buffer == nil || len(buffer.data) < rect.Width*rect.Height+2 {
		return nil
	}
	pixels := buffer.data[2:]
	for index := range pixels {
		offset := index * 3
		blue := uint32(raw[offset+0])
		green := uint32(raw[offset+1])
		red := uint32(raw[offset+2])
		pixels[index] = 0xFF000000 | red<<16 | green<<8 | blue
	}
	return buffer
}

func (presenter Presenter) windowScreenPosition() (x int, y int) {
	info, _, ok := kos.ReadCurrentThreadInfo()
	if ok && info.WindowSize.X > 0 && info.WindowSize.Y > 0 {
		return info.WindowPosition.X, info.WindowPosition.Y
	}
	return presenter.X, presenter.Y
}

func fitPresenterTitle(title string, maxWidth int) string {
	if title == "" || maxWidth < DefaultCharWidth {
		return ""
	}
	maxCols := maxWidth / DefaultCharWidth
	if maxCols <= 0 {
		return ""
	}
	cols := textColumnCount(title)
	if cols <= maxCols {
		return title
	}
	if maxCols <= 3 {
		return textSliceColumns(title, 0, maxCols)
	}
	return textSliceColumns(title, 0, maxCols-3) + "..."
}

func presenterTitleImageForPath(path string) *presenterTitleImage {
	if path == "" {
		return nil
	}
	presenterTitleImageCache.Lock()
	if entry, ok := presenterTitleImageCache.entries[path]; ok {
		presenterTitleImageCache.Unlock()
		return entry
	}
	presenterTitleImageCache.Unlock()

	entry := loadPresenterTitleImage(path)

	presenterTitleImageCache.Lock()
	presenterTitleImageCache.entries[path] = entry
	presenterTitleImageCache.Unlock()
	return entry
}

func loadPresenterTitleImage(path string) *presenterTitleImage {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	source, err := decodePresenterTitleImage(data)
	if err != nil {
		return nil
	}
	width, height, pixels, opaque := convertPresenterTitleImagePixels(source)
	if width <= 0 || height <= 0 || len(pixels) != width*height {
		return nil
	}
	return &presenterTitleImage{
		Width:  width,
		Height: height,
		Pixels: pixels,
		Opaque: opaque,
	}
}

func decodePresenterTitleImage(data []byte) (image.Image, error) {
	reader := bytes.NewReader(data)
	if isPresenterTitlePNG(data) {
		return pngimg.Decode(reader)
	}
	reader.Reset(data)
	if isPresenterTitleJPEG(data) {
		return jpegimg.Decode(reader)
	}
	reader.Reset(data)
	if isPresenterTitleGIF(data) {
		return gifimg.Decode(reader)
	}
	reader.Reset(data)
	source, _, err := image.Decode(reader)
	return source, err
}

func convertPresenterTitleImagePixels(source image.Image) (width int, height int, pixels []uint32, opaque bool) {
	if source == nil {
		return 0, 0, nil, false
	}
	bounds := source.Bounds()
	width = bounds.Dx()
	height = bounds.Dy()
	if width <= 0 || height <= 0 {
		return 0, 0, nil, false
	}
	pixels = make([]uint32, width*height)
	opaque = true
	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, alpha := source.At(x, y).RGBA()
			alpha8 := uint8(alpha >> 8)
			if alpha8 != 0xFF {
				opaque = false
			}
			pixels[index] = presenterPremultiplyPixel(
				uint8(red>>8),
				uint8(green>>8),
				uint8(blue>>8),
				alpha8,
			)
			index++
		}
	}
	return width, height, pixels, opaque
}

func presenterPremultiplyPixel(red uint8, green uint8, blue uint8, alpha uint8) uint32 {
	if alpha == 0 {
		return 0
	}
	if alpha == 0xFF {
		return 0xFF000000 | uint32(red)<<16 | uint32(green)<<8 | uint32(blue)
	}
	a := uint32(alpha)
	r := (uint32(red)*a + 127) / 255
	g := (uint32(green)*a + 127) / 255
	b := (uint32(blue)*a + 127) / 255
	return a<<24 | r<<16 | g<<8 | b
}

func isPresenterTitlePNG(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 &&
		data[1] == 0x50 &&
		data[2] == 0x4E &&
		data[3] == 0x47 &&
		data[4] == 0x0D &&
		data[5] == 0x0A &&
		data[6] == 0x1A &&
		data[7] == 0x0A
}

func isPresenterTitleJPEG(data []byte) bool {
	return len(data) >= 3 &&
		data[0] == 0xFF &&
		data[1] == 0xD8 &&
		data[2] == 0xFF
}

func isPresenterTitleGIF(data []byte) bool {
	return len(data) >= 6 &&
		(string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a")
}
