package main

import (
	html5 "net/html"
	"strings"
)

type NodeType uint8

const (
	DocumentNode NodeType = iota
	ElementNode
	TextNode
	CommentNode
)

type Node struct {
	ID       int
	Type     NodeType
	Tag      string
	Attrs    map[string]string
	Text     string
	Parent   *Node
	Children []*Node
}

type Document struct {
	Root             *Node
	nodes            []*Node
	idIndex          map[string]*Node
	fontFamilies     []fontFamilyEntry
	stylesheet       *pageStylesheet
	stylesheetParsed bool
}

func NewDocument() *Document {
	doc := &Document{
		idIndex: map[string]*Node{},
	}
	root := doc.newNode(DocumentNode)
	root.Tag = "#document"
	doc.Root = root
	return doc
}

func (doc *Document) newNode(kind NodeType) *Node {
	node := &Node{
		ID:   len(doc.nodes),
		Type: kind,
	}
	doc.nodes = append(doc.nodes, node)
	return node
}

func (doc *Document) CreateElement(tag string) *Node {
	node := doc.newNode(ElementNode)
	node.Tag = tag
	node.Attrs = map[string]string{}
	return node
}

func (doc *Document) CreateText(text string) *Node {
	node := doc.newNode(TextNode)
	node.Text = text
	return node
}

func (doc *Document) CreateComment(text string) *Node {
	node := doc.newNode(CommentNode)
	node.Text = text
	return node
}

func (doc *Document) AppendChild(parent *Node, child *Node) {
	if parent == nil || child == nil {
		return
	}
	child.Parent = parent
	parent.Children = append(parent.Children, child)
}

func (doc *Document) RegisterID(node *Node, id string) {
	if doc == nil || node == nil || id == "" {
		return
	}
	doc.idIndex[id] = node
}

func (doc *Document) GetElementByID(id string) *Node {
	if doc == nil || id == "" {
		return nil
	}
	return doc.idIndex[id]
}

func (doc *Document) GetElementsByTagName(tag string) []*Node {
	if doc == nil || tag == "" {
		return nil
	}
	tag = toLowerASCII(tag)
	var out []*Node
	walkNode(doc.Root, func(node *Node) {
		if node.Type == ElementNode && node.Tag == tag {
			out = append(out, node)
		}
	})
	return out
}

func (doc *Document) QuerySelector(selector string) *Node {
	if doc == nil || selector == "" {
		return nil
	}
	if strings.HasPrefix(selector, "#") {
		return doc.GetElementByID(selector[1:])
	}
	if strings.HasPrefix(selector, ".") {
		className := selector[1:]
		var found *Node
		walkNode(doc.Root, func(node *Node) {
			if found != nil || node.Type != ElementNode {
				return
			}
			if classAttr, ok := node.Attrs["class"]; ok && classListContains(classAttr, className) {
				found = node
			}
		})
		return found
	}
	tag := toLowerASCII(selector)
	var found *Node
	walkNode(doc.Root, func(node *Node) {
		if found != nil {
			return
		}
		if node.Type == ElementNode && node.Tag == tag {
			found = node
		}
	})
	return found
}

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
	}
}

func walkNode(node *Node, visit func(*Node)) {
	if node == nil {
		return
	}
	visit(node)
	for _, child := range node.Children {
		walkNode(child, visit)
	}
}

func classListContains(classAttr string, target string) bool {
	if classAttr == "" || target == "" {
		return false
	}
	for _, item := range strings.Split(classAttr, " ") {
		if item == target {
			return true
		}
	}
	return false
}
