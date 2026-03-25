package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const resourceCacheDirName = "tagix_browser.cache"

func initResourceCacheDir() string {
	for _, root := range []string{"/tmp0/1", "/tmp1/1"} {
		info, err := os.Stat(root)
		if err != nil || info == nil || !info.IsDir() {
			continue
		}
		dir := filepath.Join(root, resourceCacheDirName)
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir
		}
	}
	return ""
}

func (app *App) resourceCachePath(prefix string, rawURL string, suffix string) string {
	if app == nil {
		return ""
	}
	base := strings.TrimSpace(app.resourceCacheDir)
	if base == "" || strings.TrimSpace(rawURL) == "" {
		return ""
	}
	sum := sha1.Sum([]byte(rawURL))
	name := fmt.Sprintf("%s-%x%s", prefix, sum, suffix)
	return filepath.Join(base, name)
}

func readCachedResource(path string) ([]byte, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func writeCachedResource(path string, data []byte) bool {
	path = strings.TrimSpace(path)
	if path == "" || len(data) == 0 {
		return false
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false
	}
	return true
}
