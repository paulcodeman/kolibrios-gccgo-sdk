package cookiejar

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type PublicSuffixList interface {
	PublicSuffix(domain string) string
	String() string
}

type Options struct {
	PublicSuffixList PublicSuffixList
}

type Jar struct {
	psList PublicSuffixList

	mu      sync.Mutex
	entries map[string]map[string]entry
}

type entry struct {
	Name       string
	Value      string
	Domain     string
	Path       string
	Secure     bool
	HttpOnly   bool
	Persistent bool
	HostOnly   bool
	Expires    time.Time
	Creation   time.Time
	LastAccess time.Time
}

func (entry *entry) id() string {
	return entry.Domain + ";" + entry.Path + ";" + entry.Name
}

func New(options *Options) (*Jar, error) {
	jar := &Jar{
		entries: make(map[string]map[string]entry),
	}
	if options != nil {
		jar.psList = options.PublicSuffixList
	}
	return jar, nil
}

func (jar *Jar) Cookies(target *url.URL) []*http.Cookie {
	if target == nil {
		return nil
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil
	}
	host, err := canonicalHost(target.Host)
	if err != nil {
		return nil
	}
	key := jarKey(host, jar.psList)
	path := target.Path
	if path == "" {
		path = "/"
	}
	https := target.Scheme == "https"
	now := time.Now()

	jar.mu.Lock()
	defer jar.mu.Unlock()

	submap := jar.entries[key]
	if len(submap) == 0 {
		return nil
	}
	selected := make([]entry, 0, len(submap))
	for id, candidate := range submap {
		if candidate.Persistent && !candidate.Expires.After(now) {
			delete(submap, id)
			continue
		}
		if !candidate.shouldSend(https, host, path) {
			continue
		}
		candidate.LastAccess = now
		submap[id] = candidate
		selected = append(selected, candidate)
	}
	if len(submap) == 0 {
		delete(jar.entries, key)
	}
	if len(selected) == 0 {
		return nil
	}
	sortEntries(selected)
	out := make([]*http.Cookie, 0, len(selected))
	for i := 0; i < len(selected); i++ {
		out = append(out, &http.Cookie{
			Name:  selected[i].Name,
			Value: selected[i].Value,
		})
	}
	return out
}

func (jar *Jar) SetCookies(target *url.URL, cookies []*http.Cookie) {
	if target == nil || len(cookies) == 0 {
		return
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return
	}
	host, err := canonicalHost(target.Host)
	if err != nil {
		return
	}
	key := jarKey(host, jar.psList)
	defaultPath := defaultPath(target.Path)
	now := time.Now()

	jar.mu.Lock()
	defer jar.mu.Unlock()

	submap := jar.entries[key]
	for i := 0; i < len(cookies); i++ {
		candidate, remove, err := jar.newEntry(cookies[i], now, defaultPath, host)
		if err != nil {
			continue
		}
		id := candidate.id()
		if remove {
			if submap != nil {
				delete(submap, id)
			}
			continue
		}
		if submap == nil {
			submap = make(map[string]entry)
		}
		if existing, ok := submap[id]; ok {
			candidate.Creation = existing.Creation
		} else {
			candidate.Creation = now
		}
		candidate.LastAccess = now
		submap[id] = candidate
	}
	if len(submap) == 0 {
		delete(jar.entries, key)
	} else {
		jar.entries[key] = submap
	}
}

func (entry *entry) shouldSend(https bool, host string, path string) bool {
	return entry.domainMatch(host) && entry.pathMatch(path) && (https || !entry.Secure)
}

func (entry *entry) domainMatch(host string) bool {
	if entry.Domain == host {
		return true
	}
	return !entry.HostOnly && hasDotSuffix(host, entry.Domain)
}

func (entry *entry) pathMatch(requestPath string) bool {
	if requestPath == entry.Path {
		return true
	}
	if strings.HasPrefix(requestPath, entry.Path) {
		if entry.Path[len(entry.Path)-1] == '/' {
			return true
		}
		if len(requestPath) > len(entry.Path) && requestPath[len(entry.Path)] == '/' {
			return true
		}
	}
	return false
}

