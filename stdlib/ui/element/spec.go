package ui

import (
	"kos"
	"strings"
)

type ElementSpecFlags uint32

const (
	ElementSpecFocusable ElementSpecFlags = 1 << iota
	ElementSpecClickable
	ElementSpecTextInput
	ElementSpecMultiline
	ElementSpecContainer
	ElementSpecTinyGL
	ElementSpecButtonLike
	ElementSpecCheckable
	ElementSpecRadio
	ElementSpecProgress
	ElementSpecRange
)

type ElementMeasureContext struct {
	Layout     LayoutContext
	Container  Rect
	Style      Style
	Text       string
	Insets     Spacing
	Font       *ttfFont
	CharWidth  int
	LineHeight int
	TextWidth  int
	TextHeight int
}

type ElementSpecInitFunc func(*Element)
type ElementSpecMeasureFunc func(*Element, ElementMeasureContext, int) int
type ElementSpecPaintFunc func(*Element, *Canvas, Rect, Style) bool
type ElementSpecRawDrawFunc func(*Element, Style) bool
type ElementSpecClickFunc func(*Element, *Event) bool
type ElementSpecMouseMoveFunc func(*Element, int, int, PointerButtons) bool
type ElementSpecMouseButtonFunc func(*Element, int, int, MouseButton, PointerButtons) bool
type ElementSpecKeyFunc func(*Element, kos.KeyEvent) bool
type ElementSpecScrollFunc func(*Element, int, int) bool

type ElementSpec struct {
	Base            *ElementSpec
	Kind            ElementKind
	Name            string
	Flags           ElementSpecFlags
	DefaultStyle    func() Style
	HoverStyle      func() Style
	ActiveStyle     func() Style
	FocusStyle      func() Style
	Init            ElementSpecInitFunc
	MeasureWidth    ElementSpecMeasureFunc
	MeasureHeight   ElementSpecMeasureFunc
	Paint           ElementSpecPaintFunc
	DrawRaw         ElementSpecRawDrawFunc
	HandleClick     ElementSpecClickFunc
	HandleMouseMove ElementSpecMouseMoveFunc
	HandleMouseDown ElementSpecMouseButtonFunc
	HandleMouseUp   ElementSpecMouseButtonFunc
	HandleKey       ElementSpecKeyFunc
	HandleScroll    ElementSpecScrollFunc
}

func (spec *ElementSpec) hasFlag(flag ElementSpecFlags) bool {
	if spec == nil {
		return false
	}
	if spec.Flags&flag != 0 {
		return true
	}
	return spec.Base.hasFlag(flag)
}

func (spec *ElementSpec) defaultBaseStyle() Style {
	if spec == nil {
		return Style{}
	}
	style := spec.Base.defaultBaseStyle()
	if spec.DefaultStyle != nil {
		style = mergeStyle(style, spec.DefaultStyle())
	}
	return style
}

func (spec *ElementSpec) defaultHoverStyle() Style {
	if spec == nil {
		return Style{}
	}
	style := spec.Base.defaultHoverStyle()
	if spec.HoverStyle != nil {
		style = mergeStyle(style, spec.HoverStyle())
	}
	return style
}

func (spec *ElementSpec) defaultActiveStyle() Style {
	if spec == nil {
		return Style{}
	}
	style := spec.Base.defaultActiveStyle()
	if spec.ActiveStyle != nil {
		style = mergeStyle(style, spec.ActiveStyle())
	}
	return style
}

func (spec *ElementSpec) defaultFocusStyle() Style {
	if spec == nil {
		return Style{}
	}
	style := spec.Base.defaultFocusStyle()
	if spec.FocusStyle != nil {
		style = mergeStyle(style, spec.FocusStyle())
	}
	return style
}

func (spec *ElementSpec) initFunc() ElementSpecInitFunc {
	if spec == nil {
		return nil
	}
	if spec.Init != nil {
		return spec.Init
	}
	return spec.Base.initFunc()
}

func (spec *ElementSpec) measureWidthFunc() ElementSpecMeasureFunc {
	if spec == nil {
		return nil
	}
	if spec.MeasureWidth != nil {
		return spec.MeasureWidth
	}
	return spec.Base.measureWidthFunc()
}

