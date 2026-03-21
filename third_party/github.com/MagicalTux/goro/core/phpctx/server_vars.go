package phpctx

import (
	"net"
	"strconv"
	"strings"

	"github.com/MagicalTux/goro/core/phpv"
)

func (g *Global) SetServerValue(name, value string) error {
	return g.setSuperglobalString("_SERVER", name, value)
}

func (g *Global) setSuperglobalString(globalName, key, value string) error {
	arrayValue := g.h.GetString(phpv.ZString(globalName))
	if arrayValue == nil {
		return nil
	}

	arrayValue, err := arrayValue.As(g, phpv.ZtArray)
	if err != nil {
		return err
	}

	return arrayValue.Value().(*phpv.ZArray).OffsetSet(g, phpv.ZString(key), phpv.ZString(value).ZVal())
}

func (g *Global) parseCookies(target *phpv.ZArray) error {
	if g.req == nil {
		return nil
	}

	values := g.req.Header.Values("Cookie")
	for valueIndex := 0; valueIndex < len(values); valueIndex++ {
		parts := strings.Split(values[valueIndex], ";")
		for partIndex := 0; partIndex < len(parts); partIndex++ {
			part := strings.TrimSpace(parts[partIndex])
			if part == "" {
				continue
			}

			name := part
			value := ""
			if separator := strings.IndexByte(part, '='); separator >= 0 {
				name = part[:separator]
				value = part[separator+1:]
			}

			if err := target.OffsetSet(g, phpv.ZString(name), phpv.ZString(value).ZVal()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Global) populateServerRequestData(target *phpv.ZArray) {
	if g.req == nil || target == nil {
		return
	}

	g.setServerArrayValue(target, "GATEWAY_INTERFACE", "CGI/1.1")
	g.setServerArrayValue(target, "REQUEST_METHOD", g.req.Method)
	g.setServerArrayValue(target, "REQUEST_URI", requestURIValue(g))
	g.setServerArrayValue(target, "REQUEST_SCHEME", "http")
	g.setServerArrayValue(target, "QUERY_STRING", requestQueryValue(g))
	g.setServerArrayValue(target, "SERVER_SOFTWARE", "goro")
	g.setServerArrayValue(target, "SERVER_PROTOCOL", requestProtoValue(g))

	host, port := splitHostPortValue(firstNonEmpty(g.req.Host, g.req.Header.Get("Host")))
	if host != "" {
		g.setServerArrayValue(target, "HTTP_HOST", firstNonEmpty(g.req.Host, g.req.Header.Get("Host")))
		g.setServerArrayValue(target, "SERVER_NAME", host)
	}
	if port != "" {
		g.setServerArrayValue(target, "SERVER_PORT", port)
	}

	remoteHost, remotePort := splitHostPortValue(g.req.RemoteAddr)
	if remoteHost != "" {
		g.setServerArrayValue(target, "REMOTE_ADDR", remoteHost)
	}
	if remotePort != "" {
		g.setServerArrayValue(target, "REMOTE_PORT", remotePort)
	}

	localHost, localPort := splitHostPortValue(g.req.LocalAddr)
	if localHost != "" {
		g.setServerArrayValue(target, "SERVER_ADDR", localHost)
		if host == "" {
			g.setServerArrayValue(target, "SERVER_NAME", localHost)
		}
	}
	if localPort != "" && port == "" {
		g.setServerArrayValue(target, "SERVER_PORT", localPort)
	}

	if contentType := g.req.Header.Get("Content-Type"); contentType != "" {
		g.setServerArrayValue(target, "CONTENT_TYPE", contentType)
	}
	if g.req.ContentLength > 0 {
		g.setServerArrayValue(target, "CONTENT_LENGTH", strconv.FormatInt(g.req.ContentLength, 10))
	} else if contentLength := g.req.Header.Get("Content-Length"); contentLength != "" {
		g.setServerArrayValue(target, "CONTENT_LENGTH", contentLength)
	}

	for key, values := range g.req.Header {
		if len(values) == 0 {
			continue
		}

		envKey := headerEnvKey(key)
		if envKey == "" {
			continue
		}
		if envKey == "CONTENT_TYPE" || envKey == "CONTENT_LENGTH" {
			continue
		}
		g.setServerArrayValue(target, "HTTP_"+envKey, strings.Join(values, ", "))
	}
}

func (g *Global) setServerArrayValue(target *phpv.ZArray, key string, value string) {
	if target == nil || key == "" {
		return
	}
	_ = target.OffsetSet(g, phpv.ZString(key), phpv.ZString(value).ZVal())
}

func splitHostPortValue(value string) (host string, port string) {
	if value == "" {
		return "", ""
	}
	host, port, err := net.SplitHostPort(value)
	if err == nil {
		return host, port
	}

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return strings.TrimPrefix(strings.TrimSuffix(value, "]"), "["), ""
	}
	return value, ""
}

func headerEnvKey(key string) string {
	if key == "" {
		return ""
	}

	buffer := make([]byte, 0, len(key))
	for index := 0; index < len(key); index++ {
		ch := key[index]
		switch {
		case ch >= 'a' && ch <= 'z':
			buffer = append(buffer, ch-('a'-'A'))
		case (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9'):
			buffer = append(buffer, ch)
		default:
			buffer = append(buffer, '_')
		}
	}
	return string(buffer)
}

func firstNonEmpty(values ...string) string {
	for index := 0; index < len(values); index++ {
		if values[index] != "" {
			return values[index]
		}
	}
	return ""
}

type requestFieldView struct {
	requestURI string
	rawQuery   string
	proto      string
}

func (view requestFieldView) URLField() string   { return view.requestURI }
func (view requestFieldView) QueryField() string { return view.rawQuery }
func (view requestFieldView) ProtoField() string { return view.proto }

func requestView(req *Global) requestFieldView {
	view := requestFieldView{}
	if req == nil || req.req == nil {
		return view
	}

	if req.req.RequestURI != "" {
		view.requestURI = req.req.RequestURI
	} else if req.req.URL != nil {
		view.requestURI = req.req.URL.Path
		if view.requestURI == "" {
			view.requestURI = "/"
		}
		if req.req.URL.RawQuery != "" {
			view.requestURI += "?" + req.req.URL.RawQuery
		}
	}
	if view.requestURI == "" {
		view.requestURI = "/"
	}

	if req.req.URL != nil {
		view.rawQuery = req.req.URL.RawQuery
	}
	view.proto = req.req.Proto
	if view.proto == "" {
		view.proto = "HTTP/1.1"
	}
	return view
}

func requestURIValue(g *Global) string {
	return requestView(g).URLField()
}

func requestQueryValue(g *Global) string {
	return requestView(g).QueryField()
}

func requestProtoValue(g *Global) string {
	return requestView(g).ProtoField()
}
