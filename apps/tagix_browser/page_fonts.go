package main

import (
	"kos"
	nethttp "net/http"
	"os"
	pathpkg "path"
	filepathpkg "path/filepath"
	"strings"
)

const (
	maxFontContent = 4 * 1024 * 1024
	bundledFontDir = "assets/fonts"
)

type fontFamilyEntry struct {
	key  string
	path string
}

func (app *App) primeDocumentFontFaces(doc *Document, baseURL string) {
	if app == nil || doc == nil {
		return
	}
	doc.fontFamilies = nil
}

func collectBundledFontFamilies() []fontFamilyEntry {
	registry := []fontFamilyEntry{}
	registry = indexFontDirectory(registry, bundledFontDir)
	if len(registry) == 0 {
		return nil
	}
	return registry
}

func indexFontDirectory(registry []fontFamilyEntry, dir string) []fontFamilyEntry {
	if strings.TrimSpace(dir) == "" {
		return registry
	}
	if registry == nil {
		registry = []fontFamilyEntry{}
	}
	register := func(key string, path string) {
		key = strings.TrimSpace(key)
		path = strings.TrimSpace(path)
		if key == "" || path == "" {
			return
		}
		for index := range registry {
			if registry[index].key != key {
				continue
			}
			if registry[index].path == "" {
				registry[index].path = path
			}
			return
		}
		registry = append(registry, fontFamilyEntry{key: key, path: path})
	}
	start := uint32(0)
	for {
		result, status := kos.ReadDirectory(dir, start, 64)
		if status != kos.FileSystemOK && status != kos.FileSystemEOF {
			return registry
		}
		if len(result.Entries) == 0 {
			return registry
		}
		for _, entry := range result.Entries {
			if entry.Info.Attributes&kos.FileAttributeDirectory != 0 {
				continue
			}
			path := filepathpkg.Join(dir, entry.Name)
			if !fontPathSupported(path) {
				continue
			}
			for _, key := range fontLookupKeys(entry.Name) {
				if key == "" {
					continue
				}
				register(key, path)
			}
		}
		start += uint32(len(result.Entries))
		if status == kos.FileSystemEOF || (result.Total > 0 && start >= result.Total) {
			return registry
		}
	}
}

func fontLookupKeys(name string) []string {
	stem := strings.TrimSpace(strings.TrimSuffix(filepathpkg.Base(name), filepathpkg.Ext(name)))
	if stem == "" {
		return nil
	}
	keys := make([]string, 0, 2)
	appendKey := func(value string) {
		key := normalizeCSSFontFamilyName(value)
		if key == "" {
			return
		}
		for _, existing := range keys {
			if existing == key {
				return
			}
		}
		keys = append(keys, key)
	}
	appendKey(stem)
	appendKey(trimFontStyleSuffix(stem))
	return keys
}

func trimFontStyleSuffix(stem string) string {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return ""
	}
	for {
		sep := strings.LastIndexAny(stem, "-_ ")
		if sep <= 0 || sep+1 >= len(stem) {
			return stem
		}
		suffix := normalizeCSSFontFamilyName(stem[sep+1:])
		switch suffix {
		case "regular", "normal", "roman", "book", "medium",
			"semibold", "demibold", "bold", "extrabold", "ultrabold",
			"light", "extralight", "thin", "hairline", "black", "heavy",
			"italic", "oblique", "condensed", "narrow":
			stem = strings.TrimSpace(stem[:sep])
		default:
			return stem
		}
	}
}

func registerFontFamilyPath(registry []fontFamilyEntry, key string, path string) []fontFamilyEntry {
	key = strings.TrimSpace(key)
	path = strings.TrimSpace(path)
	if key == "" || path == "" {
		return registry
	}
	for index := range registry {
		if registry[index].key != key {
			continue
		}
		registry[index].path = path
		return registry
	}
	return append(registry, fontFamilyEntry{key: key, path: path})
}

