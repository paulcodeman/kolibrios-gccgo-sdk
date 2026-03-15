package dom

import "strings"

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
	Root    *Node
	nodes   []*Node
	idIndex map[string]*Node
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
	walk(doc.Root, func(node *Node) {
		if node.Type == ElementNode && node.Tag == tag {
			out = append(out, node)
		}
	})
	return out
}

// QuerySelector implements a minimal subset: "#id", ".class", or tag name.
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
		walk(doc.Root, func(node *Node) {
			if found != nil {
				return
			}
			if node.Type != ElementNode {
				return
			}
			if classAttr, ok := node.Attrs["class"]; ok {
				if classListContains(classAttr, className) {
					found = node
				}
			}
		})
		return found
	}
	tag := toLowerASCII(selector)
	var found *Node
	walk(doc.Root, func(node *Node) {
		if found != nil {
			return
		}
		if node.Type == ElementNode && node.Tag == tag {
			found = node
		}
	})
	return found
}

func walk(node *Node, visit func(*Node)) {
	if node == nil {
		return
	}
	visit(node)
	for _, child := range node.Children {
		walk(child, visit)
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
