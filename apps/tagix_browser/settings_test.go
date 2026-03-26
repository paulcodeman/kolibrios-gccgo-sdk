package main

import "testing"

func TestParseBrowserSettingsINIOverridesDefaults(t *testing.T) {
	settings := parseBrowserSettingsINI(`
[navigation]
home_url = example.com

[assets]
shell_template = custom/shell.html
welcome_page = custom/welcome.html
forms_page = custom/forms.html
ca_bundle = certs/roots.pem
font_dir = custom/fonts

[cache]
dir = /tmp/custom-cache
roots = /tmp2/1, /tmp3/1
name = browser-cache
cookie_store = browser.tsv

[network]
user_agent = TagixTest/1.0
accept_language = ru-RU,ru;q=0.9
max_content_bytes = 123456
max_font_content_bytes = 654321
`, defaultBrowserSettings())

	if settings.homeURL != "http://example.com" {
		t.Fatalf("homeURL = %q, want http://example.com", settings.homeURL)
	}
	if settings.shellTemplate != "custom/shell.html" {
		t.Fatalf("shellTemplate = %q", settings.shellTemplate)
	}
	if settings.welcomePage != "custom/welcome.html" {
		t.Fatalf("welcomePage = %q", settings.welcomePage)
	}
	if settings.formsPage != "custom/forms.html" {
		t.Fatalf("formsPage = %q", settings.formsPage)
	}
	if settings.caBundlePath != "certs/roots.pem" {
		t.Fatalf("caBundlePath = %q", settings.caBundlePath)
	}
	if settings.fontDir != "custom/fonts" {
		t.Fatalf("fontDir = %q", settings.fontDir)
	}
	if settings.cacheDir != "/tmp/custom-cache" {
		t.Fatalf("cacheDir = %q", settings.cacheDir)
	}
	if len(settings.cacheRoots) != 2 || settings.cacheRoots[0] != "/tmp2/1" || settings.cacheRoots[1] != "/tmp3/1" {
		t.Fatalf("cacheRoots = %#v", settings.cacheRoots)
	}
	if settings.cacheName != "browser-cache" {
		t.Fatalf("cacheName = %q", settings.cacheName)
	}
	if settings.cookieStore != "browser.tsv" {
		t.Fatalf("cookieStore = %q", settings.cookieStore)
	}
	if settings.userAgent != "TagixTest/1.0" {
		t.Fatalf("userAgent = %q", settings.userAgent)
	}
	if settings.acceptLanguage != "ru-RU,ru;q=0.9" {
		t.Fatalf("acceptLanguage = %q", settings.acceptLanguage)
	}
	if settings.maxContent != 123456 {
		t.Fatalf("maxContent = %d", settings.maxContent)
	}
	if settings.maxFontContent != 654321 {
		t.Fatalf("maxFontContent = %d", settings.maxFontContent)
	}
}

func TestParseBrowserSettingsINIKeepsDefaultsForInvalidInts(t *testing.T) {
	defaults := defaultBrowserSettings()
	settings := parseBrowserSettingsINI(`
[network]
max_content_bytes = nope
max_font_content_bytes = -4
`, defaults)

	if settings.maxContent != defaults.maxContent {
		t.Fatalf("maxContent = %d, want %d", settings.maxContent, defaults.maxContent)
	}
	if settings.maxFontContent != defaults.maxFontContent {
		t.Fatalf("maxFontContent = %d, want %d", settings.maxFontContent, defaults.maxFontContent)
	}
}
