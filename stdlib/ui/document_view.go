package ui

import "kos"

type DocumentEvent struct {
	Type      EventType
	X         int
	Y         int
	LocalX    int
	LocalY    int
	DocumentX int
	DocumentY int
	DeltaX    int
	DeltaY    int
	Button    MouseButton
	Key       kos.KeyEvent
	ScrollX   int
	ScrollY   int
	View      *DocumentView
	Node      *DocumentNode
}

type DocumentView struct {
	Document    *Document
	Style       Style
	StyleFocus  Style
	StyleHover  Style
	StyleActive Style
	OnClick     interface{}

	window              *Window
	layoutRect          Rect
	visualRect          Rect
	visualRectValid     bool
	layoutKey           documentViewLayoutKey
	flowX               int
	flowY               int
	flowSet             bool
	dirty               bool
	layoutDirty         bool
	hovered             bool
	active              bool
	focused             bool
	scrollY             int
	drawnScrollY        int
	scrollMaxY          int
	scrollDrag          bool
	scrollDragOff       int
	hoverNode           *DocumentNode
	activeNode          *DocumentNode
	focusNode           *DocumentNode
	layerCanvas         *Canvas
	layerTiles          []*Canvas
	layerValid          bool
	layerWidth          int
	layerHeight         int
	layerDirty          Rect
	layerDirtySet       bool
	layerOffsetX        int
	layerOffsetY        int
	layerTileCols       int
	layerTileRows       int
	layerVisualKey      styleVisualKey
	effectiveStyleCache Style
	effectiveStyleValid bool
	hostStateCache      documentViewHostStateCache
	focusablesCache     []*DocumentNode
	focusablesValid     bool
	renderVisitGen      uint32
	layoutVisitGen      uint32
	dirtyQueueGen       uint32
}

type documentViewHostStateKey struct {
	rect          Rect
	insets        Spacing
	overflowY     OverflowMode
	scrollbar     scrollbarStyle
	contentHeight int
	scrollMaxY    int
	lineStep      int
}

type documentViewHostState struct {
	viewport         Rect
	baseContent      Rect
	scrollbarVisible bool
	scrollbarTrack   Rect
	scrollbar        scrollbarStyle
	contentHeight    int
	scrollMaxY       int
	lineStep         int
	pageStep         int
}

type documentViewHostStateCache struct {
	key   documentViewHostStateKey
	state documentViewHostState
	valid bool
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
	style := Style{}
	style.SetDisplay(DisplayBlock)
	return style
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
	view.invalidateEffectiveStyleCache()
	view.invalidateVisualBoundsCache()
	view.layerValid = false
	view.clearRetainedLayerDirty()
	view.invalidateDocumentViewCaches()
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
	view.hoverNode = nil
	view.activeNode = nil
	view.focusNode = nil
	view.scrollY = 0
	view.drawnScrollY = 0
	view.scrollMaxY = 0
	view.invalidateEffectiveStyleCache()
	view.invalidateVisualBoundsCache()
	view.layerValid = false
	view.clearRetainedLayerDirty()
	view.invalidateDocumentViewCaches()
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
	if view == nil {
		return Style{}
	}
	if view.effectiveStyleValid {
		return view.effectiveStyleCache
	}
	style := view.Style
	if view.focused && view.focusNode == nil && !view.StyleFocus.IsZero() {
		style = mergeStyle(style, view.StyleFocus)
	}
	if view.active && !view.StyleActive.IsZero() {
		style = mergeStyle(style, view.StyleActive)
	} else if view.hovered && !view.StyleHover.IsZero() {
		style = mergeStyle(style, view.StyleHover)
	}
	view.effectiveStyleCache = style
	view.effectiveStyleValid = true
	return style
}

func (view *DocumentView) SetHover(hover bool) bool {
	if view == nil || view.hovered == hover {
		return false
	}
	view.hovered = hover
	view.invalidateEffectiveStyleCache()
	changed := false
	if !hover {
		changed = view.clearHoverNode()
	}
	if view.StyleHover.IsZero() {
		return changed
	}
	if view.markDirtyForStyle(view.StyleHover) {
		changed = true
	}
	return changed
}

func (view *DocumentView) SetActive(active bool) bool {
	if view == nil || view.active == active {
		return false
	}
	view.active = active
	view.invalidateEffectiveStyleCache()
	changed := false
	if !active {
		view.scrollDrag = false
		changed = view.setActiveNode(nil, DocumentEvent{Type: EventMouseUp, View: view})
	}
	if view.StyleActive.IsZero() {
		return changed
	}
	if view.markDirtyForStyle(view.StyleActive) {
		changed = true
	}
	return changed
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
	view.invalidateVisualBoundsCache()
	view.layerValid = false
	view.clearRetainedLayerDirty()
	if view.window != nil {
		view.window.noteDirty(view)
	}
}

func (view *DocumentView) MarkDirtyRect(rect Rect) {
	view.MarkDirty()
}

func (view *DocumentView) MarkLayoutDirty() {
	if view == nil {
		return
	}
	view.layoutDirty = true
	view.invalidateVisualBoundsCache()
	view.layerValid = false
	view.clearRetainedLayerDirty()
	view.invalidateDocumentViewCaches()
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
	if view.visualRectValid {
		return view.visualRect
	}
	return view.layoutRect
}

func (view *DocumentView) invalidateEffectiveStyleCache() {
	if view == nil {
		return
	}
	view.effectiveStyleCache = Style{}
	view.effectiveStyleValid = false
	view.hostStateCache = documentViewHostStateCache{}
}

func (view *DocumentView) invalidateVisualBoundsCache() {
	if view == nil {
		return
	}
	view.visualRect = Rect{}
	view.visualRectValid = false
	view.hostStateCache = documentViewHostStateCache{}
}

func (view *DocumentView) invalidateDocumentViewCaches() {
	if view == nil {
		return
	}
	view.hostStateCache = documentViewHostStateCache{}
	view.focusablesCache = nil
	view.focusablesValid = false
}

func (view *DocumentView) Handle(event Event) bool {
	if view == nil || event.Type != EventClick {
		return false
	}
	documentEvent, ok := view.documentEventFor(EventClick, event.X, event.Y, event.Button)
	if !ok {
		documentEvent = DocumentEvent{
			Type:    EventClick,
			X:       event.X,
			Y:       event.Y,
			Button:  event.Button,
			ScrollY: view.scrollY,
			View:    view,
		}
	}
	if ok && dispatchDocumentClick(documentEvent.Node, documentEvent) {
		return true
	}
	return dispatchDocumentViewClick(view, documentEvent)
}

func dispatchDocumentClick(node *DocumentNode, event DocumentEvent) bool {
	if node == nil {
		return false
	}
	return dispatchDocumentNodeHandler(node.OnClick, node, event)
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
