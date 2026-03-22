package http

import "net/url"

// A CookieJar manages storage and use of cookies in HTTP requests.
type CookieJar interface {
	SetCookies(u *url.URL, cookies []*Cookie)
	Cookies(u *url.URL) []*Cookie
}
