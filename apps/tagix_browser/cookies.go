package main

import (
	nethttp "net/http"
	netcookiejar "net/http/cookiejar"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

const cookieStoreFileName = "cookies.tsv"

type persistentCookieJar struct {
	base    *netcookiejar.Jar
	path    string
	records map[string]persistedCookie
}

type persistedCookie struct {
	SourceURL   string
	Name        string
	Value       string
	Path        string
	Domain      string
	ExpiresUnix int64
	MaxAge      int
	Secure      bool
	HttpOnly    bool
	HostOnly    bool
	SameSite    int
}

func newPersistentCookieJar(cacheDir string) (*persistentCookieJar, error) {
	base, err := netcookiejar.New(&netcookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	jar := &persistentCookieJar{
		base:    base,
		path:    cookieStorePath(cacheDir),
		records: map[string]persistedCookie{},
	}
	_ = jar.load()
	return jar, nil
}

func cookieStorePath(cacheDir string) string {
	cacheDir = strings.TrimSpace(cacheDir)
	if cacheDir == "" {
		return ""
	}
	return filepath.Join(cacheDir, cookieStoreFileName)
}

func (jar *persistentCookieJar) Cookies(u *neturl.URL) []*nethttp.Cookie {
	if jar == nil || jar.base == nil || u == nil {
		return nil
	}
	return jar.base.Cookies(u)
}

func (jar *persistentCookieJar) SetCookies(u *neturl.URL, cookies []*nethttp.Cookie) {
	if jar == nil || jar.base == nil || u == nil {
		return
	}
	jar.base.SetCookies(u, cookies)
	now := time.Now()
	changed := false
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		key := persistentCookieKey(u, cookie)
		if key == "" {
			continue
		}
		if persistentCookieDeleted(cookie, now) {
			if _, ok := jar.records[key]; ok {
				delete(jar.records, key)
				changed = true
			}
			continue
		}
		record := newPersistedCookie(u, cookie)
		if existing, ok := jar.records[key]; ok && existing == record {
			continue
		}
		jar.records[key] = record
		changed = true
	}
	if changed {
		_ = jar.Save()
	}
}

func (jar *persistentCookieJar) Save() error {
	if jar == nil {
		return nil
	}
	path := strings.TrimSpace(jar.path)
	if path == "" {
		return nil
	}
	now := time.Now()
	lines := make([]string, 0, len(jar.records))
	for key, record := range jar.records {
		if record.expired(now) {
			delete(jar.records, key)
			continue
		}
		lines = append(lines, record.serialize())
	}
	if len(lines) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	sort.Strings(lines)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func (jar *persistentCookieJar) load() error {
	if jar == nil {
		return nil
	}
	path := strings.TrimSpace(jar.path)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	now := time.Now()
	dirty := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		record, ok := parsePersistedCookie(line)
		if !ok || record.SourceURL == "" || record.Name == "" {
			dirty = true
			continue
		}
		if record.expired(now) {
			dirty = true
			continue
		}
		sourceURL, err := neturl.Parse(record.SourceURL)
		if err != nil || sourceURL == nil {
			dirty = true
			continue
		}
		cookie := record.cookie()
		if cookie == nil {
			dirty = true
			continue
		}
		jar.base.SetCookies(sourceURL, []*nethttp.Cookie{cookie})
		jar.records[persistentCookieKey(sourceURL, cookie)] = record
	}
	if dirty {
		return jar.Save()
	}
	return nil
}

func newPersistedCookie(sourceURL *neturl.URL, cookie *nethttp.Cookie) persistedCookie {
	record := persistedCookie{
		SourceURL: cookieSourceURL(sourceURL),
		Name:      cookie.Name,
		Value:     cookie.Value,
		Path:      cookie.Path,
		Domain:    cookie.Domain,
		MaxAge:    cookie.MaxAge,
		Secure:    cookie.Secure,
		HttpOnly:  cookie.HttpOnly,
		HostOnly:  strings.TrimSpace(cookie.Domain) == "",
		SameSite:  int(cookie.SameSite),
	}
	if !cookie.Expires.IsZero() {
		record.ExpiresUnix = cookie.Expires.Unix()
	}
	return record
}

func persistentCookieDeleted(cookie *nethttp.Cookie, now time.Time) bool {
	if cookie == nil {
		return false
	}
	if cookie.MaxAge < 0 {
		return true
	}
	if !cookie.Expires.IsZero() && cookie.Expires.Before(now) {
		return true
	}
	return false
}

func persistentCookieKey(sourceURL *neturl.URL, cookie *nethttp.Cookie) string {
	if sourceURL == nil || cookie == nil || strings.TrimSpace(cookie.Name) == "" {
		return ""
	}
	domain := strings.ToLower(strings.TrimSpace(cookie.Domain))
	hostOnly := domain == ""
	domain = strings.TrimPrefix(domain, ".")
	if domain == "" {
		domain = sourceURLHostname(sourceURL)
	}
	if domain == "" {
		return ""
	}
	path := strings.TrimSpace(cookie.Path)
	if path == "" {
		path = defaultCookiePath(sourceURL)
	}
	if path == "" {
		path = "/"
	}
	hostOnlyField := "0"
	if hostOnly {
		hostOnlyField = "1"
	}
	return domain + "\t" + path + "\t" + cookie.Name + "\t" + hostOnlyField
}

func defaultCookiePath(sourceURL *neturl.URL) string {
	if sourceURL == nil {
		return "/"
	}
	path := strings.TrimSpace(sourceURL.Path)
	if path == "" || path[0] != '/' {
		return "/"
	}
	if path == "/" {
		return "/"
	}
	slash := lastIndexByte(path, '/')
	if slash <= 0 {
		return "/"
	}
	return path[:slash]
}

func sourceURLHostname(sourceURL *neturl.URL) string {
	if sourceURL == nil {
		return ""
	}
	host := strings.TrimSpace(sourceURL.Host)
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		if end := strings.IndexByte(host, ']'); end >= 0 {
			return strings.ToLower(host[:end+1])
		}
	}
	if strings.Count(host, ":") == 1 {
		if colon := lastIndexByte(host, ':'); colon >= 0 {
			return strings.ToLower(host[:colon])
		}
	}
	return strings.ToLower(host)
}

