//go:build !plan9
// +build !plan9

package browserfs

import (
	"github.com/knusbaum/go9p"
)

func post(srv go9p.Srv) (err error) {
	return go9p.PostSrv("mycel", srv)
}
