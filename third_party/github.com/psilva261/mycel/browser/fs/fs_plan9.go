//go:build plan9
// +build plan9

package browserfs

import (
	"fmt"
	"github.com/knusbaum/go9p"
)

// postPlan9 keeps the original upstream plan9 variant documented in-tree, but
// KolibriOS uses the non-plan9 service path from fs_unix.go.
func postPlan9(srv go9p.Srv) (err error) {
	_ = srv
	return fmt.Errorf("plan9 mount path is not used on kolibrios")
}
