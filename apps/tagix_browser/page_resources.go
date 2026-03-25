package main

import (
	nethttp "net/http"
	"os"
	"strings"
)

const maxStylesheetContent = 256 * 1024

func (app *App) inlineLinkedStylesheets(doc *Document, baseURL string) {
	if app == nil || doc == nil {
		return
	}
	links := doc.GetElementsByTagName("link")
	if len(links) == 0 {
		return
	}
	target := doc.GetElementsByTagName("head")
	parent := doc.Root
	if len(target) > 0 && target[0] != nil {
		parent = target[0]
	}
	injected := 0
	for _, link := range links {
		if link == nil || !isStylesheetLink(link) {
			continue
		}
		href := resolveURL(baseURL, attrValue(link, "href"))
		if href == "" {
			continue
		}
		source, ok := app.loadStylesheetText(href, baseURL)
		if !ok || strings.TrimSpace(source) == "" {
			continue
		}
		style := doc.CreateElement("style")
		doc.AppendChild(style, doc.CreateText(source))
		doc.AppendChild(parent, style)
		injected++
	}
	if injected > 0 {
		doc.stylesheet = nil
		doc.stylesheetParsed = false
	}
}

func isStylesheetLink(node *Node) bool {
	if node == nil || node.Type != ElementNode || node.Tag != "link" {
		return false
	}
	rel := toLowerASCII(attrValue(node, "rel"))
	if rel == "" {
		return false
	}
	for _, item := range strings.Fields(rel) {
		if item == "stylesheet" {
			return attrValue(node, "href") != ""
		}
	}
	return false
}

func (app *App) loadStylesheetText(rawURL string, referrer string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if app == nil || rawURL == "" {
		return "", false
	}
	if cached, ok := app.stylesheetCache[rawURL]; ok {
		return cached, cached != ""
	}
	if path, ok := fileURLPath(rawURL); ok {
		body, err := os.ReadFile(path)
		if err != nil || len(body) == 0 {
			if err != nil {
				app.debugError("css read "+path, err)
			} else {
				app.debugf("css read %s: empty file", path)
			}
			app.stylesheetCache[rawURL] = ""
			return "", false
		}
		source := rewriteStylesheetAssetURLs(string(body), rawURL)
		app.stylesheetCache[rawURL] = source
		return source, true
	}
	cachePath := app.resourceCachePath("css", rawURL, ".css")
	if data, ok := readCachedResource(cachePath); ok {
		source := rewriteStylesheetAssetURLs(string(data), rawURL)
		app.stylesheetCache[rawURL] = source
		return source, true
	}
	if app.httpClient == nil {
		app.debugf("css fetch disabled for %s: http client unavailable", rawURL)
		app.stylesheetCache[rawURL] = ""
		return "", false
	}
	request, err := nethttp.NewRequest(nethttp.MethodGet, rawURL, nil)
	if err != nil {
		app.debugError("css request "+rawURL, err)
		app.stylesheetCache[rawURL] = ""
		return "", false
	}
	app.applyStylesheetRequestHeaders(request, referrer)
	response, err := app.httpClient.Do(request)
	if err != nil {
		app.debugError("css get "+rawURL, err)
		app.stylesheetCache[rawURL] = ""
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		app.debugf("css status %s for %s", response.Status, rawURL)
		app.stylesheetCache[rawURL] = ""
		return "", false
	}
	body, err := readDecodedHTTPResponseBody(response, maxStylesheetContent)
	if err != nil || len(body) == 0 {
		if err != nil {
			app.debugError("css read body "+rawURL, err)
		} else {
			app.debugf("css read body %s: empty response", rawURL)
		}
		app.stylesheetCache[rawURL] = ""
		return "", false
	}
	if len(body) > maxStylesheetContent {
		body = body[:maxStylesheetContent]
	}
	finalURL := rawURL
	if response.Request != nil && response.Request.URL != nil {
		if resolved := strings.TrimSpace(response.Request.URL.String()); resolved != "" {
			finalURL = resolved
		}
	}
	source := rewriteStylesheetAssetURLs(string(body), finalURL)
	rewrittenBody := []byte(source)
	writeCachedResource(cachePath, rewrittenBody)
	app.stylesheetCache[rawURL] = source
	if finalURL != "" && finalURL != rawURL {
		app.stylesheetCache[finalURL] = source
		if finalPath := app.resourceCachePath("css", finalURL, ".css"); finalPath != "" && finalPath != cachePath {
			writeCachedResource(finalPath, rewrittenBody)
		}
	}
	return source, true
}

func rewriteStylesheetAssetURLs(source string, stylesheetURL string) string {
	source = strings.TrimSpace(source)
	stylesheetURL = strings.TrimSpace(stylesheetURL)
	if source == "" || stylesheetURL == "" {
		return source
	}
	var builder strings.Builder
	builder.Grow(len(source) + 32)
	for index := 0; index < len(source); {
		match := strings.Index(strings.ToLower(source[index:]), "url(")
		if match < 0 {
			builder.WriteString(source[index:])
			break
		}
		match += index
		builder.WriteString(source[index:match])
		end := cssURLFunctionEnd(source, match+4)
		if end < 0 {
			builder.WriteString(source[match:])
			break
		}
		rawValue := source[match : end+1]
		builder.WriteString(rewriteCSSURLFunction(rawValue, stylesheetURL))
		index = end + 1
	}
	return builder.String()
}

func cssURLFunctionEnd(source string, start int) int {
	depth := 1
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
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func rewriteCSSURLFunction(raw string, stylesheetURL string) string {
	if imageURL, ok := extractCSSURLValue(raw); ok {
		trimmed := strings.TrimSpace(imageURL)
		lower := toLowerASCII(trimmed)
		switch {
		case trimmed == "",
			strings.HasPrefix(lower, "data:"),
			strings.HasPrefix(lower, "blob:"),
			strings.HasPrefix(lower, "about:"),
			strings.HasPrefix(lower, "#"):
			return raw
		}
		resolved := resolveURL(stylesheetURL, trimmed)
		if resolved == "" {
			return raw
		}
		quote := byte('"')
		inner := strings.TrimSpace(raw[4 : len(raw)-1])
		if len(inner) >= 2 {
			if inner[0] == '\'' || inner[0] == '"' {
				quote = inner[0]
			}
		}
		return "url(" + string(quote) + resolved + string(quote) + ")"
	}
	return raw
}
