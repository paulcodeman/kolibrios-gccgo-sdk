package main

import (
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"kos"
	nethttp "net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type browserRequestProfile struct {
	userAgent      string
	acceptLanguage string
	platform       string
	secCHUA        string
	screenWidth    int
	screenHeight   int
}

func newBrowserRequestProfile() browserRequestProfile {
	screenWidth, screenHeight := kos.ScreenSize()
	if screenWidth <= 0 {
		screenWidth = defaultWindowWidth
	}
	if screenHeight <= 0 {
		screenHeight = defaultWindowHeight
	}
	userAgent := buildBrowserUserAgent()
	if value := strings.TrimSpace(browserConfiguredUserAgent); value != "" {
		userAgent = value
	}
	acceptLanguage := detectBrowserAcceptLanguage()
	if value := strings.TrimSpace(browserConfiguredAcceptLanguage); value != "" {
		acceptLanguage = value
	}
	return browserRequestProfile{
		userAgent:      userAgent,
		acceptLanguage: acceptLanguage,
		platform:       "KolibriOS",
		secCHUA:        buildBrowserSecCHUA(),
		screenWidth:    screenWidth,
		screenHeight:   screenHeight,
	}
}

func buildBrowserUserAgent() string {
	platformComment := "X11; KolibriOS; i686"
	version := kos.KernelVersion()
	if version.Major != 0 || version.Minor != 0 || version.Patch != 0 || version.Build != 0 {
		platformComment = fmt.Sprintf(
			"X11; KolibriOS %d.%d.%d; i686",
			version.Major,
			version.Minor,
			version.Patch,
		)
	}
	return "Mozilla/5.0 (" + platformComment + ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 TagixBrowser/0.1 Safari/537.36"
}

func buildBrowserSecCHUA() string {
	return "\"Not/A)Brand\";v=\"8\", \"Chromium\";v=\"130\", \"TagixBrowser\";v=\"0\""
}

func detectBrowserAcceptLanguage() string {
	language := kos.SystemLanguage()
	if !browserLanguageKnown(language) {
		language = kos.KeyboardLayoutLanguage()
	}
	switch language {
	case kos.KeyboardLanguageEnglish:
		return "en-US,en;q=0.9"
	case kos.KeyboardLanguageFinnish:
		return "fi-FI,fi;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageGerman:
		return "de-DE,de;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageRussian:
		return "ru-RU,ru;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageFrench:
		return "fr-FR,fr;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageEstonian:
		return "et-EE,et;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageUkrainian:
		return "uk-UA,uk;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageItalian:
		return "it-IT,it;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageBelarusian:
		return "be-BY,be;q=0.9,ru;q=0.7,en;q=0.5"
	case kos.KeyboardLanguageSpanish:
		return "es-ES,es;q=0.9,en;q=0.6"
	case kos.KeyboardLanguageCatalan:
		return "ca-ES,ca;q=0.9,es;q=0.7,en;q=0.5"
	default:
		return "en-US,en;q=0.9"
	}
}

func browserLanguageKnown(language kos.KeyboardLanguage) bool {
	switch language {
	case kos.KeyboardLanguageEnglish,
		kos.KeyboardLanguageFinnish,
		kos.KeyboardLanguageGerman,
		kos.KeyboardLanguageRussian,
		kos.KeyboardLanguageFrench,
		kos.KeyboardLanguageEstonian,
		kos.KeyboardLanguageUkrainian,
		kos.KeyboardLanguageItalian,
		kos.KeyboardLanguageBelarusian,
		kos.KeyboardLanguageSpanish,
		kos.KeyboardLanguageCatalan:
		return true
	default:
		return false
	}
}

func (app *App) applyNavigationRequestHeaders(request *nethttp.Request, referrer string) {
	app.applyStandardBrowserHeaders(request, "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8", referrer)
	if request == nil {
		return
	}
	request.Header.Set("Upgrade-Insecure-Requests", "1")
	request.Header.Set("Sec-Fetch-Dest", "document")
	request.Header.Set("Sec-Fetch-Mode", "navigate")
	request.Header.Set("Sec-Fetch-Site", browserFetchSite(request.URL, referrer))
	request.Header.Set("Sec-Fetch-User", "?1")
}

