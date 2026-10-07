package http

import (
	"crypto/tls"
	"net/http/httptrace"
)

// Retain the SDK progress API through real upstream transport events.
func (t *Transport) withProgress(req *Request) *Request {
	if req == nil || (req.Progress == nil && t.Progress == nil) { return req }
	progress := func(message string) {
		if req.Progress != nil { req.Progress(message) }
		if t.Progress != nil { t.Progress(message) }
	}
	trace := &httptrace.ClientTrace{
		GetConn: func(address string) { progress("Connecting to " + address) },
		DNSStart: func(info httptrace.DNSStartInfo) { progress("Resolving " + info.Host) },
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if info.Err != nil { progress("DNS: " + info.Err.Error()) }
		},
		GotConn: func(info httptrace.GotConnInfo) { progress("Connected") },
		TLSHandshakeStart: func() { progress("TLS handshake") },
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if err != nil { progress("TLS: " + err.Error()) } else { progress("TLS established") }
		},
		GotFirstResponseByte: func() { progress("Receiving response") },
	}
	return req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
}
