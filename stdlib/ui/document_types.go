package ui

type DocumentNodeKind uint8

const (
	DocumentNodeElement DocumentNodeKind = iota
	DocumentNodeText
)

type FragmentKind uint8

const (
	FragmentKindBlock FragmentKind = iota
	FragmentKindText
)

type DocumentNode struct {
	Kind     DocumentNodeKind
	Name     string
	Text     string
	Style    Style
	OnClick  interface{}
	Parent   *DocumentNode
	Children []*DocumentNode
}

type Fragment struct {
	Kind        FragmentKind
	Node        *DocumentNode
	Style       Style
	Bounds      Rect
	PaintBounds Rect
	Content     Rect
	Text        string
	Children    []*Fragment

	font    *ttfFont
	metrics fontMetrics
	lines   []textLine
}

type FragmentDisplayItem struct {
	Fragment *Fragment
	Bounds   Rect
	Paint    Rect
	Clip     Rect
	ClipSet  bool
}

type FragmentDisplayList struct {
	items []FragmentDisplayItem
}

type Document struct {
	Root *DocumentNode

	rootFragment *Fragment
	displayList  FragmentDisplayList
	viewport     Rect
	content      Rect
	host         *DocumentView
}

func NewDocument(root *DocumentNode) *Document {
	document := &Document{}
	document.SetRoot(root)
	return document
}

func NewDocumentElement(name string, style Style, children ...*DocumentNode) *DocumentNode {
	node := &DocumentNode{
		Kind:  DocumentNodeElement,
		Name:  name,
		Style: style,
	}
	node.Append(children...)
	return node
}

func NewDocumentText(text string, style Style) *DocumentNode {
	return &DocumentNode{
		Kind:  DocumentNodeText,
		Text:  text,
		Style: style,
	}
}

func (node *DocumentNode) Append(children ...*DocumentNode) {
	if node == nil {
		return
	}
	for _, child := range children {
		if child == nil {
			continue
		}
		node.Children = append(node.Children, child)
		linkDocumentTree(node, child)
	}
}

func (node *DocumentNode) ClearChildren() {
	if node == nil {
		return
	}
	for _, child := range node.Children {
		if child != nil {
			child.Parent = nil
		}
	}
	node.Children = nil
}

func (document *Document) SetRoot(root *DocumentNode) {
	if document == nil {
		return
	}
	document.clearLayout()
	document.Root = root
	linkDocumentTree(nil, root)
	if document.host != nil {
		document.host.MarkLayoutDirty()
	}
}

func (document *Document) RootFragment() *Fragment {
	if document == nil {
		return nil
	}
	return document.rootFragment
}

func (document *Document) DisplayList() FragmentDisplayList {
	if document == nil {
		return FragmentDisplayList{}
	}
	return document.displayList
}

func (document *Document) Viewport() Rect {
	if document == nil {
		return Rect{}
	}
	return document.viewport
}

func (document *Document) ContentBounds() Rect {
	if document == nil {
		return Rect{}
	}
	return document.content
}

func (document *Document) MarkDirty() {
	if document == nil || document.host == nil {
		return
	}
	document.host.MarkDirty()
}

func (document *Document) MarkLayoutDirty() {
	if document == nil || document.host == nil {
		return
	}
	document.host.MarkLayoutDirty()
}

func (list FragmentDisplayList) Items() []FragmentDisplayItem {
	return list.items
}

func linkDocumentTree(parent *DocumentNode, node *DocumentNode) {
	if node == nil {
		return
	}
	node.Parent = parent
	for _, child := range node.Children {
		linkDocumentTree(node, child)
	}
}