func (spec *ElementSpec) measureHeightFunc() ElementSpecMeasureFunc {
	if spec == nil {
		return nil
	}
	if spec.MeasureHeight != nil {
		return spec.MeasureHeight
	}
	return spec.Base.measureHeightFunc()
}

func (spec *ElementSpec) paintFunc() ElementSpecPaintFunc {
	if spec == nil {
		return nil
	}
	if spec.Paint != nil {
		return spec.Paint
	}
	return spec.Base.paintFunc()
}

func (spec *ElementSpec) drawRawFunc() ElementSpecRawDrawFunc {
	if spec == nil {
		return nil
	}
	if spec.DrawRaw != nil {
		return spec.DrawRaw
	}
	return spec.Base.drawRawFunc()
}

func (spec *ElementSpec) handleClickFunc() ElementSpecClickFunc {
	if spec == nil {
		return nil
	}
	if spec.HandleClick != nil {
		return spec.HandleClick
	}
	return spec.Base.handleClickFunc()
}

func (spec *ElementSpec) handleMouseMoveFunc() ElementSpecMouseMoveFunc {
	if spec == nil {
		return nil
	}
	if spec.HandleMouseMove != nil {
		return spec.HandleMouseMove
	}
	return spec.Base.handleMouseMoveFunc()
}

func (spec *ElementSpec) handleMouseDownFunc() ElementSpecMouseButtonFunc {
	if spec == nil {
		return nil
	}
	if spec.HandleMouseDown != nil {
		return spec.HandleMouseDown
	}
	return spec.Base.handleMouseDownFunc()
}

func (spec *ElementSpec) handleMouseUpFunc() ElementSpecMouseButtonFunc {
	if spec == nil {
		return nil
	}
	if spec.HandleMouseUp != nil {
		return spec.HandleMouseUp
	}
	return spec.Base.handleMouseUpFunc()
}

func (spec *ElementSpec) handleKeyFunc() ElementSpecKeyFunc {
	if spec == nil {
		return nil
	}
	if spec.HandleKey != nil {
		return spec.HandleKey
	}
	return spec.Base.handleKeyFunc()
}

func (spec *ElementSpec) handleScrollFunc() ElementSpecScrollFunc {
	if spec == nil {
		return nil
	}
	if spec.HandleScroll != nil {
		return spec.HandleScroll
	}
	return spec.Base.handleScrollFunc()
}

