package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"io"
	nethttp "net/http"
	neturl "net/url"
	"os"
	pathpkg "path"
	"strings"
	"ui"

	webpimg "golang.org/x/image/webp"
	gifimg "image/gif"
	jpegimg "image/jpeg"
	pngimg "image/png"
)

const maxImageContent = 4 * 1024 * 1024

var (
	errImageEmpty  = errors.New("empty image data")
	errImageDecode = errors.New("image decode failed")
	errImageSize   = errors.New("invalid image size")
)

func imageResourceURL(baseURL string, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(toLowerASCII(raw), "data:") {
		return raw
	}
	return resolveURL(baseURL, raw)
}

func (app *App) loadDocumentImage(rawURL string) *ui.DocumentImage {
	rawURL = strings.TrimSpace(rawURL)
	if app == nil || rawURL == "" {
		return nil
	}
	if image, ok := app.imageCache[rawURL]; ok {
		return image
	}
	app.clearImageError(rawURL)
	data, finalURL, cachePath, fromCache, ok := app.loadImageResourceBytes(rawURL, true)
	if !ok || len(data) == 0 {
		app.debugf("img fetch failed url=%s", rawURL)
		app.setImageError(rawURL, "fetch failed")
		return nil
	}
	image, err := decodeDocumentImage(data)
	if image == nil && fromCache && cachePath != "" {
		app.debugf("img decode retry url=%s cache=%s kind=%s size=%d err=%v", rawURL, cachePath, detectImageKind(data), len(data), err)
		_ = os.Remove(cachePath)
		if finalURL != "" && finalURL != rawURL {
			if finalPath := app.resourceCachePath("img", finalURL, imageCacheSuffix(finalURL)); finalPath != "" && finalPath != cachePath {
				_ = os.Remove(finalPath)
			}
		}
		data, finalURL, _, _, ok = app.loadImageResourceBytes(rawURL, false)
		if ok && len(data) != 0 {
			image, err = decodeDocumentImage(data)
		}
	}
	if image == nil {
		app.debugf("img decode failed url=%s final=%s kind=%s size=%d cache=%t err=%v", rawURL, finalURL, detectImageKind(data), len(data), fromCache, err)
		app.setImageError(rawURL, describeImageDecodeError(err))
		if finalURL != "" && finalURL != rawURL {
			app.setImageError(finalURL, describeImageDecodeError(err))
		}
		return nil
	}
	app.clearImageError(rawURL)
	app.imageCache[rawURL] = image
	if finalURL != "" && finalURL != rawURL {
		app.clearImageError(finalURL)
		app.imageCache[finalURL] = image
	}
	return image
}

func (app *App) loadImageResourceBytes(rawURL string, allowCache bool) ([]byte, string, string, bool, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if app == nil || rawURL == "" {
		return nil, "", "", false, false
	}
	if strings.HasPrefix(toLowerASCII(rawURL), "data:") {
		data, ok := decodeDataResource(rawURL)
		if !ok {
			app.debugf("img data url decode failed url=%s", abbreviateDebugURL(rawURL))
		}
		return data, rawURL, "", false, ok
	}
	if path, ok := fileURLPath(rawURL); ok {
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			if err != nil {
				app.debugError("img read "+path, err)
			} else {
				app.debugf("img read %s: empty file", path)
			}
			return nil, "", "", false, false
		}
		return data, rawURL, path, false, true
	}
	cachePath := app.resourceCachePath("img", rawURL, imageCacheSuffix(rawURL))
	if allowCache {
		if data, ok := readCachedResource(cachePath); ok {
			return data, rawURL, cachePath, true, true
		}
	}
	if app.httpClient == nil {
		app.debugf("img fetch disabled for %s: http client unavailable", rawURL)
		return nil, "", "", false, false
	}
	request, err := nethttp.NewRequest(nethttp.MethodGet, rawURL, nil)
	if err != nil {
		app.debugError("img request "+rawURL, err)
		return nil, "", "", false, false
	}
	request.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif,image/*;q=0.8,*/*;q=0.1")
	request.Header.Set("User-Agent", "TagixBrowser/0.1")
	response, err := app.httpClient.Do(request)
	if err != nil {
		app.debugError("img get "+rawURL, err)
		return nil, "", "", false, false
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		app.debugf("img status %s for %s", response.Status, rawURL)
		return nil, "", "", false, false
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxImageContent+1))
	if err != nil || len(body) == 0 || len(body) > maxImageContent {
		if err != nil {
			app.debugError("img read body "+rawURL, err)
		} else if len(body) == 0 {
			app.debugf("img read body %s: empty response", rawURL)
		} else {
			app.debugf("img read body %s: too large (%d bytes)", rawURL, len(body))
		}
		return nil, "", "", false, false
	}
	writeCachedResource(cachePath, body)
	finalURL := rawURL
	if response.Request != nil && response.Request.URL != nil {
		if resolved := strings.TrimSpace(response.Request.URL.String()); resolved != "" {
			finalURL = resolved
		}
	}
	if finalURL != "" && finalURL != rawURL {
		if finalPath := app.resourceCachePath("img", finalURL, imageCacheSuffix(finalURL)); finalPath != "" && finalPath != cachePath {
			writeCachedResource(finalPath, body)
		}
	}
	return body, finalURL, cachePath, false, true
}