func (app *App) collectDocumentFontFaces(registry []fontFamilyEntry, source string, baseURL string) []fontFamilyEntry {
	if app == nil || len(registry) == 0 && strings.TrimSpace(source) == "" {
		return registry
	}
	source = stripCSSComments(source)
	lower := strings.ToLower(source)
	for index := 0; index < len(source); {
		match := strings.Index(lower[index:], "@font-face")
		if match < 0 {
			break
		}
		match += index
		brace := strings.IndexByte(source[match:], '{')
		if brace < 0 {
			break
		}
		brace += match
		end := matchingCSSBrace(source, brace)
		if end < 0 {
			break
		}
		family, candidates := parseCSSFontFaceBlock(source[brace+1 : end])
		if family != "" && len(candidates) > 0 {
			for _, candidate := range candidates {
				if path, ok := app.loadFontFaceResource(candidate, baseURL); ok && strings.TrimSpace(path) != "" {
					registry = registerFontFamilyPath(registry, family, path)
					break
				}
			}
		}
		index = end + 1
	}
	return registry
}

func matchingCSSBrace(source string, start int) int {
	if start < 0 || start >= len(source) || source[start] != '{' {
		return -1
	}
	depth := 0
	quote := byte(0)
	for index := start; index < len(source); index++ {
		ch := source[index]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func parseCSSFontFaceBlock(block string) (string, []string) {
	family := ""
	candidates := make([]string, 0, 2)
	for _, declaration := range splitCSSDeclarations(block) {
		colon := strings.IndexByte(declaration, ':')
		if colon <= 0 || colon+1 >= len(declaration) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(declaration[:colon]))
		value := strings.TrimSpace(declaration[colon+1:])
		switch name {
		case "font-family":
			if family == "" {
				family = firstCSSFontFamilyName(value)
			}
		case "src":
			candidates = append(candidates, fontFaceSourceCandidates(value)...)
		}
	}
	return family, candidates
}

func splitCSSDeclarations(source string) []string {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	parts := make([]string, 0, 8)
	start := 0
	depth := 0
	quote := byte(0)
	for index := 0; index < len(source); index++ {
		ch := source[index]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 {
				part := strings.TrimSpace(source[start:index])
				if part != "" {
					parts = append(parts, part)
				}
				start = index + 1
			}
		}
	}
	if tail := strings.TrimSpace(source[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

func firstCSSFontFamilyName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return ""
	}
	return normalizeCSSFontFamilyName(parts[0])
}

func fontFaceSourceCandidates(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	candidates := make([]string, 0, 4)
	lower := strings.ToLower(value)
	for index := 0; index < len(value); {
		match := strings.Index(lower[index:], "url(")
		if match < 0 {
			break
		}
		match += index
		end := cssURLFunctionEnd(value, match+4)
		if end < 0 {
			break
		}
		raw := value[match : end+1]
		if resolved, ok := extractCSSURLValue(raw); ok {
			if candidate := strings.TrimSpace(resolved); candidate != "" {
				candidates = append(candidates, candidate)
			}
		}
		index = end + 1
	}
	return candidates
}

func (app *App) loadFontFaceResource(rawURL string, referrer string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if app == nil || rawURL == "" {
		return "", false
	}
	if !strings.HasPrefix(toLowerASCII(rawURL), "data:") {
		rawURL = resolveURL(referrer, rawURL)
		if rawURL == "" {
			return "", false
		}
	}
	if cached := app.cachedFontResourcePath(rawURL); cached != "" {
		return cached, true
	}
	if strings.HasPrefix(toLowerASCII(rawURL), "data:") {
		data, ok := decodeDataResource(rawURL)
		if !ok || len(data) == 0 {
			return "", false
		}
		cachePath := app.resourceCachePath("font", rawURL, fontCacheSuffix(rawURL, ""))
		if writeCachedResource(cachePath, data) {
			return cachePath, true
		}
		return "", false
	}
	if path, ok := fileURLPath(rawURL); ok {
		if info, err := os.Stat(path); err == nil && info != nil && !info.IsDir() && fontPathSupported(path) {
			return path, true
		}
		return "", false
	}
	if app.httpClient == nil {
		return "", false
	}
	request, err := nethttp.NewRequest(nethttp.MethodGet, rawURL, nil)
	if err != nil {
		app.debugError("font request "+rawURL, err)
		return "", false
	}
	app.applyFontRequestHeaders(request, referrer)
	response, err := app.httpClient.Do(request)
	if err != nil {
		app.debugError("font get "+rawURL, err)
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		app.debugf("font status %s for %s", response.Status, rawURL)
		return "", false
	}
	body, err := readDecodedHTTPResponseBody(response, maxFontContent)
	if err != nil || len(body) == 0 {
		if err != nil {
			app.debugError("font read body "+rawURL, err)
		}
		return "", false
	}
	finalURL := rawURL
	if response.Request != nil && response.Request.URL != nil {
		if resolved := strings.TrimSpace(response.Request.URL.String()); resolved != "" {
			finalURL = resolved
		}
	}
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if !fontResponseSupported(finalURL, contentType) {
		app.debugf("font unsupported type %q for %s", contentType, rawURL)
		return "", false
	}
	suffix := fontCacheSuffix(finalURL, contentType)
	cachePath := app.resourceCachePath("font", rawURL, suffix)
	if !writeCachedResource(cachePath, body) {
		return "", false
	}
	if finalURL != "" && finalURL != rawURL {
		if finalPath := app.resourceCachePath("font", finalURL, suffix); finalPath != "" && finalPath != cachePath {
			writeCachedResource(finalPath, body)
			return finalPath, true
		}
	}
	return cachePath, true
}

func (app *App) cachedFontResourcePath(rawURL string) string {
	if app == nil {
		return ""
	}
	for _, suffix := range []string{".ttf", ".otf", ".ttc"} {
		path := app.resourceCachePath("font", rawURL, suffix)
		if _, ok := readCachedResource(path); ok {
			return path
		}
	}
	return ""
}

func fontResponseSupported(rawURL string, contentType string) bool {
	if fontPathSupported(rawURL) {
		return true
	}
	ext := strings.ToLower(fontPathExtension(rawURL))
	if ext != "" {
		return false
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if semicolon := strings.IndexByte(contentType, ';'); semicolon >= 0 {
		contentType = strings.TrimSpace(contentType[:semicolon])
	}
	switch contentType {
	case "font/ttf",
		"font/otf",
		"font/sfnt",
		"application/font-sfnt",
		"application/x-font-ttf",
		"application/x-font-opentype",
		"application/octet-stream":
		return true
	default:
		return false
	}
}

func fontPathSupported(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(toLowerASCII(raw), "data:") {
		return true
	}
	ext := strings.ToLower(fontPathExtension(raw))
	switch ext {
	case ".ttf", ".otf", ".ttc":
		return true
	default:
		return false
	}
}

func fontCacheSuffix(rawURL string, contentType string) string {
	if ext := strings.ToLower(fontPathExtension(rawURL)); ext == ".ttf" || ext == ".otf" || ext == ".ttc" {
		return ext
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(contentType, "otf"), strings.Contains(contentType, "opentype"):
		return ".otf"
	case strings.Contains(contentType, "ttc"):
		return ".ttc"
	default:
		return ".ttf"
	}
}

func fontPathExtension(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(toLowerASCII(raw), "data:") {
		header := raw[len("data:"):]
		if comma := strings.IndexByte(header, ','); comma >= 0 {
			header = header[:comma]
		}
		header = strings.ToLower(strings.TrimSpace(header))
		switch {
		case strings.Contains(header, "font/otf"), strings.Contains(header, "opentype"):
			return ".otf"
		case strings.Contains(header, "font/ttc"):
			return ".ttc"
		default:
			return ".ttf"
		}
	}
	if path, ok := fileURLPath(raw); ok {
		return filepathpkg.Ext(stripFragment(stripQuery(path)))
	}
	if strings.Contains(raw, "://") {
		return pathpkg.Ext(stripFragment(stripQuery(raw)))
	}
	return filepathpkg.Ext(stripFragment(stripQuery(raw)))
}
