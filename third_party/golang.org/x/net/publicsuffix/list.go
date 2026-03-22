package publicsuffix

import (
	"errors"
	"net"
	"strings"
)

type list struct{}

var List list

var multiLabel = map[string]bool{
	"ac.jp":   true,
	"ac.uk":   true,
	"co.in":   true,
	"co.jp":   true,
	"co.kr":   true,
	"co.uk":   true,
	"com.au":  true,
	"com.br":  true,
	"com.cn":  true,
	"com.tr":  true,
	"edu.au":  true,
	"firm.in": true,
	"gen.in":  true,
	"go.jp":   true,
	"gov.cn":  true,
	"gov.uk":  true,
	"ind.in":  true,
	"net.au":  true,
	"net.cn":  true,
	"net.in":  true,
	"ne.jp":   true,
	"or.jp":   true,
	"org.au":  true,
	"org.cn":  true,
	"org.in":  true,
	"org.uk":  true,
}

func (list) PublicSuffix(domain string) string {
	domain = strings.ToLower(strings.Trim(domain, "."))
	if domain == "" || net.ParseIP(domain) != nil {
		return domain
	}
	labels := strings.Split(domain, ".")
	if len(labels) <= 1 {
		return domain
	}
	lastTwo := labels[len(labels)-2] + "." + labels[len(labels)-1]
	if multiLabel[lastTwo] {
		return lastTwo
	}
	return labels[len(labels)-1]
}

func (list) String() string {
	return "kolibrios heuristic public suffix list"
}

func PublicSuffix(domain string) (string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", errors.New("publicsuffix: empty domain")
	}
	return List.PublicSuffix(domain), nil
}

func EffectiveTLDPlusOne(domain string) (string, error) {
	domain = strings.ToLower(strings.Trim(domain, "."))
	if domain == "" {
		return "", errors.New("publicsuffix: empty domain")
	}
	suffix := List.PublicSuffix(domain)
	if suffix == "" || suffix == domain {
		return "", errors.New("publicsuffix: no registrable domain")
	}
	index := len(domain) - len(suffix)
	if index <= 0 || domain[index-1] != '.' {
		return "", errors.New("publicsuffix: invalid suffix")
	}
	prefix := domain[:index-1]
	prevDot := strings.LastIndex(prefix, ".")
	if prevDot < 0 {
		return domain, nil
	}
	return domain[prevDot+1:], nil
}
