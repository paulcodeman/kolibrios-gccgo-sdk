package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	settingsAssetPath          = "assets/settings.ini"
	defaultWelcomePageAsset    = "assets/pages/welcome.html"
	defaultFormsPageAsset      = "assets/pages/forms.html"
	defaultShellTemplateAsset  = "assets/shell.html"
	defaultLocalCABundleAsset  = "assets/ca-bundle.pem"
	defaultBundledFontDir      = "assets/fonts"
	defaultResourceCacheName   = "tagix_browser.cache"
	defaultCookieStoreFile     = "cookies.tsv"
	defaultMaxContentBytes     = 512 * 1024
	defaultMaxFontContentBytes = 4 * 1024 * 1024
)

var (
	browserHomeURL                  = defaultURL
	welcomePageAsset                = defaultWelcomePageAsset
	formsPageAsset                  = defaultFormsPageAsset
	localCABundleAsset              = defaultLocalCABundleAsset
	bundledFontDir                  = defaultBundledFontDir
	webSansFontPath                 = defaultBundledFontDir + "/Go.ttf"
	webSansBoldFontPath             = defaultBundledFontDir + "/GoBold.ttf"
	webSansItalicFontPath           = defaultBundledFontDir + "/GoItalic.ttf"
	webSansBoldItalicFontPath       = defaultBundledFontDir + "/GoBoldItalic.ttf"
	webMonoFontPath                 = defaultBundledFontDir + "/GoMono.ttf"
	webMonoBoldFontPath             = defaultBundledFontDir + "/GoMonoBold.ttf"
	webMonoItalicFontPath           = defaultBundledFontDir + "/GoMonoItalic.ttf"
	webMonoBoldItalicFontPath       = defaultBundledFontDir + "/GoMonoBoldItalic.ttf"
	webIconFontPath                 = defaultBundledFontDir + "/MaterialDesignIconsDesktop.ttf"
	webShellHTML                    = defaultShellTemplateAsset
	resourceCacheDirOverride        string
	resourceCacheRootCandidates     = []string{"/tmp0/1", "/tmp1/1"}
	resourceCacheDirName            = defaultResourceCacheName
	cookieStoreFileName             = defaultCookieStoreFile
	browserConfiguredUserAgent      string
	browserConfiguredAcceptLanguage string
	maxContent                      = defaultMaxContentBytes
	maxFontContent                  = defaultMaxFontContentBytes
	loadedBrowserSettings           = defaultBrowserSettings()
)

type browserSettings struct {
	homeURL        string
	shellTemplate  string
	welcomePage    string
	formsPage      string
	caBundlePath   string
	fontDir        string
	cacheDir       string
	cacheRoots     []string
	cacheName      string
	cookieStore    string
	userAgent      string
	acceptLanguage string
	maxContent     int
	maxFontContent int
}

func init() {
	loadedBrowserSettings = loadBrowserSettings()
	applyBrowserSettings(loadedBrowserSettings)
}

func defaultBrowserSettings() browserSettings {
	return browserSettings{
		homeURL:        defaultURL,
		shellTemplate:  defaultShellTemplateAsset,
		welcomePage:    defaultWelcomePageAsset,
		formsPage:      defaultFormsPageAsset,
		caBundlePath:   defaultLocalCABundleAsset,
		fontDir:        defaultBundledFontDir,
		cacheRoots:     []string{"/tmp0/1", "/tmp1/1"},
		cacheName:      defaultResourceCacheName,
		cookieStore:    defaultCookieStoreFile,
		maxContent:     defaultMaxContentBytes,
		maxFontContent: defaultMaxFontContentBytes,
	}
}

func loadBrowserSettings() browserSettings {
	settings := defaultBrowserSettings()
	data, err := os.ReadFile(settingsAssetPath)
	if err != nil || len(data) == 0 {
		return settings
	}
	return parseBrowserSettingsINI(string(data), settings)
}