func (app *App) applyFormRequestHeaders(request *nethttp.Request, referrer string) {
	app.applyNavigationRequestHeaders(request, referrer)
	if request == nil {
		return
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin := browserOriginHeaderValue(referrer); origin != "" {
		request.Header.Set("Origin", origin)
	}
}

func (app *App) applyStylesheetRequestHeaders(request *nethttp.Request, referrer string) {
	app.applyStandardBrowserHeaders(request, "text/css,*/*;q=0.1", referrer)
	if request == nil {
		return
	}
	request.Header.Set("Sec-Fetch-Dest", "style")
	request.Header.Set("Sec-Fetch-Mode", "no-cors")
	request.Header.Set("Sec-Fetch-Site", browserFetchSite(request.URL, referrer))
}

func (app *App) applyImageRequestHeaders(request *nethttp.Request, referrer string) {
	app.applyStandardBrowserHeaders(request, "image/webp,image/png,image/jpeg,image/gif,image/*;q=0.8,*/*;q=0.5", referrer)
	if request == nil {
		return
	}
	request.Header.Set("Sec-Fetch-Dest", "image")
	request.Header.Set("Sec-Fetch-Mode", "no-cors")
	request.Header.Set("Sec-Fetch-Site", browserFetchSite(request.URL, referrer))
	if viewportWidth := browserViewportWidthHint(app); viewportWidth > 0 {
		request.Header.Set("Width", strconv.Itoa(viewportWidth))
	}
}

func (app *App) applyFontRequestHeaders(request *nethttp.Request, referrer string) {
	app.applyStandardBrowserHeaders(request, "font/ttf,font/otf,font/sfnt,application/font-sfnt,application/octet-stream;q=0.8,*/*;q=0.5", referrer)
	if request == nil {
		return
	}
	request.Header.Set("Sec-Fetch-Dest", "font")
	request.Header.Set("Sec-Fetch-Mode", "no-cors")
	request.Header.Set("Sec-Fetch-Site", browserFetchSite(request.URL, referrer))
}

func (app *App) applyStandardBrowserHeaders(request *nethttp.Request, accept string, referrer string) {
	if request == nil {
		return
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("Accept-Encoding", "gzip, deflate")
	profile := browserRequestProfile{}
	if app != nil {
		profile = app.browserProfile
	}
	if strings.TrimSpace(profile.userAgent) == "" {
		profile.userAgent = buildBrowserUserAgent()
	}
	request.Header.Set("User-Agent", profile.userAgent)
	if strings.TrimSpace(profile.acceptLanguage) != "" {
		request.Header.Set("Accept-Language", profile.acceptLanguage)
	}
	if strings.TrimSpace(profile.secCHUA) != "" {
		request.Header.Set("Sec-CH-UA", profile.secCHUA)
	}
	platform := strings.TrimSpace(profile.platform)
	if platform == "" {
		platform = "KolibriOS"
	}
	request.Header.Set("Sec-CH-UA-Mobile", "?0")
	request.Header.Set("Sec-CH-UA-Platform", "\""+platform+"\"")
	request.Header.Set("DPR", "1")
	if viewportWidth := browserViewportWidthHint(app); viewportWidth > 0 {
		request.Header.Set("Viewport-Width", strconv.Itoa(viewportWidth))
	}
	if refererValue := browserRefererHeaderValue(referrer); refererValue != "" {
		request.Header.Set("Referer", refererValue)
	}
}

func browserViewportWidthHint(app *App) int {
	if app != nil {
		if width, _ := app.pageViewportSize(); width > 0 {
			return width
		}
		if app.browserProfile.screenWidth > 0 {
			return app.browserProfile.screenWidth
		}
	}
	return defaultWindowWidth
}

func browserRefererHeaderValue(raw string) string {
	parsed, ok := parseHTTPHeaderURL(raw)
	if !ok {
		return ""
	}
	parsed.Fragment = ""
	return parsed.String()
}

func browserOriginHeaderValue(raw string) string {
	parsed, ok := parseHTTPHeaderURL(raw)
	if !ok {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func browserFetchSite(target *neturl.URL, referrer string) string {
	if target == nil {
		return "none"
	}
	referrerURL, ok := parseHTTPHeaderURL(referrer)
	if !ok {
		return "none"
	}
	if sameOriginURL(target, referrerURL) {
		return "same-origin"
	}
	if sameSiteURL(target, referrerURL) {
		return "same-site"
	}
	return "cross-site"
}

func parseHTTPHeaderURL(raw string) (*neturl.URL, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	parsed, err := neturl.Parse(raw)
	if err != nil || parsed == nil {
		return nil, false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, false
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, false
	}
	return parsed, true
}

func sameOriginURL(left *neturl.URL, right *neturl.URL) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func sameSiteURL(left *neturl.URL, right *neturl.URL) bool {
	if left == nil || right == nil {
		return false
	}
	if !strings.EqualFold(left.Scheme, right.Scheme) {
		return false
	}
	leftSite := effectiveSiteName(left)
	rightSite := effectiveSiteName(right)
	return leftSite != "" && rightSite != "" && strings.EqualFold(leftSite, rightSite)
}

func effectiveSiteName(value *neturl.URL) string {
	if value == nil {
		return ""
	}
	host := sourceURLHostname(value)
	if host == "" {
		return ""
	}
	site, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err == nil && strings.TrimSpace(site) != "" {
		return strings.ToLower(site)
	}
	return strings.ToLower(host)
}

func readDecodedHTTPResponseBody(response *nethttp.Response, maxBytes int64) ([]byte, error) {
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("response body unavailable")
	}
	reader, closer, err := decodedHTTPResponseReader(response)
	if err != nil {
		return nil, err
	}
	if closer != nil {
		defer closer.Close()
	}
	return io.ReadAll(io.LimitReader(reader, maxBytes+1))
}

func decodedHTTPResponseReader(response *nethttp.Response) (io.Reader, io.Closer, error) {
	if response == nil || response.Body == nil {
		return nil, nil, fmt.Errorf("response body unavailable")
	}
	encoding := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Encoding")))
	switch encoding {
	case "", "identity":
		return response.Body, nil, nil
	case "gzip":
		reader, err := gzip.NewReader(response.Body)
		if err != nil {
			return nil, nil, err
		}
		return reader, reader, nil
	case "deflate":
		reader, err := zlib.NewReader(response.Body)
		if err != nil {
			return nil, nil, err
		}
		return reader, reader, nil
	default:
		return nil, nil, fmt.Errorf("unsupported content-encoding: %s", encoding)
	}
}
