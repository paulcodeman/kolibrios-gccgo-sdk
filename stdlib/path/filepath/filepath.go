// Package filepath implements utility routines for manipulating filename paths.
//
// This KolibriOS port provides a Unix-style implementation with '/' as the
// path separator and treats '\\' as an alternate separator for compatibility
// with existing code.
package filepath

import (
    "os"
    "path"
)

const (
    Separator     = os.PathSeparator
    ListSeparator = os.PathListSeparator
)

// Clean returns the shortest path name equivalent to path by purely lexical processing.
func Clean(p string) string {
    return path.Clean(ToSlash(p))
}

// Join joins any number of path elements into a single path, separating them
// with the OS-specific separator. The result is Cleaned.
func Join(elem ...string) string {
    if len(elem) == 0 {
        return ""
    }
    for i := range elem {
        elem[i] = ToSlash(elem[i])
    }
    return path.Join(elem...)
}

// Split splits path immediately following the final separator, separating it
// into a directory and file name component.
func Split(p string) (dir, file string) {
    p = ToSlash(p)
    i := lastSlash(p)
    return p[:i+1], p[i+1:]
}

// Base returns the last element of path. Trailing separators are removed.
func Base(p string) string {
    return path.Base(ToSlash(p))
}

// Ext returns the file name extension used by path.
func Ext(p string) string {
    return path.Ext(ToSlash(p))
}

// IsAbs reports whether the path is absolute.
func IsAbs(p string) bool {
    p = ToSlash(p)
    return len(p) > 0 && p[0] == '/'
}

// VolumeName returns the leading volume name. KolibriOS has no volumes.
func VolumeName(p string) string {
    return ""
}

// ToSlash returns the result of replacing each separator character in path with a slash ('/').
func ToSlash(p string) string {
    changed := false
    for i := 0; i < len(p); i++ {
        if p[i] == '\\' {
            changed = true
            break
        }
    }
    if !changed {
        return p
    }
    out := make([]byte, len(p))
    for i := 0; i < len(p); i++ {
        ch := p[i]
        if ch == '\\' {
            ch = '/'
        }
        out[i] = ch
    }
    return string(out)
}

// FromSlash returns the result of replacing each slash ('/') character in path with a separator character.
func FromSlash(p string) string {
    if Separator == '/' {
        return p
    }
    out := make([]byte, len(p))
    for i := 0; i < len(p); i++ {
        ch := p[i]
        if ch == '/' {
            ch = byte(Separator)
        }
        out[i] = ch
    }
    return string(out)
}

// Abs returns an absolute representation of path.
func Abs(p string) (string, error) {
    if IsAbs(p) {
        return Clean(p), nil
    }
    wd, err := os.Getwd()
    if err != nil {
        return "", err
    }
    return Clean(Join(wd, p)), nil
}

func lastSlash(p string) int {
    for i := len(p) - 1; i >= 0; i-- {
        if p[i] == '/' {
            return i
        }
    }
    return -1
}
