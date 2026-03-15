package main

import (
	"dom"
	"strings"
)

type LinkSpan struct {
	Start int
	End   int
	Link  int
}

type RenderLine struct {
	Text  string
	Spans []LinkSpan
}

type Run struct {
	Text string
	Link int
}

func renderDocument(doc *dom.Document, baseURL string) ([]Run, []string) {
	renderer := &htmlRenderer{
		curLink:   -1,
		lastSpace: true,
	}
	if doc != nil && doc.Root != nil {
		renderNode(doc.Root, baseURL, renderer)
	}
	renderer.flush()
	return renderer.runs, renderer.links
}

type htmlRenderer struct {
	runs      []Run
	links     []string
	curLink   int
	lastSpace bool
	buf       strings.Builder
}

func (r *htmlRenderer) flush() {
	if r.buf.Len() == 0 {
		return
	}
	r.runs = append(r.runs, Run{Text: r.buf.String(), Link: r.curLink})
	r.buf.Reset()
}

func (r *htmlRenderer) appendText(text string) {
	for i := 0; i < len(text); i++ {
		c := text[i]
		if isSpaceByte(c) {
			if r.lastSpace {
				continue
			}
			r.lastSpace = true
			_ = r.buf.WriteByte(' ')
			continue
		}
		r.lastSpace = false
		_ = r.buf.WriteByte(c)
	}
}

func (r *htmlRenderer) appendNewline() {
	r.flush()
	r.runs = append(r.runs, Run{Text: "\n", Link: -1})
	r.lastSpace = true
}

func (r *htmlRenderer) setLink(link int) {
	if link == r.curLink {
		return
	}
	r.flush()
	r.curLink = link
}

func renderNode(node *dom.Node, baseURL string, r *htmlRenderer) {
	if node == nil {
		return
	}

	switch node.Type {
	case dom.TextNode:
		r.appendText(node.Text)
		return
	case dom.CommentNode:
		return
	case dom.DocumentNode:
		for _, child := range node.Children {
			renderNode(child, baseURL, r)
		}
		return
	case dom.ElementNode:
	default:
		return
	}

	tag := node.Tag
	switch tag {
	case "script", "style", "head", "title", "meta", "link":
		return
	case "br":
		r.appendNewline()
		return
	case "p", "div", "tr", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol":
		r.appendNewline()
		for _, child := range node.Children {
			renderNode(child, baseURL, r)
		}
		r.appendNewline()
		return
	case "li":
		r.appendNewline()
		r.appendText("- ")
		for _, child := range node.Children {
			renderNode(child, baseURL, r)
		}
		return
	case "a":
		prev := r.curLink
		if href, ok := node.Attrs["href"]; ok {
			if resolved := resolveURL(baseURL, href); resolved != "" {
				r.links = append(r.links, resolved)
				r.setLink(len(r.links) - 1)
			}
		}
		for _, child := range node.Children {
			renderNode(child, baseURL, r)
		}
		r.setLink(prev)
		return
	}

	for _, child := range node.Children {
		renderNode(child, baseURL, r)
	}
}

func wrapRuns(runs []Run, width int) []RenderLine {
	if width <= 0 {
		width = 1
	}

	lines := []RenderLine{}
	lineBuf := make([]byte, 0, width)
	spans := make([]LinkSpan, 0, 4)
	col := 0
	lineDirty := false

	flush := func(force bool) {
		if !force && !lineDirty {
			return
		}
		lines = append(lines, RenderLine{
			Text:  string(lineBuf),
			Spans: append([]LinkSpan(nil), spans...),
		})
		lineBuf = lineBuf[:0]
		spans = spans[:0]
		col = 0
		lineDirty = false
	}

	for _, run := range runs {
		text := run.Text
		for i := 0; i < len(text); i++ {
			c := text[i]
			if c == '\n' {
				flush(true)
				continue
			}
			if c == ' ' && col == 0 {
				continue
			}
			if col >= width {
				flush(true)
				if c == ' ' {
					continue
				}
			}
			lineBuf = append(lineBuf, c)
			lineDirty = true
			if run.Link >= 0 {
				if len(spans) == 0 || spans[len(spans)-1].Link != run.Link || spans[len(spans)-1].End != col {
					spans = append(spans, LinkSpan{Start: col, End: col + 1, Link: run.Link})
				} else {
					spans[len(spans)-1].End++
				}
			}
			col++
		}
	}

	flush(len(lines) == 0 || lineDirty)
	return lines
}
