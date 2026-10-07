// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import "context"

var DefaultResolver = &Resolver{}

type Resolver struct {
	// PreferGo controls whether Go's built-in DNS resolver is preferred
	// on platforms where it's available. It is equivalent to setting
	// GODEBUG=netdns=go, but scoped to just this resolver.
	PreferGo bool

	// StrictErrors controls the behavior of temporary errors
	// (including timeout, socket errors, and SERVFAIL) when using
	// Go's built-in resolver. For a query composed of multiple
	// sub-queries (such as an A+AAAA address lookup, or walking the
	// DNS search list), this option causes such errors to abort the
	// whole query instead of returning a partial result. This is
	// not enabled by default because it may affect compatibility
	// with resolvers that process AAAA queries incorrectly.
	StrictErrors bool

	// Dial optionally specifies an alternate dialer for use by
	// Go's built-in DNS resolver to make TCP and UDP connections
	// to DNS services. The host in the address parameter will
	// always be a literal IP address and not a host name, and the
	// port in the address parameter will be a literal port number
	// and not a service name.
	// If the Conn returned is also a PacketConn, sent and received DNS
	// messages must adhere to RFC 1035 section 4.2.1, "UDP usage".
	// Otherwise, DNS messages transmitted over Conn must adhere
	// to RFC 7766 section 5, "Transport Protocol Selection".
	// If nil, the default dialer is used.
	Dial func(ctx context.Context, network, address string) (Conn, error)

}

func (r *Resolver) LookupHost(ctx context.Context, host string) (addrs []string, err error) {
	// Make sure that no matter what we do later, host=="" is rejected.
	// parseIP, for example, does accept empty strings.
	if host == "" {
		return nil, &DNSError{Err: errNoSuchHost.Error(), Name: host, IsNotFound: true}
	}
	if ip, _ := parseIPZone(host); ip != nil {
		return []string{host}, nil
	}
	return r.lookupHost(ctx, host)
}

func LookupSRV(service, proto, name string) (cname string, addrs []*SRV, err error) {
	return DefaultResolver.LookupSRV(context.Background(), service, proto, name)
}

func (r *Resolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*SRV, error) {
	cname, addrs, err := r.lookupSRV(ctx, service, proto, name)
	if err != nil {
		return "", nil, err
	}
	if cname != "" && !isDomainName(cname) {
		return "", nil, &DNSError{Err: "SRV header name is invalid", Name: name}
	}
	filteredAddrs := make([]*SRV, 0, len(addrs))
	for _, addr := range addrs {
		if addr == nil {
			continue
		}
		if !isDomainName(addr.Target) {
			continue
		}
		filteredAddrs = append(filteredAddrs, addr)
	}
	if len(addrs) != len(filteredAddrs) {
		return cname, filteredAddrs, &DNSError{Err: errMalformedDNSRecordsDetail, Name: name}
	}
	return cname, filteredAddrs, nil
}

func LookupTXT(name string) ([]string, error) {
	return DefaultResolver.lookupTXT(context.Background(), name)
}

func (r *Resolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.lookupTXT(ctx, name)
}

var errMalformedDNSRecordsDetail = "DNS response contained records which contain invalid names"
