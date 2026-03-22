package textunicode // import "golang.org/x/text/encoding/unicode"

import (
	textencoding "golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type utf8Encoding struct{}

func (utf8Encoding) NewDecoder() *textencoding.Decoder {
	return &textencoding.Decoder{Transformer: transform.Nop}
}

func (utf8Encoding) NewEncoder() *textencoding.Encoder {
	return &textencoding.Encoder{Transformer: transform.Nop}
}

var UTF8 textencoding.Encoding = utf8Encoding{}
