package js

import (
	"io"
	"os"
)

// The Kolibri toolchain compiles all variant files together, so keep the
// plan9-specific helpers under distinct names and let js_unix.go provide the
// active implementation.
func (js *JS) hangupPlan9() {}

func (js *JS) callSparkleCtlPlan9() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile("/mnt/sparkle/ctl", os.O_RDWR, 0600)
}
