package surface

import (
	"bytes"
	"fmt"
	"image"
	gifimg "image/gif"
	jpegimg "image/jpeg"
	pngimg "image/png"
	"os"
	"sync"
)

var imageCache = struct {
	sync.Mutex
	entries map[string]*Image
}{
	entries: map[string]*Image{},
}

func GetImage(path string) *Image {
	if path == "" {
		return nil
	}
	imageCache.Lock()
	if entry, ok := imageCache.entries[path]; ok {
		imageCache.Unlock()
		return entry
	}
	imageCache.Unlock()
	entry, err := LoadImageFile(path)
	if err != nil {
		return nil
	}
	imageCache.Lock()
	imageCache.entries[path] = entry
	imageCache.Unlock()
	return entry
}

func MustLoadImageFile(path string) *Image {
	image, err := LoadImageFile(path)
	if err != nil {
		panic(err)
	}
	return image
}

func LoadImageFile(path string) (*Image, error) {
	if path == "" {
		return nil, fmt.Errorf("surface: empty image path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("surface: failed to read image %q: %w", path, err)
	}
	source, err := decodeRasterImage(data)
	if err != nil {
		return nil, fmt.Errorf("surface: failed to decode image %q: %w", path, err)
	}
	width, height, pixels := convertImagePixels(source)
	if width <= 0 || height <= 0 || len(pixels) == 0 {
		return nil, fmt.Errorf("surface: empty decoded image %q", path)
	}
	return &Image{
		Width:  width,
		Height: height,
		Pixels: pixels,
	}, nil
}

func decodeRasterImage(data []byte) (image.Image, error) {
	reader := bytes.NewReader(data)
	if isPNGData(data) {
		return pngimg.Decode(reader)
	}
	reader.Reset(data)
	if isJPEGData(data) {
		return jpegimg.Decode(reader)
	}
	reader.Reset(data)
	if isGIFData(data) {
		return gifimg.Decode(reader)
	}
	reader.Reset(data)
	source, _, err := image.Decode(reader)
	return source, err
}

func convertImagePixels(source image.Image) (int, int, []uint32) {
	if source == nil {
		return 0, 0, nil
	}
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return 0, 0, nil
	}
	pixels := make([]uint32, width*height)
	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, alpha := source.At(x, y).RGBA()
			pixels[index] = premultiplyPixel(
				uint8(red>>8),
				uint8(green>>8),
				uint8(blue>>8),
				uint8(alpha>>8),
			)
			index++
		}
	}
	return width, height, pixels
}

func premultiplyPixel(red uint8, green uint8, blue uint8, alpha uint8) uint32 {
	if alpha == 0 {
		return 0
	}
	if alpha >= 0xFF {
		return 0xFF000000 | uint32(red)<<16 | uint32(green)<<8 | uint32(blue)
	}
	a := uint32(alpha)
	r := (uint32(red)*a + 127) / 255
	g := (uint32(green)*a + 127) / 255
	b := (uint32(blue)*a + 127) / 255
	return a<<24 | r<<16 | g<<8 | b
}

func isPNGData(data []byte) bool {
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

func isJPEGData(data []byte) bool {
	return len(data) >= 3 &&
		data[0] == 0xFF &&
		data[1] == 0xD8 &&
		data[2] == 0xFF
}

func isGIFData(data []byte) bool {
	return len(data) >= 6 &&
		(string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a")
}
