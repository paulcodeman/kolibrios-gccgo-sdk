//go:build kolibrios && gccgo

package os

import (
	"errors"
	"kos"
	"path"
)

func executable() (string, error) {
	name := kos.LoaderPath()
	if !path.IsAbs(name) {
		return "", errors.New("os: executable path is unavailable")
	}
	return path.Clean(name), nil
}
