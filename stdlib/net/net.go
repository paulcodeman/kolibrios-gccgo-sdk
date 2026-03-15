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
	Err  string
	Name string
}

func (err *DNSError) Error() string {
	if err == nil {
		return ""
	}
	if err.Name == "" {
		return err.Err
	}

	return "lookup " + err.Name + ": " + err.Err
}

func (err *DNSError) As(target interface{}) bool {
	if err == nil {
		return false
	}

	switch typed := target.(type) {
	case **DNSError:
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
