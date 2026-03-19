package ui

import "strings"

type ElementSpecFlags uint32

const (
	ElementSpecFocusable ElementSpecFlags = 1 << iota
	ElementSpecClickable
	ElementSpecTextInput
	ElementSpecMultiline
	ElementSpecContainer
	ElementSpecTinyGL
	ElementSpecButtonLike
)

type ElementSpec struct {
	Kind         ElementKind
	Name         string
	Flags        ElementSpecFlags
	DefaultStyle func() Style
	HoverStyle   func() Style
	ActiveStyle  func() Style
	FocusStyle   func() Style
}

func (spec *ElementSpec) hasFlag(flag ElementSpecFlags) bool {
	return spec != nil && spec.Flags&flag != 0
}

func (spec *ElementSpec) defaultBaseStyle() Style {
	if spec == nil || spec.DefaultStyle == nil {
		return Style{}
	}
	return spec.DefaultStyle()
}

func (spec *ElementSpec) defaultHoverStyle() Style {
	if spec == nil || spec.HoverStyle == nil {
		return Style{}
	}
	return spec.HoverStyle()
}

func (spec *ElementSpec) defaultActiveStyle() Style {
	if spec == nil || spec.ActiveStyle == nil {
		return Style{}
	}
	return spec.ActiveStyle()
}

func (spec *ElementSpec) defaultFocusStyle() Style {
	if spec == nil || spec.FocusStyle == nil {
		return Style{}
	}
	return spec.FocusStyle()
}

var (
	SpecButton = &ElementSpec{
		Kind:         ElementKindButton,
		Name:         "button",
		Flags:        ElementSpecFocusable | ElementSpecClickable | ElementSpecButtonLike,
		DefaultStyle: DefaultButtonStyle,
		HoverStyle:   DefaultButtonHoverStyle,
		ActiveStyle:  DefaultButtonActiveStyle,
	}
	SpecLabel = &ElementSpec{
		Kind:         ElementKindLabel,
		Name:         "label",
		DefaultStyle: DefaultLabelStyle,
	}
	SpecInput = &ElementSpec{
		Kind:         ElementKindInput,
		Name:         "input",
		Flags:        ElementSpecFocusable | ElementSpecTextInput,
		DefaultStyle: DefaultInputStyle,
	}
	SpecTextarea = &ElementSpec{
		Kind:         ElementKindTextarea,
		Name:         "textarea",
		Flags:        ElementSpecFocusable | ElementSpecTextInput | ElementSpecMultiline,
		DefaultStyle: DefaultTextareaStyle,
	}
	SpecTinyGL = &ElementSpec{
		Kind:         ElementKindTinyGL,
		Name:         "tinygl",
		Flags:        ElementSpecTinyGL,
		DefaultStyle: DefaultTinyGLStyle,
	}
	SpecBox = &ElementSpec{
		Kind:         ElementKindBox,
		Name:         "box",
		Flags:        ElementSpecContainer,
		DefaultStyle: DefaultBoxStyle,
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
	elementSpecsByKind[spec.Kind] = spec
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
