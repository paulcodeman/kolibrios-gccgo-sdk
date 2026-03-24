package main

import (
	"os"
	"path/filepath"
	"strings"
)

func normalizeURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "about:") {
		end := len(value)
		if pos := indexByte(value, '?'); pos >= 0 && pos < end {
			end = pos
		}
		if pos := indexByte(value, '#'); pos >= 0 && pos < end {
			end = pos
		}
		return strings.ToLower(value[:end]) + value[end:]
	}
	if strings.HasPrefix(lower, "file://") {
		if path, ok := fileURLPath(value); ok {
			return fileURLFromPath(path)
		}
		return value
	}
	if path, ok := localHTMLPathFromURL(value); ok {
		return fileURLFromPath(path)
	}
	if strings.Contains(value, "://") {
		return value
	}
	return "http://" + value
}

func resolveURL(baseURL string, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	lower := toLowerASCII(href)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "mailto:") {
		return ""
	}
	if strings.Contains(href, "://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		scheme := urlScheme(baseURL)
		if scheme == "" || scheme == "file" {
			scheme = "http"
		}
		return scheme + ":" + href
	}
	if strings.HasPrefix(href, "#") {
		return stripFragment(baseURL) + href
	}
	if strings.HasPrefix(href, "?") {
		return stripQuery(stripFragment(baseURL)) + href
	}
	if isFileURL(baseURL) {
		return resolveFileURL(baseURL, href)
	}

	scheme, host, path := splitURL(baseURL)
	if scheme == "" || host == "" {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return scheme + "://" + host + href
	}

	baseDir := basePathDir(path)
	href = cleanRelative(href)
	for strings.HasPrefix(href, "../") {
		href = href[3:]
		baseDir = parentDir(baseDir)
	}
	if strings.HasPrefix(href, "./") {
		href = href[2:]
	}
	return scheme + "://" + host + baseDir + href
}

func appendURLQuery(rawURL string, encoded string) string {
	rawURL = strings.TrimSpace(rawURL)
	encoded = strings.TrimSpace(encoded)
	if rawURL == "" || encoded == "" {
		return rawURL
	}
	fragment := ""
	if pos := indexByte(rawURL, '#'); pos >= 0 {
		fragment = rawURL[pos:]
		rawURL = rawURL[:pos]
	}
	separator := "?"
	if strings.Contains(rawURL, "?") {
		separator = "&"
		if strings.HasSuffix(rawURL, "?") || strings.HasSuffix(rawURL, "&") {
			separator = ""
		}
	}
	return rawURL + separator + encoded + fragment
}

func splitURL(raw string) (string, string, string) {
	pos := strings.Index(raw, "://")
	if pos < 0 {
		return "", "", ""
	}
	scheme := raw[:pos]
	rest := raw[pos+3:]
	slash := indexByte(rest, '/')
	if slash < 0 {
		return scheme, rest, "/"
	}
	return scheme, rest[:slash], rest[slash:]
}

func urlScheme(raw string) string {
	pos := strings.Index(raw, "://")
	if pos < 0 {
		return ""
	}
	return raw[:pos]
}

func basePathDir(path string) string {
	path = stripFragment(path)
	path = stripQuery(path)
	if path == "" {
		return "/"
	}
	last := lastIndexByte(path, '/')
	if last < 0 {
		return "/"
	}
	return path[:last+1]
}

func stripQuery(path string) string {
	if pos := indexByte(path, '?'); pos >= 0 {
		return path[:pos]
	}
	return path
}

func stripFragment(path string) string {
	if pos := indexByte(path, '#'); pos >= 0 {
		return path[:pos]
	}
	return path
}

func parentDir(path string) string {
	if path == "" {
		return "/"
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	last := lastIndexByte(path, '/')
	if last < 0 {
		return "/"
	}
	return path[:last+1]
}

func cleanRelative(path string) string {
	for strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	return path
}

func resolveStartupTarget(args []string) (string, bool) {
	for _, raw := range args {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if path, ok := localHTMLPathFromArg(value); ok {
			return fileURLFromPath(path), true
		}
		return normalizeURL(value), false
	}
	return defaultURL, false
}

func localHTMLPathFromArg(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(value), "about:") {
		return "", false
	}
	return localHTMLPathFromURL(value)
}

func localHTMLPathFromURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if path, ok := fileURLPath(value); ok {
		return ensureLocalHTMLPath(path)
	}
	if strings.Contains(value, "://") {
		return "", false
	}
	return ensureLocalHTMLPath(value)
}

func ensureLocalHTMLPath(value string) (string, bool) {
	value = stripFragment(stripQuery(strings.TrimSpace(value)))
	if !looksLikeHTMLPath(value) {
		return "", false
	}
	resolved, err := filepath.Abs(value)
	if err == nil {
		value = resolved
	}
	info, err := os.Stat(value)
	if err != nil || info == nil || info.IsDir() {
		return "", false
	}
	return filepath.Clean(value), true
}

func looksLikeHTMLPath(value string) bool {
	ext := strings.ToLower(filepath.Ext(value))
	switch ext {
	case ".html", ".htm", ".xhtml":
		return true
	default:
		return false
	}
}

func isFileURL(value string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "file://")
}

func fileURLFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	resolved, err := filepath.Abs(path)
	if err == nil {
		path = resolved
	}
	path = filepath.ToSlash(filepath.Clean(path))
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "file://" + path
}

func fileURLPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if !isFileURL(raw) {
		return "", false
	}
	path := stripFragment(stripQuery(raw[len("file://"):]))
	if path == "" {
		return "", false
	}
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.Clean(path), true
}

func resolveFileURL(baseURL string, href string) string {
	basePath, ok := fileURLPath(baseURL)
	if !ok {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return fileURLFromPath(filepath.FromSlash(href))
	}
	baseDir := filepath.Dir(basePath)
	resolved := filepath.Join(baseDir, filepath.FromSlash(href))
	return fileURLFromPath(resolved)
}
