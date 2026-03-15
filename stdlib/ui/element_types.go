package ui

import "kos"

type Element struct {
	kind             ElementKind
	ID               kos.ButtonID
	Text             string
	Label            string
	Parent           *Element
	window           *Window
	Children         []Node
	Style            Style
	StyleHover       Style
	StyleActive      Style
	OnClick          interface{}
	tinygl           *tinyGLState
	layoutRect       Rect
	layoutMargin     Spacing
	layoutMarginSet  bool
	layoutPosition   PositionMode
	layoutHidden     bool
	dirty            bool
	renderKey        elementRenderKey
	wrapCache        textWrapCache
	preserveCache    textPreserveCache
	visualRect       Rect
	subtreeRect      Rect
	layoutKey        elementLayoutKey
	flowX            int
	flowY            int
	flowSet          bool
	hovered          bool
	active           bool
	focused          bool
	caret            int
	selectAnchor     int
	scrollX          int
	scrollY          int
	desiredCol       int
	dragMode         textDragMode
	dragScrollOffset int
	dragMoved        bool
	cache            *elementCache
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
	display *DisplayMode
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