func cookieSourceURL(sourceURL *neturl.URL) string {
	if sourceURL == nil {
		return ""
	}
	value := *sourceURL
	value.RawQuery = ""
	value.Fragment = ""
	if strings.TrimSpace(value.Path) == "" {
		value.Path = "/"
	}
	return value.String()
}

func (record persistedCookie) expired(now time.Time) bool {
	if record.MaxAge < 0 {
		return true
	}
	if record.ExpiresUnix != 0 && time.Unix(record.ExpiresUnix, 0).Before(now) {
		return true
	}
	return false
}

func (record persistedCookie) cookie() *nethttp.Cookie {
	if strings.TrimSpace(record.Name) == "" {
		return nil
	}
	cookie := &nethttp.Cookie{
		Name:     record.Name,
		Value:    record.Value,
		Path:     record.Path,
		MaxAge:   record.MaxAge,
		Secure:   record.Secure,
		HttpOnly: record.HttpOnly,
		SameSite: nethttp.SameSite(record.SameSite),
	}
	if !record.HostOnly {
		cookie.Domain = record.Domain
	}
	if record.ExpiresUnix != 0 {
		cookie.Expires = time.Unix(record.ExpiresUnix, 0)
	}
	return cookie
}

func (record persistedCookie) serialize() string {
	fields := []string{
		cookieFieldEscape(record.SourceURL),
		cookieFieldEscape(record.Name),
		cookieFieldEscape(record.Value),
		cookieFieldEscape(record.Path),
		cookieFieldEscape(record.Domain),
		strconv.FormatInt(record.ExpiresUnix, 10),
		strconv.Itoa(record.MaxAge),
		cookieBoolString(record.Secure),
		cookieBoolString(record.HttpOnly),
		cookieBoolString(record.HostOnly),
		strconv.Itoa(record.SameSite),
	}
	return strings.Join(fields, "\t")
}

func parsePersistedCookie(line string) (persistedCookie, bool) {
	record := persistedCookie{}
	fields := strings.Split(line, "\t")
	if len(fields) != 11 {
		return record, false
	}
	var ok bool
	record.SourceURL, ok = cookieFieldUnescape(fields[0])
	if !ok {
		return record, false
	}
	record.Name, ok = cookieFieldUnescape(fields[1])
	if !ok {
		return record, false
	}
	record.Value, ok = cookieFieldUnescape(fields[2])
	if !ok {
		return record, false
	}
	record.Path, ok = cookieFieldUnescape(fields[3])
	if !ok {
		return record, false
	}
	record.Domain, ok = cookieFieldUnescape(fields[4])
	if !ok {
		return record, false
	}
	if fields[5] != "" {
		value, err := strconv.ParseInt(fields[5], 10, 64)
		if err != nil {
			return record, false
		}
		record.ExpiresUnix = value
	}
	value, err := strconv.Atoi(fields[6])
	if err != nil {
		return record, false
	}
	record.MaxAge = value
	record.Secure = cookieBoolValue(fields[7])
	record.HttpOnly = cookieBoolValue(fields[8])
	record.HostOnly = cookieBoolValue(fields[9])
	value, err = strconv.Atoi(fields[10])
	if err != nil {
		return record, false
	}
	record.SameSite = value
	return record, true
}

func cookieFieldEscape(value string) string {
	return neturl.QueryEscape(value)
}

func cookieFieldUnescape(value string) (string, bool) {
	decoded, err := neturl.QueryUnescape(value)
	if err != nil {
		return "", false
	}
	return decoded, true
}

func cookieBoolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func cookieBoolValue(value string) bool {
	return strings.TrimSpace(value) == "1"
}
