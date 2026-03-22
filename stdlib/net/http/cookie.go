package http

import (
	"strconv"
	"strings"
	"time"
)

type Cookie struct {
	Name  string
	Value string

	Path       string
	Domain     string
	Expires    time.Time
	RawExpires string
	MaxAge     int
	Secure     bool
	HttpOnly   bool
	SameSite   SameSite
	Raw        string
	Unparsed   []string
}

type SameSite int

const (
	SameSiteDefaultMode SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

func (cookie *Cookie) String() string {
	if cookie == nil || cookie.Name == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(cookie.Name)
	builder.WriteString("=")
	builder.WriteString(cookie.Value)
	if cookie.Path != "" {
		builder.WriteString("; Path=")
		builder.WriteString(cookie.Path)
	}
	if cookie.Domain != "" {
		builder.WriteString("; Domain=")
		builder.WriteString(cookie.Domain)
	}
	if !cookie.Expires.IsZero() {
		builder.WriteString("; Expires=")
		builder.WriteString(cookie.Expires.UTC().Format(time.RFC1123))
	}
	if cookie.MaxAge > 0 {
		builder.WriteString("; Max-Age=")
		builder.WriteString(strconv.Itoa(cookie.MaxAge))
	} else if cookie.MaxAge < 0 {
		builder.WriteString("; Max-Age=0")
	}
	if cookie.HttpOnly {
		builder.WriteString("; HttpOnly")
	}
	if cookie.Secure {
		builder.WriteString("; Secure")
	}
	switch cookie.SameSite {
	case SameSiteLaxMode:
		builder.WriteString("; SameSite=Lax")
	case SameSiteStrictMode:
		builder.WriteString("; SameSite=Strict")
	case SameSiteNoneMode:
		builder.WriteString("; SameSite=None")
	}
	return builder.String()
}

func readSetCookies(header Header) []*Cookie {
	lines := header.Values("Set-Cookie")
	if len(lines) == 0 {
		return nil
	}
	cookies := make([]*Cookie, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		nameValue := strings.TrimSpace(parts[0])
		name, value, ok := strings.Cut(nameValue, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		value = cookieTrimValue(value)
		cookie := &Cookie{
			Name:  name,
			Value: value,
			Raw:   line,
		}
		for j := 1; j < len(parts); j++ {
			rawAttr := strings.TrimSpace(parts[j])
			if rawAttr == "" {
				continue
			}
			attr, attrValue, hasValue := strings.Cut(rawAttr, "=")
			attr = strings.ToLower(strings.TrimSpace(attr))
			attrValue = cookieTrimValue(attrValue)
			switch attr {
			case "path":
				if hasValue {
					cookie.Path = attrValue
				}
			case "domain":
				if hasValue {
					cookie.Domain = strings.ToLower(attrValue)
				}
			case "expires":
				if hasValue {
					cookie.RawExpires = attrValue
					if parsed, ok := parseCookieExpires(attrValue); ok {
						cookie.Expires = parsed
					}
				}
			case "max-age":
				if hasValue {
					if seconds, err := strconv.Atoi(attrValue); err == nil {
						if seconds <= 0 {
							cookie.MaxAge = -1
						} else {
							cookie.MaxAge = seconds
						}
					}
				}
			case "secure":
				cookie.Secure = true
			case "httponly":
				cookie.HttpOnly = true
			case "samesite":
				switch strings.ToLower(attrValue) {
				case "lax":
					cookie.SameSite = SameSiteLaxMode
				case "strict":
					cookie.SameSite = SameSiteStrictMode
				case "none":
					cookie.SameSite = SameSiteNoneMode
				default:
					cookie.SameSite = SameSiteDefaultMode
				}
			default:
				cookie.Unparsed = append(cookie.Unparsed, rawAttr)
			}
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

func cookieHeaderValue(cookies []*Cookie) string {
	if len(cookies) == 0 {
		return ""
	}
	var builder strings.Builder
	first := true
	for i := 0; i < len(cookies); i++ {
		cookie := cookies[i]
		if cookie == nil || cookie.Name == "" {
			continue
		}
		if !first {
			builder.WriteString("; ")
		}
		first = false
		builder.WriteString(cookie.Name)
		builder.WriteString("=")
		builder.WriteString(cookie.Value)
	}
	return builder.String()
}

func cookieTrimValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		value = value[1 : len(value)-1]
	}
	return value
}

func parseCookieExpires(value string) (time.Time, bool) {
	formats := []string{
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02-Jan-2006 15:04:05 MST",
	}
	for i := 0; i < len(formats); i++ {
		if parsed, err := time.Parse(formats[i], value); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}
