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
	Kind           DocumentNodeKind
	Name           string
	Text           string
	Style          Style
	StyleHover     Style
	StyleActive    Style
	StyleFocus     Style
	Focusable      bool
	OnEvent        interface{}
	OnEventCapture interface{}
	OnClick        interface{}
	OnChange       interface{}
	OnInput        interface{}
	OnMouseDown    interface{}
	OnMouseUp      interface{}
	OnMouseMove    interface{}
	OnMouseEnter   interface{}
	OnMouseLeave   interface{}
	OnFocus        interface{}
	OnBlur         interface{}
	OnScroll       interface{}
	OnKeyDown      interface{}
	OnFocusIn      interface{}
	OnFocusOut     interface{}
	Parent         *DocumentNode
	Children       []*DocumentNode

	hovered   bool
	active    bool
	focused   bool
	wrapCache textWrapCache
}

type Fragment struct {
	Kind        FragmentKind
	Node        *DocumentNode
	Style       Style
	PaintStyle  Style
	Bounds      Rect
	PaintBounds Rect
	Content     Rect
	Text        string
	Children    []*Fragment

	font       *ttfFont
	metrics    fontMetrics
	lineHeight int
	lines      []textLine
	linesOwned bool
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

	rootFragment    *Fragment
	displayList     FragmentDisplayList
	viewport        Rect
	content         Rect
	host            *DocumentView
	fragmentByNode  map[*DocumentNode]*Fragment
	hitGrid         fragmentHitTestGrid
	hitGridValid    bool
	displayVersion  uint32
	geometryVersion uint32
	hitGridVersion  uint32
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
			clearDocumentNodeCaches(child)
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
	clearDocumentNodeCaches(document.Root)
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

func (document *Document) FragmentForNode(node *DocumentNode) *Fragment {
	if document == nil || node == nil {
		return nil
	}
	if document.fragmentByNode != nil {
		if fragment, ok := document.fragmentByNode[node]; ok {
			return fragment
		}
	}
	return findDocumentFragment(document.rootFragment, node)
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

func (document *Document) MarkNodeDirty(node *DocumentNode) {
	if document == nil || document.host == nil || node == nil {
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

func (node *DocumentNode) clearTextCache() {
	if node == nil {
		return
	}
	releaseTextLines(node.wrapCache.lines)
	node.wrapCache = textWrapCache{}
}

func clearDocumentNodeCaches(node *DocumentNode) {
	if node == nil {
		return
	}
	node.clearTextCache()
	for _, child := range node.Children {
		clearDocumentNodeCaches(child)
	}
}

func findDocumentFragment(fragment *Fragment, node *DocumentNode) *Fragment {
	if fragment == nil || node == nil {
		return nil
	}
	if fragment.Node == node {
		return fragment
	}
	for _, child := range fragment.Children {
		if match := findDocumentFragment(child, node); match != nil {
			return match
		}
	}
	return nil
}