func parseBrowserSettingsINI(source string, settings browserSettings) browserSettings {
	section := ""
	for _, rawLine := range strings.Split(source, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(rawLine, "\r"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		split := strings.IndexByte(line, '=')
		if split <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:split]))
		value := strings.TrimSpace(line[split+1:])
		switch section {
		case "navigation":
			switch key {
			case "home_url":
				if normalized := strings.TrimSpace(normalizeURL(value)); normalized != "" {
					settings.homeURL = normalized
				}
			}
		case "assets":
			switch key {
			case "shell_template":
				if value != "" {
					settings.shellTemplate = value
				}
			case "welcome_page":
				if value != "" {
					settings.welcomePage = value
				}
			case "forms_page":
				if value != "" {
					settings.formsPage = value
				}
			case "ca_bundle":
				if value != "" {
					settings.caBundlePath = value
				}
			case "font_dir":
				if value != "" {
					settings.fontDir = value
				}
			}
		case "cache":
			switch key {
			case "dir":
				settings.cacheDir = value
			case "roots":
				if roots := splitSettingsList(value); len(roots) > 0 {
					settings.cacheRoots = roots
				}
			case "name":
				if value != "" {
					settings.cacheName = value
				}
			case "cookie_store":
				if value != "" {
					settings.cookieStore = value
				}
			}
		case "network":
			switch key {
			case "user_agent":
				settings.userAgent = value
			case "accept_language":
				settings.acceptLanguage = value
			case "max_content_bytes":
				if parsed, ok := parsePositiveSettingsInt(value); ok {
					settings.maxContent = parsed
				}
			case "max_font_content_bytes":
				if parsed, ok := parsePositiveSettingsInt(value); ok {
					settings.maxFontContent = parsed
				}
			}
		}
	}
	return settings
}

func splitSettingsList(value string) []string {
	parts := strings.Split(value, ",")
	list := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		list = append(list, part)
	}
	if len(list) == 0 {
		return nil
	}
	return list
}

func parsePositiveSettingsInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func applyBrowserSettings(settings browserSettings) {
	browserHomeURL = defaultURL
	if value := strings.TrimSpace(settings.homeURL); value != "" {
		browserHomeURL = value
	}
	welcomePageAsset = defaultWelcomePageAsset
	if value := strings.TrimSpace(settings.welcomePage); value != "" {
		welcomePageAsset = value
	}
	formsPageAsset = defaultFormsPageAsset
	if value := strings.TrimSpace(settings.formsPage); value != "" {
		formsPageAsset = value
	}
	localCABundleAsset = defaultLocalCABundleAsset
	if value := strings.TrimSpace(settings.caBundlePath); value != "" {
		localCABundleAsset = value
	}
	webShellHTML = defaultShellTemplateAsset
	if value := strings.TrimSpace(settings.shellTemplate); value != "" {
		webShellHTML = value
	}
	bundledFontDir = defaultBundledFontDir
	if value := strings.TrimSpace(settings.fontDir); value != "" {
		bundledFontDir = value
	}
	configureBundledFontPaths(bundledFontDir)

	resourceCacheDirOverride = strings.TrimSpace(settings.cacheDir)
	resourceCacheDirName = defaultResourceCacheName
	if value := strings.TrimSpace(settings.cacheName); value != "" {
		resourceCacheDirName = value
	}
	resourceCacheRootCandidates = append([]string{}, defaultBrowserSettings().cacheRoots...)
	if len(settings.cacheRoots) > 0 {
		resourceCacheRootCandidates = append([]string{}, settings.cacheRoots...)
	}
	cookieStoreFileName = defaultCookieStoreFile
	if value := strings.TrimSpace(settings.cookieStore); value != "" {
		cookieStoreFileName = value
	}

	browserConfiguredUserAgent = strings.TrimSpace(settings.userAgent)
	browserConfiguredAcceptLanguage = strings.TrimSpace(settings.acceptLanguage)

	maxContent = defaultMaxContentBytes
	if settings.maxContent > 0 {
		maxContent = settings.maxContent
	}
	maxFontContent = defaultMaxFontContentBytes
	if settings.maxFontContent > 0 {
		maxFontContent = settings.maxFontContent
	}
}

func configureBundledFontPaths(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = defaultBundledFontDir
	}
	webSansFontPath = filepath.Join(dir, "Go.ttf")
	webSansBoldFontPath = filepath.Join(dir, "GoBold.ttf")
	webSansItalicFontPath = filepath.Join(dir, "GoItalic.ttf")
	webSansBoldItalicFontPath = filepath.Join(dir, "GoBoldItalic.ttf")
	webMonoFontPath = filepath.Join(dir, "GoMono.ttf")
	webMonoBoldFontPath = filepath.Join(dir, "GoMonoBold.ttf")
	webMonoItalicFontPath = filepath.Join(dir, "GoMonoItalic.ttf")
	webMonoBoldItalicFontPath = filepath.Join(dir, "GoMonoBoldItalic.ttf")
	webIconFontPath = filepath.Join(dir, "MaterialDesignIconsDesktop.ttf")
}
