package phpctx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func (g *Global) ResponseHeaders() http.Header {
	if g == nil {
		return nil
	}
	if g.responseHeader == nil {
		g.responseHeader = make(http.Header)
	}
	return g.responseHeader.Clone()
}

func (g *Global) ResponseStatusCode() int {
	if g == nil {
		return 0
	}
	return g.responseStatus
}

func (g *Global) HeadersSent() bool {
	return false
}

func (g *Global) SetResponseStatus(code int) {
	if g == nil || code <= 0 {
		return
	}
	g.responseStatus = code
}

func (g *Global) RemoveResponseHeader(name string) {
	if g == nil {
		return
	}
	if name == "" {
		g.responseHeader = make(http.Header)
		return
	}
	if g.responseHeader == nil {
		return
	}
	g.responseHeader.Del(name)
}

func (g *Global) AddResponseHeaderLine(line string, replace bool, statusCode int) error {
	if g == nil {
		return nil
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	if statusCode > 0 {
		g.SetResponseStatus(statusCode)
	}
	if rawStatus, ok := parseRawStatusCode(line); ok {
		g.SetResponseStatus(rawStatus)
		return nil
	}

	separator := strings.IndexByte(line, ':')
	if separator < 0 {
		return errors.New("header(): invalid header")
	}

	key := strings.TrimSpace(line[:separator])
	value := strings.TrimSpace(line[separator+1:])
	if key == "" {
		return errors.New("header(): invalid header name")
	}
	if strings.EqualFold(key, "Status") {
		code, ok := parseLeadingStatusCode(value)
		if !ok {
			return errors.New("header(): invalid status header")
		}
		g.SetResponseStatus(code)
		return nil
	}

	if g.responseHeader == nil {
		g.responseHeader = make(http.Header)
	}
	if replace {
		g.responseHeader.Set(key, value)
	} else {
		g.responseHeader.Add(key, value)
	}

	if strings.EqualFold(key, "Location") && statusCode <= 0 {
		switch g.responseStatus {
		case 0, http.StatusOK:
			g.responseStatus = http.StatusFound
		}
	}
	return nil
}

func parseRawStatusCode(line string) (int, bool) {
	if strings.HasPrefix(strings.ToUpper(line), "HTTP/") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		return parseLeadingStatusCode(fields[1])
	}
	return 0, false
}

func parseLeadingStatusCode(value string) (int, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0, false
	}
	code, err := strconv.Atoi(fields[0])
	if err != nil || code <= 0 {
		return 0, false
	}
	return code, true
}