func imageCacheSuffix(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ".img"
	}
	if strings.HasPrefix(toLowerASCII(rawURL), "data:") {
		header := rawURL[len("data:"):]
		if comma := strings.Index(header, ","); comma >= 0 {
			header = header[:comma]
		}
		header = toLowerASCII(header)
		switch {
		case strings.Contains(header, "image/png"):
			return ".png"
		case strings.Contains(header, "image/jpeg"):
			return ".jpg"
		case strings.Contains(header, "image/gif"):
			return ".gif"
		case strings.Contains(header, "image/webp"):
			return ".webp"
		}
		return ".img"
	}
	parsed, err := neturl.Parse(rawURL)
	if err == nil && parsed != nil {
		if ext := sanitizeImageCacheExt(pathpkg.Ext(parsed.Path)); ext != "" {
			return ext
		}
	}
	if ext := sanitizeImageCacheExt(pathpkg.Ext(rawURL)); ext != "" {
		return ext
	}
	return ".img"
}

func sanitizeImageCacheExt(ext string) string {
	ext = toLowerASCII(strings.TrimSpace(ext))
	if len(ext) < 2 || len(ext) > 8 || ext[0] != '.' {
		return ""
	}
	for index := 1; index < len(ext); index++ {
		ch := ext[index]
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') {
			return ""
		}
	}
	return ext
}

func decodeDocumentImage(data []byte) (*ui.DocumentImage, error) {
	if len(data) == 0 {
		return nil, errImageEmpty
	}
	source, err := decodeDocumentRaster(data)
	if err != nil || source == nil {
		if err == nil {
			err = errImageDecode
		}
		return nil, err
	}
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, errImageSize
	}
	pixels := make([]uint32, width*height)
	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := source.At(x, y).RGBA()
			pixels[index] = premultiplyDocumentPixel(
				uint8(r>>8),
				uint8(g>>8),
				uint8(b>>8),
				uint8(a>>8),
			)
			index++
		}
	}
	return &ui.DocumentImage{
		Width:  width,
		Height: height,
		Pixels: pixels,
	}, nil
}

func decodeDocumentRaster(data []byte) (image.Image, error) {
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
	if isWEBPData(data) {
		return webpimg.Decode(reader)
	}
	reader.Reset(data)
	source, _, err := image.Decode(reader)
	return source, err
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

func isWEBPData(data []byte) bool {
	return len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"
}

func premultiplyDocumentPixel(red uint8, green uint8, blue uint8, alpha uint8) uint32 {
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

func (app *App) setImageError(rawURL string, reason string) {
	if app == nil || app.imageErrors == nil {
		return
	}
	rawURL = strings.TrimSpace(rawURL)
	reason = strings.TrimSpace(reason)
	if rawURL == "" || reason == "" {
		return
	}
	app.imageErrors[rawURL] = reason
}

func (app *App) clearImageError(rawURL string) {
	if app == nil || app.imageErrors == nil {
		return
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}
	delete(app.imageErrors, rawURL)
}

func describeImageDecodeError(err error) string {
	if err == nil {
		return "image unavailable"
	}
	switch err {
	case errImageEmpty:
		return "empty image data"
	case errImageDecode:
		return "image decode failed"
	case errImageSize:
		return "invalid image size"
	default:
		return strings.TrimSpace(err.Error())
	}
}

func detectImageKind(data []byte) string {
	switch {
	case isPNGData(data):
		return "png"
	case isJPEGData(data):
		return "jpeg"
	case isGIFData(data):
		return "gif"
	case isWEBPData(data):
		return "webp"
	case len(data) == 0:
		return "empty"
	default:
		size := len(data)
		if size > 8 {
			size = 8
		}
		return fmt.Sprintf("unknown:%x", data[:size])
	}
}

func abbreviateDebugURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if len(rawURL) <= 96 {
		return rawURL
	}
	return rawURL[:93] + "..."
}

func decodeDataResource(raw string) ([]byte, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(toLowerASCII(raw), "data:") {
		return nil, false
	}
	payload := raw[len("data:"):]
	comma := strings.Index(payload, ",")
	if comma < 0 {
		return nil, false
	}
	header := payload[:comma]
	dataPart := payload[comma+1:]
	if strings.Contains(header, ";base64") {
		encoding := base64.RawStdEncoding
		if len(dataPart)%4 == 0 {
			encoding = base64.StdEncoding
		}
		data, err := encoding.DecodeString(dataPart)
		if err != nil {
			return nil, false
		}
		return data, true
	}
	data, err := neturl.QueryUnescape(dataPart)
	if err != nil {
		return nil, false
	}
	return []byte(data), true
}
