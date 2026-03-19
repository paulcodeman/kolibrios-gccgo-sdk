package ui

import "kos"

type Element struct {
	kind                   ElementKind
	spec                   *ElementSpec
	ID                     kos.ButtonID
	Text                   string
	Label                  string
	OnEvent                interface{}
	OnEventCapture         interface{}
	OnChange               interface{}
	OnInput                interface{}
	OnPointerDown          interface{}
	OnPointerUp            interface{}
	OnPointerMove          interface{}
	OnPointerEnter         interface{}
	OnPointerLeave         interface{}
	OnMouseDown            interface{}
	OnMouseUp              interface{}
	OnMouseMove            interface{}
	OnMouseEnter           interface{}
	OnMouseLeave           interface{}
	OnScroll               interface{}
	OnFocus                interface{}
	OnBlur                 interface{}
	OnFocusIn              interface{}
	OnFocusOut             interface{}
	OnKeyDown              interface{}
	Parent                 *Element
	window                 *Window
	Children               []Node
	Style                  Style
	StyleHover             Style
	StyleActive            Style
	StyleFocus             Style
	OnClick                interface{}
	tinygl                 *tinyGLState
	layoutRect             Rect
	layoutMargin           Spacing
	layoutMarginSet        bool
	layoutPosition         PositionMode
	layoutHidden           bool
	dirty                  bool
	renderKey              elementRenderKey
	wrapCache              textWrapCache
	preserveCache          textPreserveCache
	textInputLayoutCache   textInputLayoutCache
	effectiveStyleCache    Style
	effectiveStyleValid    bool
	visualRect             Rect
	visualRectValid        bool
	subtreeRect            Rect
	subtreeRectValid       bool
	layoutKey              elementLayoutKey
	flowX                  int
	flowY                  int
	flowSet                bool
	hovered                bool
	active                 bool
	focused                bool
	checked                bool
	controlGroup           string
	caret                  int
	selectAnchor           int
	scrollX                int
	scrollY                int
	value                  int
	minValue               int
	maxValue               int
	stepValue              int
	rangeDragActive        bool
	desiredCol             int
	dragMode               textDragMode
	dragScrollOffset       int
	dragMoved              bool
	cache                  *elementCache
	subtreeLayer           *Canvas
	subtreeLayerTiles      []*Canvas
	subtreeLayerValid      bool
	subtreeLayerWidth      int
	subtreeLayerHeight     int
	subtreeLayerTileCols   int
	subtreeLayerTileRows   int
	subtreeLayerDirty      [elementRetainedLayerMaxDirtyRects]Rect
	subtreeLayerDirtyCount int
	subtreeLayerDirtyFull  bool
	subtreeLayerTreeKnown  bool
	subtreeLayerTreeOK     bool
	subtreeLayerTreeCount  int
	renderVisitGen         uint32
	layoutVisitGen         uint32
	dirtyQueueGen          uint32
}

type textDragMode uint8

const (
	textDragNone textDragMode = iota
	textDragSelect
	textDragScroll
)

type ElementKind uint8

const (
	ElementKindUnknown ElementKind = iota
	ElementKindButton
	ElementKindLabel
	ElementKindInput
	ElementKindTextarea
	ElementKindTinyGL
	ElementKindBox
	ElementKindCheckbox
	ElementKindRadio
	ElementKindProgress
	ElementKindRange
)

func (kind ElementKind) String() string {
	switch kind {
	case ElementKindButton:
		return "button"
	case ElementKindLabel:
		return "label"
	case ElementKindInput:
		return "input"
	case ElementKindTextarea:
		return "textarea"
	case ElementKindTinyGL:
		return "tinygl"
	case ElementKindBox:
		return "box"
	case ElementKindCheckbox:
		return "checkbox"
	case ElementKindRadio:
		return "radio"
	case ElementKindProgress:
		return "progress"
	case ElementKindRange:
		return "range"
	default:
		return "unknown"
	}
}

type elementCache struct {
	canvas    *Canvas
	width     int
	height    int
	offsetX   int
	offsetY   int
	alpha     bool
	renderKey elementRenderKey
}

type elementRenderKey struct {
	kind    ElementKind
	text    string
	checked bool
	value   int
	min     int
	max     int
	display *DisplayMode
	focus   bool
	visual  styleVisualKey
}

type elementLayoutKey struct {
	kind        ElementKind
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
	width       int
	height      int
	styleWidth  *int
	styleHeight *int
	margin      *Spacing
	flowSet     bool
	flowX       int
	flowY       int
}
