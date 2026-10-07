//go:build kolibrios && gccgo

package net

import (
	"context"
	"errors"
	"kos"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

var errNoSuchHost = errors.New("no such host")

// KolibriOS currently supplies TCP sockets to package net. DNS over TCP is
// the actual upstream framed transport, rather than a fabricated SRV/TXT
// result. UDP and OS-specific search-domain configuration remain to be ported.
func (r *Resolver) lookup(ctx context.Context, name string, qtype dnsmessage.Type) (dnsmessage.Parser, string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if !isDomainName(name) || strings.HasSuffix(strings.ToLower(strings.TrimSuffix(name, ".")), ".onion") {
		return dnsmessage.Parser{}, "", &DNSError{Err: errNoSuchHost.Error(), Name: name, IsNotFound: true}
	}
	rooted := name
	if !strings.HasSuffix(rooted, ".") {
		rooted += "."
	}
	domain, err := dnsmessage.NewName(rooted)
	if err != nil {
		return dnsmessage.Parser{}, "", &DNSError{Err: err.Error(), Name: name}
	}
	servers := nativeDNSServers()
	if len(servers) == 0 && r != nil && r.Dial != nil {
		// A custom dialer (including gRPC's authority dialer) selects its
		// own transport. A literal default avoids recursive name lookup.
		servers = []string{"127.0.0.1:53"}
	}
	if len(servers) == 0 {
		return dnsmessage.Parser{}, "", &DNSError{Err: "no DNS server configured", Name: name}
	}
	var last error
	var lastServer string
	for attempt := 0; attempt < 2; attempt++ {
		for _, server := range servers {
			lastServer = server
			p, h, err := r.exchange(ctx, server, dnsmessage.Question{Name: domain, Type: qtype}, 5*time.Second, useTCPOnly)
			if err == nil {
				err = checkHeader(&p, h)
			}
			if err == nil {
				err = skipToAnswer(&p, qtype)
			}
			if err == nil {
				return p, server, nil
			}
			last = err
			if err == errNoSuchHost {
				return dnsmessage.Parser{}, server, dnsLookupError(err, name, server)
			}
			select {
			case <-ctx.Done():
				return dnsmessage.Parser{}, server, dnsLookupError(ctx.Err(), name, server)
			default:
			}
		}
	}
	return dnsmessage.Parser{}, lastServer, dnsLookupError(last, name, lastServer)
}

func nativeDNSServers() []string {
	var servers []string
	for slot := uint32(0); slot < 16; slot++ {
		regs := kos.SyscallRegs{EAX: 76, EBX: 1<<16 | slot<<8 | 4}
		kos.SyscallRaw(&regs)
		address := regs.EAX
		if address == 0 || address == ^uint32(0) {
			continue
		}
		server := JoinHostPort(IPv4(byte(address), byte(address>>8), byte(address>>16), byte(address>>24)).String(), "53")
		duplicate := false
		for _, old := range servers {
			if old == server {
				duplicate = true
				break
			}
		}
		if !duplicate {
			servers = append(servers, server)
		}
	}
	return servers
}

func dnsLookupError(err error, name, server string) *DNSError {
	nerr := &DNSError{Err: err.Error(), Name: name, Server: server, IsNotFound: err == errNoSuchHost, IsTemporary: err == errServerTemporarilyMisbehaving}
	if timeout, ok := err.(Error); ok {
		nerr.IsTimeout = timeout.Timeout()
		nerr.IsTemporary = nerr.IsTemporary || timeout.Temporary()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		nerr.IsTimeout = true
	}
	return nerr
}

func mapErr(err error) error { return err }

func (r *Resolver) lookupHost(ctx context.Context, host string) ([]string, error) {
	if ctx == nil {
		panic("nil context")
	}
	select {
	case <-ctx.Done():
		return nil, dnsLookupError(ctx.Err(), host, "")
	default:
	}
	if r == nil || (!r.PreferGo && r.Dial == nil) {
		// Keep the native resolver for its OS-specific host configuration.
		// As in an upstream cgo lookup, cancellation stops waiting for the
		// call but cannot interrupt a native resolver already in progress.
		type result struct {
			addresses []string
			err       error
		}
		done := make(chan result, 1)
		go func() { addresses, err := LookupHost(host); done <- result{addresses, err} }()
		select {
		case <-ctx.Done():
			return nil, dnsLookupError(ctx.Err(), host, "")
		case value := <-done:
			return value.addresses, value.err
		}
	}
	var addresses []string
	var last error
	for _, kind := range []dnsmessage.Type{dnsmessage.TypeA, dnsmessage.TypeAAAA} {
		p, server, err := r.lookup(ctx, host, kind)
		if err != nil {
			last = err
			if r.StrictErrors {
				if nerr, ok := err.(Error); ok && nerr.Temporary() {
					return nil, err
				}
			}
			continue
		}
		for {
			h, err := p.AnswerHeader()
			if err == dnsmessage.ErrSectionDone {
				break
			}
			if err != nil {
				return nil, dnsLookupError(errCannotUnmarshalDNSMessage, host, server)
			}
			if h.Type != kind {
				if err := p.SkipAnswer(); err != nil {
					return nil, dnsLookupError(err, host, server)
				}
				continue
			}
			if kind == dnsmessage.TypeA {
				a, err := p.AResource()
				if err != nil {
					return nil, dnsLookupError(err, host, server)
				}
				addresses = append(addresses, IP(a.A[:]).String())
			} else {
				a, err := p.AAAAResource()
				if err != nil {
					return nil, dnsLookupError(err, host, server)
				}
				addresses = append(addresses, IP(a.AAAA[:]).String())
			}
		}
	}
	if len(addresses) > 0 {
		return addresses, nil
	}
	if last != nil {
		return nil, last
	}
	return nil, dnsLookupError(errNoSuchHost, host, "")
}
