package main

import (
	"io"
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
		source, ok := app.loadStylesheetText(href)
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

func (app *App) loadStylesheetText(rawURL string) (string, bool) {
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
		source := string(body)
		app.stylesheetCache[rawURL] = source
		return source, true
	}
	cachePath := app.resourceCachePath("css", rawURL, ".css")
	if data, ok := readCachedResource(cachePath); ok {
		source := string(data)
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
	request.Header.Set("Accept", "text/css,text/plain;q=0.9,*/*;q=0.1")
	request.Header.Set("User-Agent", "TagixBrowser/0.1")
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
	body, err := io.ReadAll(io.LimitReader(response.Body, maxStylesheetContent+1))
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
	source := string(body)
	writeCachedResource(cachePath, body)
	app.stylesheetCache[rawURL] = source
	if response.Request != nil && response.Request.URL != nil {
		if finalURL := strings.TrimSpace(response.Request.URL.String()); finalURL != "" && finalURL != rawURL {
			app.stylesheetCache[finalURL] = source
			if finalPath := app.resourceCachePath("css", finalURL, ".css"); finalPath != "" && finalPath != cachePath {
				writeCachedResource(finalPath, body)
			}
		}
	}
	return source, true
}
