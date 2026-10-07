// Execute OpenCode's unchanged configuration finder and concurrency dependency.
package main

import (
	"errors"
	"io/fs"
	"net"
	"os"
	"strings"

	"github.com/sagikazarmark/locafero"
	"github.com/sourcegraph/conc/iter"
	"github.com/spf13/afero"
	"kos"
)

func main() {
	kos.DebugString("OPENCODE_FINDER_START")
	console, ok := kos.OpenConsole("OpenCode original finder dependency test")
	if !ok {
		panic("console unavailable")
	}
	defer console.Close()
	if net.IPMask(nil).String() != "<nil>" || (net.IPMask{}).String() != "<nil>" ||
		net.IPv4Mask(255, 255, 255, 0).String() != "ffffff00" ||
		net.CIDRMask(65, 128).String() != "ffffffffffffffff8000000000000000" ||
		(net.IPMask{0, 0xab, 0xcd, 0xef, 0xff}).String() != "00abcdefff" {
		panic("upstream mask string formatting")
	}
	if os.ErrInvalid != fs.ErrInvalid || os.ErrPermission != fs.ErrPermission ||
		os.ErrExist != fs.ErrExist || os.ErrNotExist != fs.ErrNotExist || os.ErrClosed != fs.ErrClosed {
		panic("shared filesystem error identity")
	}
	kos.DebugString("FILESYSTEM_ERRORS_MASKS_OK")
	v6 := net.ParseIP("2001:db8:0:1::9")
	if v6 == nil || v6.String() != "2001:db8:0:1::9" || v6.To4() != nil || len(v6.To16()) != 16 {
		panic("original IPv6 parsing and compressed formatting")
	}
	for _, invalid := range []string{"01.2.3.4", "2001::db8::1", "::ffff:999.1.2.3", "fe80::1%eth0"} {
		if net.ParseIP(invalid) != nil {
			panic("invalid IP accepted")
		}
	}
	if mapped := net.ParseIP("::ffff:192.0.2.7"); mapped == nil || mapped.String() != "192.0.2.7" || !mapped.Equal(net.IPv4(192, 0, 2, 7)) {
		panic("original IPv4-mapped IPv6")
	}
	_, network, err := net.ParseCIDR("2001:db8::1234/64")
	if err != nil || network.String() != "2001:db8::/64" || network.Network() != "ip+net" ||
		!network.Contains(net.ParseIP("2001:db8::5")) || network.Contains(net.ParseIP("2001:db9::5")) {
		panic("original IPv6 CIDR networks")
	}
	_, network, err = net.ParseCIDR("192.0.2.123/24")
	if err != nil || network.String() != "192.0.2.0/24" {
		panic("original IPv4 CIDR networks")
	}
	irregular := &net.IPNet{IP: net.IP{192, 0, 2, 0}, Mask: net.IPMask{255, 0, 255, 0}}
	if irregular.String() != "192.0.2.0/ff00ff00" || net.IP(nil).String() != "<nil>" || (*net.IPNet)(nil).String() != "<nil>" {
		panic("original noncanonical masks or nil IP formatting")
	}
	text, err := v6.MarshalText()
	var decoded net.IP
	if err != nil || decoded.UnmarshalText(text) != nil || !decoded.Equal(v6) {
		panic("original IP text marshaling")
	}
	if !net.ParseIP("10.1.2.3").IsPrivate() || !net.ParseIP("::1").IsLoopback() || !net.ParseIP("ff02::1").IsMulticast() {
		panic("original IP classification")
	}
	kos.DebugString("UPSTREAM_IP_NETWORKS_OK")
	const root = "/hd0/1/FINDCASE"
	if err := os.RemoveAll(root); err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	for _, dir := range []string{root + "/configs", root + "/logs"} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			panic(err)
		}
	}
	for _, file := range []string{root + "/configs/settings.json", root + "/logs/application.log"} {
		if err := os.WriteFile(file, []byte("original finder test\n"), 0600); err != nil {
			panic(err)
		}
	}
	kos.DebugString("FINDER_FILES_READY")
	finder := locafero.Finder{
		Paths: []string{root + "/configs", root + "/logs"},
		Names: []string{"settings.json", "*.log"},
		Type:  locafero.FileTypeFile,
	}
	found, err := finder.Find(afero.NewOsFs())
	if err != nil {
		kos.DebugString("FINDER_ERROR:" + err.Error())
		panic(err)
	}
	kos.DebugString("FINDER_RETURNED")
	for _, name := range found {
		kos.DebugString("FOUND:" + name)
	}
	if len(found) != 2 {
		panic("original finder result count")
	}
	seenSettings, seenLog := false, false
	for _, name := range found {
		seenSettings = seenSettings || strings.EqualFold(name, root+"/configs/settings.json")
		seenLog = seenLog || strings.EqualFold(name, root+"/logs/application.log")
	}
	if !seenSettings || !seenLog {
		panic("original finder paths")
	}
	kos.DebugString("ORIGINAL_FINDER_OK")
	type row struct{ value int }
	rows := []row{{7}, {11}, {13}}
	sentinel := errors.New("mapper callback error")
	mapper := iter.Mapper[row, int]{MaxGoroutines: 3}
	values, err := mapper.MapErr(rows, func(v *row) (int, error) {
		if v.value == 11 {
			return v.value * 2, sentinel
		}
		return v.value * 2, nil
	})
	if len(values) != 3 || values[0] != 14 || values[1] != 22 || values[2] != 26 || !errors.Is(err, sentinel) {
		panic("original concurrent mapper results or error propagation")
	}
	kos.DebugString("ORIGINAL_MAPPER_OK")
	if empty, err := (locafero.Finder{}).Find(afero.NewOsFs()); err != nil || len(empty) != 0 {
		panic("empty original finder")
	}
	kos.DebugString("OPENCODE_FINDER_PASS")
	console.WriteString("OPENCODE_FINDER_PASS\n")
}