func (jar *Jar) newEntry(cookie *http.Cookie, now time.Time, defaultPath string, host string) (entry entry, remove bool, err error) {
	if cookie == nil || cookie.Name == "" {
		return entry, false, errors.New("cookiejar: invalid cookie")
	}
	entry.Name = cookie.Name
	entry.Value = cookie.Value
	entry.Secure = cookie.Secure
	entry.HttpOnly = cookie.HttpOnly
	if cookie.Path == "" || cookie.Path[0] != '/' {
		entry.Path = defaultPath
	} else {
		entry.Path = cookie.Path
	}

	entry.Domain, entry.HostOnly, err = jar.domainAndType(host, cookie.Domain)
	if err != nil {
		return entry, false, err
	}

	switch {
	case cookie.MaxAge < 0:
		return entry, true, nil
	case cookie.MaxAge > 0:
		entry.Expires = now.Add(time.Duration(cookie.MaxAge) * time.Second)
		entry.Persistent = true
	case !cookie.Expires.IsZero():
		if !cookie.Expires.After(now) {
			return entry, true, nil
		}
		entry.Expires = cookie.Expires
		entry.Persistent = true
	default:
		entry.Expires = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	return entry, false, nil
}

func (jar *Jar) domainAndType(host string, domain string) (string, bool, error) {
	if domain == "" {
		return host, true, nil
	}
	if isIP(host) {
		if host != domain {
			return "", false, errors.New("cookiejar: illegal cookie domain attribute")
		}
		return host, true, nil
	}

	if domain[0] == '.' {
		domain = domain[1:]
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", false, errors.New("cookiejar: malformed cookie domain attribute")
	}

	if jar.psList != nil {
		suffix := jar.psList.PublicSuffix(domain)
		if suffix != "" && !hasDotSuffix(domain, suffix) {
			if host == domain {
				return host, true, nil
			}
			return "", false, errors.New("cookiejar: illegal cookie domain attribute")
		}
	}
	if host != domain && !hasDotSuffix(host, domain) {
		return "", false, errors.New("cookiejar: illegal cookie domain attribute")
	}
	return domain, false, nil
}

func canonicalHost(host string) (string, error) {
	if hasPort(host) {
		trimmed, _, err := net.SplitHostPort(host)
		if err != nil {
			return "", err
		}
		host = trimmed
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return "", errors.New("cookiejar: empty host")
	}
	return host, nil
}

func hasPort(host string) bool {
	colons := strings.Count(host, ":")
	if colons == 0 {
		return false
	}
	if colons == 1 {
		return true
	}
	return strings.HasPrefix(host, "[") && strings.Contains(host, "]:")
}

func jarKey(host string, publicSuffixList PublicSuffixList) string {
	if isIP(host) {
		return host
	}
	if publicSuffixList == nil {
		if index := strings.LastIndex(host, "."); index > 0 {
			if prev := strings.LastIndex(host[:index], "."); prev >= 0 {
				return host[prev+1:]
			}
		}
		return host
	}
	suffix := publicSuffixList.PublicSuffix(host)
	if suffix == "" || suffix == host {
		return host
	}
	index := len(host) - len(suffix)
	if index <= 0 || host[index-1] != '.' {
		return host
	}
	if prev := strings.LastIndex(host[:index-1], "."); prev >= 0 {
		return host[prev+1:]
	}
	return host
}

func defaultPath(path string) string {
	if path == "" || path[0] != '/' {
		return "/"
	}
	index := strings.LastIndex(path, "/")
	if index <= 0 {
		return "/"
	}
	return path[:index]
}

func isIP(host string) bool {
	if strings.ContainsAny(host, ":%") {
		return true
	}
	return net.ParseIP(host) != nil
}

func hasDotSuffix(value string, suffix string) bool {
	return len(value) > len(suffix) && value[len(value)-len(suffix)-1] == '.' && value[len(value)-len(suffix):] == suffix
}

func sortEntries(entries []entry) {
	for i := 1; i < len(entries); i++ {
		current := entries[i]
		j := i - 1
		for ; j >= 0; j-- {
			left := current
			right := entries[j]
			if len(left.Path) < len(right.Path) {
				break
			}
			if len(left.Path) == len(right.Path) && !left.Creation.Before(right.Creation) {
				break
			}
			entries[j+1] = entries[j]
		}
		entries[j+1] = current
	}
}
