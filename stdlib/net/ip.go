package net

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// IP address lengths (bytes).
const (
	IPv4len = 4
	IPv6len = 16
)

// An IP is a single IP address.
type IP []byte

// An IPMask is a bitmask that can be used to manipulate IP addresses.
type IPMask []byte

// An IPNet represents an IP network.
type IPNet struct {
	IP   IP
	Mask IPMask
}

var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

// IPv4 returns the IP address (in 16-byte form) of the IPv4 address a.b.c.d.
func IPv4(a, b, c, d byte) IP {
	p := make(IP, IPv6len)
	copy(p, v4InV6Prefix)
	p[12] = a
	p[13] = b
	p[14] = c
	p[15] = d
	return p
}

// IPv4Mask returns the IP mask (in 4-byte form) of the IPv4 mask a.b.c.d.
func IPv4Mask(a, b, c, d byte) IPMask {
	p := make(IPMask, IPv4len)
	p[0] = a
	p[1] = b
	p[2] = c
	p[3] = d
	return p
}

// CIDRMask returns an IPMask consisting of 'ones' 1 bits followed by 0s.
func CIDRMask(ones, bits int) IPMask {
	if bits != 8*IPv4len && bits != 8*IPv6len {
		return nil
	}
	if ones < 0 || ones > bits {
		return nil
	}
	l := bits / 8
	m := make(IPMask, l)
	n := uint(ones)
	for i := 0; i < l; i++ {
		if n >= 8 {
			m[i] = 0xff
			n -= 8
			continue
		}
		m[i] = ^byte(0xff >> n)
		n = 0
	}
	return m
}

var IPv4zero = IPv4(0, 0, 0, 0)

// ParseIP parses s as an IP address, returning nil if s is invalid.
func ParseIP(s string) IP {
	if ip := parseIPv4(s); ip != nil {
		return ip
	}
	return nil
}

func parseIPv4(s string) IP {
	var p [4]byte
	for i := 0; i < IPv4len; i++ {
		if len(s) == 0 {
			return nil
		}
		n := 0
		j := 0
		for j < len(s) && s[j] != '.' {
			c := s[j]
			if c < '0' || c > '9' {
				return nil
			}
			n = n*10 + int(c-'0')
			if n > 255 {
				return nil
			}
			j++
		}
		if j == 0 {
			return nil
		}
		p[i] = byte(n)
		if i < IPv4len-1 {
			if j >= len(s) || s[j] != '.' {
				return nil
			}
			s = s[j+1:]
		} else if j != len(s) {
			return nil
		}
	}
	return IPv4(p[0], p[1], p[2], p[3])
}

// To4 converts an IPv4 address in IPv6 form to a 4-byte representation.
func (ip IP) To4() IP {
	if len(ip) == IPv4len {
		return ip
	}
	if len(ip) == IPv6len && isIPv4Mapped(ip) {
		return ip[12:16]
	}
	return nil
}

// To16 converts an IP address to 16-byte form.
func (ip IP) To16() IP {
	if len(ip) == IPv6len {
		return ip
	}
	if ip4 := ip.To4(); ip4 != nil {
		return IPv4(ip4[0], ip4[1], ip4[2], ip4[3])
	}
	return nil
}

func isIPv4Mapped(ip IP) bool {
	return len(ip) == IPv6len &&
		ip[0] == 0 && ip[1] == 0 && ip[2] == 0 && ip[3] == 0 &&
		ip[4] == 0 && ip[5] == 0 && ip[6] == 0 && ip[7] == 0 &&
		ip[8] == 0 && ip[9] == 0 && ip[10] == 0xff && ip[11] == 0xff
}

// Mask returns the result of masking the IP address with mask.
func (ip IP) Mask(mask IPMask) IP {
	if len(mask) == 0 {
		return nil
	}
	if len(mask) == IPv4len && len(ip) == IPv6len {
		if ip4 := ip.To4(); ip4 != nil {
			ip = ip4
		}
	}
	if len(mask) != len(ip) {
		return nil
	}
	out := make(IP, len(ip))
	for i := 0; i < len(ip); i++ {
		out[i] = ip[i] & mask[i]
	}
	return out
}

// Equal reports whether ip and x are the same IP address.
func (ip IP) Equal(x IP) bool {
	ip4 := ip.To4()
	x4 := x.To4()
	if ip4 != nil && x4 != nil {
		return ip4[0] == x4[0] && ip4[1] == x4[1] && ip4[2] == x4[2] && ip4[3] == x4[3]
	}
	if len(ip) != len(x) {
		return false
	}
	for i := 0; i < len(ip); i++ {
		if ip[i] != x[i] {
			return false
		}
	}
	return true
}

// String returns the string form of the IP address.
func (ip IP) String() string {
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d", ip4[0], ip4[1], ip4[2], ip4[3])
	}
	if len(ip) == IPv6len {
		parts := make([]string, 0, 8)
		for i := 0; i < IPv6len; i += 2 {
			value := uint16(ip[i])<<8 | uint16(ip[i+1])
			parts = append(parts, strconv.FormatUint(uint64(value), 16))
		}
		return strings.Join(parts, ":")
	}
	return ""
}

// Size returns the number of leading ones and total bits.
func (m IPMask) Size() (ones, bits int) {
	bits = len(m) * 8
	ones = 0
	foundZero := false
	for _, b := range m {
		for i := 7; i >= 0; i-- {
			bit := (b >> uint(i)) & 1
			if bit == 1 {
				if foundZero {
					return 0, 0
				}
				ones++
			} else {
				foundZero = true
			}
		}
	}
	return ones, bits
}

// Contains reports whether the network includes ip.
func (n *IPNet) Contains(ip IP) bool {
	if n == nil || len(n.IP) == 0 || len(n.Mask) == 0 {
		return false
	}
	masked := ip.Mask(n.Mask)
	if masked == nil {
		return false
	}
	return n.IP.Mask(n.Mask).Equal(masked)
}

// ParseCIDR parses s as a CIDR notation IP address and prefix length.
func ParseCIDR(s string) (IP, *IPNet, error) {
	i := strings.LastIndex(s, "/")
	if i < 0 {
		return nil, nil, errors.New("invalid CIDR address")
	}
	ip := ParseIP(s[:i])
	if ip == nil {
		return nil, nil, &AddrError{Err: "invalid CIDR address", Addr: s}
	}
	maskBits, err := strconv.Atoi(s[i+1:])
	if err != nil {
		return nil, nil, &AddrError{Err: "invalid CIDR mask", Addr: s}
	}
	bits := 8 * IPv4len
	if ip.To4() == nil {
		bits = 8 * IPv6len
	}
	mask := CIDRMask(maskBits, bits)
	if mask == nil {
		return nil, nil, &AddrError{Err: "invalid CIDR mask", Addr: s}
	}
	ip = ip.Mask(mask)
	return ip, &IPNet{IP: ip, Mask: mask}, nil
}
