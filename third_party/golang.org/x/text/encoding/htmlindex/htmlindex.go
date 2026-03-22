package htmlindex // import "golang.org/x/text/encoding/htmlindex"

import (
	"strings"

	textencoding "golang.org/x/text/encoding"
	textunicode "golang.org/x/text/encoding/unicode"
)

// Get returns a small compatibility subset for the encodings currently needed
// by the KolibriOS browser path. Unknown charsets fall back to UTF-8.
func Get(name string) (textencoding.Encoding, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "utf-8", "utf8":
		return textunicode.UTF8, nil
	default:
		return textunicode.UTF8, nil
	}
}
