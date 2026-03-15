package dom

import (
	html5 "net/html"
	"strings"
)

func Parse(html string) *Document {
	doc := NewDocument()
	if html == "" {
		return doc
	}

	root, err := html5.Parse(strings.NewReader(html))
	if root == nil {
		return doc
	}
	if err != nil {
		// Parser still returns a best-effort tree; continue.
	}
	appendHTMLNodes(doc, doc.Root, root)
	return doc
}

func appendHTMLNodes(doc *Document, parent *Node, node *html5.Node) {
	if node == nil || doc == nil || parent == nil {
		return
	}
	switch node.Type {
	case html5.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendHTMLNodes(doc, parent, child)
		}
	case html5.ElementNode:
		tag := toLowerASCII(node.Data)
		elem := doc.CreateElement(tag)
		if len(node.Attr) > 0 {
			for _, attr := range node.Attr {
				key := attr.Key
				if attr.Namespace != "" {
					key = attr.Namespace + ":" + attr.Key
				}
				if elem.Attrs == nil {
					elem.Attrs = map[string]string{}
				}
				elem.Attrs[key] = attr.Val
				if attr.Namespace == "" && attr.Key == "id" && attr.Val != "" {
					doc.RegisterID(elem, attr.Val)
				}
			}
		}
		doc.AppendChild(parent, elem)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendHTMLNodes(doc, elem, child)
		}
	case html5.TextNode:
		if node.Data == "" {
			return
		}
		text := doc.CreateText(node.Data)
		doc.AppendChild(parent, text)
	case html5.CommentNode:
		comment := doc.CreateComment(node.Data)
		doc.AppendChild(parent, comment)
	case html5.DoctypeNode:
		// Ignore doctype nodes for now.
	}
}

func parseTag(raw string) (string, map[string]string, bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, false, false
	}
	if strings.HasPrefix(raw, "!") || strings.HasPrefix(raw, "?") {
		return "", nil, false, false
	}
	closing := false
	if strings.HasPrefix(raw, "/") {
		closing = true
		raw = strings.TrimSpace(raw[1:])
	}
	if raw == "" {
		return "", nil, closing, false
	}
	selfClosing := false
	if strings.HasSuffix(raw, "/") {
		selfClosing = true
		raw = strings.TrimSpace(raw[:len(raw)-1])
	}

	i := 0
	for i < len(raw) && !isSpaceByte(raw[i]) && raw[i] != '/' {
		i++
	}
	name := toLowerASCII(raw[:i])
	if name == "" {
		return "", nil, closing, selfClosing
	}
	if closing {
		return name, nil, true, selfClosing
	}

	attrs := map[string]string{}
	for i < len(raw) {
		for i < len(raw) && isSpaceByte(raw[i]) {
			i++
		}
		if i >= len(raw) {
			break
		}
		if raw[i] == '/' {
			selfClosing = true
			i++
			continue
		}
		start := i
		for i < len(raw) && !isSpaceByte(raw[i]) && raw[i] != '=' && raw[i] != '/' {
			i++
		}
		key := toLowerASCII(raw[start:i])
		if key == "" {
			continue
		}
		for i < len(raw) && isSpaceByte(raw[i]) {
			i++
		}
		if i < len(raw) && raw[i] == '=' {
			i++
			for i < len(raw) && isSpaceByte(raw[i]) {
				i++
			}
			if i >= len(raw) {
				attrs[key] = ""
				break
			}
			quote := raw[i]
			if quote == '"' || quote == '\'' {
				i++
				startVal := i
				for i < len(raw) && raw[i] != quote {
					i++
				}
				attrs[key] = raw[startVal:i]
				if i < len(raw) {
					i++
				}
			} else {
				startVal := i
				for i < len(raw) && !isSpaceByte(raw[i]) && raw[i] != '/' {
					i++
				}
				attrs[key] = raw[startVal:i]
			}
		} else {
			attrs[key] = ""
		}
	}

	return name, attrs, closing, selfClosing
}

func isVoidTag(name string) bool {
	switch name {
	case "br", "img", "hr", "meta", "link", "input":
		return true
	}
	return false
}

func DecodeEntities(text string) string {
	if !strings.Contains(text, "&") {
		return text
	}
	var builder strings.Builder
	builder.Grow(len(text))
	data := []byte(text)
	for i := 0; i < len(data); i++ {
		if data[i] == '&' {
			if decoded, size := decodeEntity(data[i:]); size > 0 {
				_ = builder.WriteByte(decoded)
				i += size - 1
				continue
			}
		}
		_ = builder.WriteByte(data[i])
	}
	return builder.String()
}

func decodeEntity(data []byte) (byte, int) {
	switch {
	case hasPrefix(data, "&lt;"):
		return '<', 4
	case hasPrefix(data, "&gt;"):
		return '>', 4
	case hasPrefix(data, "&amp;"):
		return '&', 5
	case hasPrefix(data, "&quot;"):
		return '"', 6
	case hasPrefix(data, "&apos;"):
		return '\'', 6
	case hasPrefix(data, "&nbsp;"):
		return ' ', 6
	}
	return 0, 0
}

func hasPrefix(data []byte, prefix string) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func indexByte(value string, target byte) int {
	for i := 0; i < len(value); i++ {
		if value[i] == target {
			return i
		}
	}
	return -1
}

func toLowerASCII(value string) string {
	if value == "" {
		return ""
	}
	buf := make([]byte, len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		buf[i] = c
	}
	return string(buf)
}
