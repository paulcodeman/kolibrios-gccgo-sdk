package net

import (
	"kos"
	"strings"
)

type AddrError struct {
	Err  string
	Addr string
}

func (err *AddrError) Error() string {
	if err == nil {
		return ""
	}
	if err.Addr == "" {
		return err.Err
	}

	return err.Err + ": " + err.Addr
}

func (err *AddrError) As(target interface{}) bool {
	if err == nil {
		return false
	}

	switch typed := target.(type) {
	case **AddrError:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	case *error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	}

	return false
}

type DNSError struct {
	Err         string // description of the error
	Name        string // name looked for
	Server      string // server used
	IsTimeout   bool   // if true, timed out; not all timeouts set this
	IsTemporary bool   // if true, error is temporary; not all errors set this
	IsNotFound  bool   // if true, host could not be found
}

func (e *DNSError) Error() string {
	if e == nil {
		return "<nil>"
	}
	s := "lookup " + e.Name
	if e.Server != "" {
		s += " on " + e.Server
	}
	s += ": " + e.Err
	return s
}

// Timeout reports whether the DNS lookup is known to have timed out.
// This is not always known; a DNS lookup may fail due to a timeout
// and return a DNSError for which Timeout returns false.
func (e *DNSError) Timeout() bool { return e.IsTimeout }

// Temporary reports whether the DNS error is known to be temporary.
// This is not always known; a DNS lookup may fail due to a temporary
// error and return a DNSError for which Temporary returns false.
func (e *DNSError) Temporary() bool { return e.IsTimeout || e.IsTemporary }

func LookupHost(host string) ([]string, error) {
	network, ok := kos.LoadNetwork()
	if !ok {
		return nil, &DNSError{
			Err:  "network.obj unavailable",
			Name: host,
		}
	}

	addrs, err := network.LookupHost(host)
	if err != nil {
		return nil, &DNSError{
			Err:  err.Error(),
			Name: host,
		}
	}
	if len(addrs) == 0 {
		return nil, &DNSError{
			Err:  "no such host",
			Name: host,
		}
	}

	return addrs, nil
}

func JoinHostPort(host string, port string) string {
	if strings.Index(host, ":") >= 0 && !(strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]")) {
		host = "[" + host + "]"
	}

	return host + ":" + port
}

func SplitHostPort(hostport string) (host string, port string, err error) {
	if hostport == "" {
		return "", "", &AddrError{Err: "missing port in address", Addr: hostport}
	}

	if strings.HasPrefix(hostport, "[") {
		end := strings.Index(hostport, "]")
		if end < 0 {
			return "", "", &AddrError{Err: "missing ']' in address", Addr: hostport}
		}
		if end+1 >= len(hostport) || hostport[end+1] != ':' {
			return "", "", &AddrError{Err: "missing port in address", Addr: hostport}
		}

		host = hostport[1:end]
		port = hostport[end+2:]
		if port == "" {
			return "", "", &AddrError{Err: "missing port in address", Addr: hostport}
		}
		return host, port, nil
	}

	separator := strings.LastIndex(hostport, ":")
	if separator < 0 {
		return "", "", &AddrError{Err: "missing port in address", Addr: hostport}
	}
	if strings.Index(hostport[:separator], ":") >= 0 {
		return "", "", &AddrError{Err: "too many colons in address", Addr: hostport}
	}

	host = hostport[:separator]
	port = hostport[separator+1:]
	if port == "" {
		return "", "", &AddrError{Err: "missing port in address", Addr: hostport}
	}
	return host, port, nil
}
