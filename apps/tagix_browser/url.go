package main

import "strings"

func normalizeURL(value string) string {
	value = strings.TrimSpace(value)
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
		if scheme == "" {
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
