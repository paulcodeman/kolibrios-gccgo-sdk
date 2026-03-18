package ui

type DocumentEvent struct {
	Type   EventType
	X      int
	Y      int
	Button MouseButton
	View   *DocumentView
	Node   *DocumentNode
}

type DocumentView struct {
	Document    *Document
	Style       Style
	StyleHover  Style
	StyleActive Style
	OnClick     interface{}

	window         *Window
	layoutRect     Rect
	visualRect     Rect
	layoutKey      documentViewLayoutKey
	flowX          int
	flowY          int
	flowSet        bool
	dirty          bool
	layoutDirty    bool
	hovered        bool
	active         bool
	renderVisitGen uint32
	layoutVisitGen uint32
	dirtyQueueGen  uint32
}

type documentViewLayoutKey struct {
	position    *PositionMode
	display     *DisplayMode
	containerX  int
	containerY  int
	containerW  int
	containerH  int
	left        *int
	top         *int
	right       *int
	bottom      *int
	styleWidth  *int
	styleHeight *int
	margin      *Spacing
	flowSet     bool
	flowX       int
	flowY       int
}

func DefaultDocumentViewStyle() Style {
	return Style{
		Display: DisplayPtr(DisplayBlock),
	}
}

func CreateDocumentView(document *Document) *DocumentView {
	view := &DocumentView{
		Style: DefaultDocumentViewStyle(),
	}
	view.setDocument(document)
	return view
}

func (view *DocumentView) setWindow(window *Window) {
	if view == nil {
		return
	}
	view.window = window
	view.renderVisitGen = 0
	view.layoutVisitGen = 0
	view.dirtyQueueGen = 0
}

func (view *DocumentView) setDocument(document *Document) {
	if view == nil {
		return
	}
	if view.Document == document {
		if document != nil && document.host != view {
			document.host = view
		}
		return
	}
	if view.Document != nil && view.Document.host == view {
		view.Document.host = nil
	}
	view.Document = document
	if document != nil {
		document.host = view
	}
}

func (view *DocumentView) SetDocument(document *Document) bool {
	if view == nil || view.Document == document {
		return false
	}
	view.setDocument(document)
	view.MarkLayoutDirty()
	return true
}

func (view *DocumentView) effectiveStyle() Style {
	style := view.Style
	if view.active && !view.StyleActive.IsZero() {
		style = mergeStyle(style, view.StyleActive)
	} else if view.hovered && !view.StyleHover.IsZero() {
		style = mergeStyle(style, view.StyleHover)
	}
	return style
}

func (view *DocumentView) SetHover(hover bool) bool {
	if view == nil || view.hovered == hover {
		return false
	}
	view.hovered = hover
	if view.StyleHover.IsZero() {
		return false
	}
	return view.markDirtyForStyle(view.StyleHover)
}

func (view *DocumentView) SetActive(active bool) bool {
	if view == nil || view.active == active {
		return false
	}
	view.active = active
	if view.StyleActive.IsZero() {
		return false
	}
	return view.markDirtyForStyle(view.StyleActive)
}

func (view *DocumentView) markDirtyForStyle(style Style) bool {
	if view == nil {
		return false
	}
	if style.HasLayout() {
		view.MarkLayoutDirty()
		return true
	}
	if style.HasVisual() {
		view.MarkDirty()
		return true
	}
	return false
}

func (view *DocumentView) MarkDirty() {
	if view == nil {
		return
	}
	view.dirty = true
	if view.window != nil {
		view.window.noteDirty(view)
	}
}

func (view *DocumentView) MarkLayoutDirty() {
	if view == nil {
		return
	}
	view.layoutDirty = true
	view.MarkDirty()
	if view.window != nil {
		view.window.layoutDirty = true
		view.window.renderListValid = false
	}
}

func (view *DocumentView) Dirty() bool {
	if view == nil {
		return false
	}
	return view.dirty || view.layoutDirty
}

func (view *DocumentView) ClearDirty() {
	if view == nil {
		return
	}
	view.dirty = false
}

func (view *DocumentView) LayoutDirty() bool {
	if view == nil {
		return false
	}
	return view.layoutDirty || !documentViewLayoutKeyEqual(view.layoutKeyFor(view.effectiveStyle(), view.layoutContainer()), view.layoutKey)
}

func (view *DocumentView) Bounds() Rect {
	if view == nil {
		return Rect{}
	}
	return view.layoutRect
}

func (view *DocumentView) VisualBounds() Rect {
	if view == nil {
		return Rect{}
	}
	if !view.visualRect.Empty() {
		return view.visualRect
	}
	return view.layoutRect
}

func (view *DocumentView) Handle(event Event) bool {
	if view == nil || event.Type != EventClick {
		return false
	}
	documentEvent := DocumentEvent{
		Type:   event.Type,
		X:      event.X,
		Y:      event.Y,
		Button: event.Button,
		View:   view,
	}
	content := contentRectFor(view.layoutRect, view.effectiveStyle())
	if view.Document != nil && content.Contains(event.X, event.Y) {
		documentEvent.Node = view.Document.HitTest(event.X, event.Y)
		if dispatchDocumentClick(documentEvent.Node, documentEvent) {
			return true
		}
	}
	return dispatchDocumentViewClick(view, documentEvent)
}

func dispatchDocumentClick(node *DocumentNode, event DocumentEvent) bool {
	if node == nil || node.OnClick == nil {
		return false
	}
	switch handler := node.OnClick.(type) {
	case func():
		handler()
		return true
	case func(*DocumentNode):
		handler(node)
		return true
	case func(DocumentEvent):
		handler(event)
		return true
	default:
		return false
	}
}

func dispatchDocumentViewClick(view *DocumentView, event DocumentEvent) bool {
	if view == nil || view.OnClick == nil {
		return false
	}
	switch handler := view.OnClick.(type) {
	case func():
		handler()
		return true
	case func(DocumentEvent):
		handler(event)
		return true
	default:
		return false
	}
}

func (view *DocumentView) layoutContainer() Rect {
	if view == nil {
		return Rect{}
	}
	key := view.layoutKey
	return Rect{
		X:      key.containerX,
		Y:      key.containerY,
		Width:  key.containerW,
		Height: key.containerH,
	}
}

func documentViewLayoutKeyEqual(a documentViewLayoutKey, b documentViewLayoutKey) bool {
	return equalPositionPtr(a.position, b.position) &&
		equalDisplayPtr(a.display, b.display) &&
		a.containerX == b.containerX &&
		a.containerY == b.containerY &&
		a.containerW == b.containerW &&
		a.containerH == b.containerH &&
		equalIntPtr(a.left, b.left) &&
		equalIntPtr(a.top, b.top) &&
		equalIntPtr(a.right, b.right) &&
		equalIntPtr(a.bottom, b.bottom) &&
		equalIntPtr(a.styleWidth, b.styleWidth) &&
		equalIntPtr(a.styleHeight, b.styleHeight) &&
		equalSpacingPtr(a.margin, b.margin) &&
		a.flowSet == b.flowSet &&
		a.flowX == b.flowX &&
		a.flowY == b.flowY
}
