//go:build kolibrios && gccgo

package os

func mkdirTempExclusive(name string, mode FileMode) error {
	return Mkdir(name, mode)
}