var (
	SpecButton = &ElementSpec{
		Kind:          ElementKindButton,
		Name:          "button",
		Flags:         ElementSpecFocusable | ElementSpecClickable | ElementSpecButtonLike,
		DefaultStyle:  DefaultButtonStyle,
		HoverStyle:    DefaultButtonHoverStyle,
		ActiveStyle:   DefaultButtonActiveStyle,
		MeasureWidth:  measureButtonWidth,
		MeasureHeight: measureButtonHeight,
		DrawRaw:       drawButtonRaw,
	}
	SpecLabel = &ElementSpec{
		Kind:         ElementKindLabel,
		Name:         "label",
		DefaultStyle: DefaultLabelStyle,
	}
	SpecInput = &ElementSpec{
		Kind:            ElementKindInput,
		Name:            "input",
		Flags:           ElementSpecFocusable | ElementSpecTextInput,
		DefaultStyle:    DefaultInputStyle,
		Paint:           paintTextInputElement,
		DrawRaw:         drawTextInputRaw,
		HandleClick:     handleTextInputClick,
		HandleMouseMove: handleTextInputMouseMove,
		HandleMouseDown: handleTextInputMouseDown,
		HandleMouseUp:   handleTextInputMouseUp,
		HandleKey:       handleTextInputKey,
		HandleScroll:    handleTextInputScroll,
	}
	SpecTextarea = &ElementSpec{
		Base:         SpecInput,
		Kind:         ElementKindTextarea,
		Name:         "textarea",
		Flags:        ElementSpecMultiline,
		DefaultStyle: DefaultTextareaStyle,
	}
	SpecTinyGL = &ElementSpec{
		Kind:         ElementKindTinyGL,
		Name:         "tinygl",
		Flags:        ElementSpecTinyGL,
		DefaultStyle: DefaultTinyGLStyle,
		Paint:        paintTinyGLElement,
		DrawRaw:      drawTinyGLRaw,
	}
	SpecBox = &ElementSpec{
		Kind:         ElementKindBox,
		Name:         "box",
		Flags:        ElementSpecContainer,
		DefaultStyle: DefaultBoxStyle,
	}
	SpecCheckbox = &ElementSpec{
		Kind:          ElementKindCheckbox,
		Name:          "checkbox",
		Flags:         ElementSpecFocusable | ElementSpecClickable | ElementSpecCheckable,
		DefaultStyle:  DefaultCheckboxStyle,
		HoverStyle:    DefaultCheckboxHoverStyle,
		ActiveStyle:   DefaultCheckboxActiveStyle,
		MeasureWidth:  measureCheckableWidth,
		MeasureHeight: measureCheckableHeight,
		Paint:         paintCheckboxElement,
		DrawRaw:       drawElementViaSurfaceRaw,
		HandleClick:   handleCheckableClick,
		HandleKey:     handleControlKeySpec,
	}
	SpecRadio = &ElementSpec{
		Base:  SpecCheckbox,
		Kind:  ElementKindRadio,
		Name:  "radio",
		Flags: ElementSpecRadio,
	}
	SpecProgress = &ElementSpec{
		Kind:          ElementKindProgress,
		Name:          "progress",
		Flags:         ElementSpecProgress,
		DefaultStyle:  DefaultProgressStyle,
		Init:          initRangedElement,
		MeasureWidth:  measureProgressWidth,
		MeasureHeight: measureProgressHeight,
		Paint:         paintProgressElement,
		DrawRaw:       drawElementViaSurfaceRaw,
	}
	SpecRange = &ElementSpec{
		Kind:            ElementKindRange,
		Name:            "range",
		Flags:           ElementSpecFocusable | ElementSpecClickable | ElementSpecRange,
		DefaultStyle:    DefaultRangeStyle,
		HoverStyle:      DefaultRangeHoverStyle,
		ActiveStyle:     DefaultRangeActiveStyle,
		Init:            initRangedElement,
		MeasureWidth:    measureRangeWidth,
		MeasureHeight:   measureRangeHeight,
		Paint:           paintRangeElement,
		DrawRaw:         drawElementViaSurfaceRaw,
		HandleClick:     handleRangeClick,
		HandleMouseMove: handleRangeMouseMoveSpec,
		HandleMouseDown: handleRangeMouseDownSpec,
		HandleMouseUp:   handleRangeMouseUpSpec,
		HandleKey:       handleControlKeySpec,
	}
)

var (
	elementSpecsByKind = map[ElementKind]*ElementSpec{}
	elementSpecsByName = map[string]*ElementSpec{}
)

func init() {
	RegisterElementSpec(SpecButton)
	RegisterElementSpec(SpecLabel)
	RegisterElementSpec(SpecInput)
	RegisterElementSpec(SpecTextarea)
	RegisterElementSpec(SpecTinyGL)
	RegisterElementSpec(SpecBox)
	RegisterElementSpec(SpecCheckbox)
	RegisterElementSpec(SpecRadio)
	RegisterElementSpec(SpecProgress)
	RegisterElementSpec(SpecRange)
}

func normalizeElementSpecName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// RegisterElementSpec installs or replaces an element specification in the
// global registry used by CreateElementFromSpec/CreateElementByName.
func RegisterElementSpec(spec *ElementSpec) {
	if spec == nil {
		return
	}
	if spec.Kind != ElementKindUnknown {
		elementSpecsByKind[spec.Kind] = spec
	}
	if name := normalizeElementSpecName(spec.Name); name != "" {
		elementSpecsByName[name] = spec
	}
}

func ElementSpecForKind(kind ElementKind) *ElementSpec {
	if spec, ok := elementSpecsByKind[kind]; ok {
		return spec
	}
	return nil
}

func ElementSpecForName(name string) *ElementSpec {
	if spec, ok := elementSpecsByName[normalizeElementSpecName(name)]; ok {
		return spec
	}
	return nil
}
