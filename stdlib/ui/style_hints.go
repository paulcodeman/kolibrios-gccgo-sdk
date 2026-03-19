package ui

import "strings"

func containString(value ContainMode) string {
	switch value {
	case ContainNone:
		return "none"
	case ContainLayout:
		return "layout"
	case ContainPaint:
		return "paint"
	case ContainContent:
		return "content"
	default:
		return ""
	}
}

func ParseContain(value string) (ContainMode, bool) {
	normalized := normalizeCSSKeyword(value)
	switch normalized {
	case "none":
		return ContainNone, true
	case "content", "strict":
		return ContainContent, true
	}
	if normalized == "" {
		return 0, false
	}
	var parsed ContainMode
	for _, token := range strings.Fields(strings.ReplaceAll(normalized, ",", " ")) {
		switch token {
		case "layout":
			parsed |= ContainLayout
		case "paint":
			parsed |= ContainPaint
		default:
			return 0, false
		}
	}
	return parsed, parsed != ContainNone
}

func willChangeString(value WillChangeHints) string {
	if value == WillChangeAuto {
		return "auto"
	}
	parts := make([]string, 0, 4)
	if value&WillChangeContents != 0 {
		parts = append(parts, "contents")
	}
	if value&WillChangeScrollPosition != 0 {
		parts = append(parts, "scroll-position")
	}
	if value&WillChangeTransform != 0 {
		parts = append(parts, "transform")
	}
	if value&WillChangeOpacity != 0 {
		parts = append(parts, "opacity")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func ParseWillChange(value string) (WillChangeHints, bool) {
	normalized := normalizeCSSKeyword(value)
	if normalized == "auto" {
		return WillChangeAuto, true
	}
	if normalized == "" {
		return 0, false
	}
	var parsed WillChangeHints
	for _, token := range strings.Fields(strings.ReplaceAll(normalized, ",", " ")) {
		switch token {
		case "contents":
			parsed |= WillChangeContents
		case "scroll-position":
			parsed |= WillChangeScrollPosition
		case "transform":
			parsed |= WillChangeTransform
		case "opacity":
			parsed |= WillChangeOpacity
		default:
			return 0, false
		}
	}
	return parsed, parsed != WillChangeAuto
}

func (style *Style) SetContainString(value string) bool {
	parsed, ok := ParseContain(value)
	if !ok {
		return false
	}
	style.SetContain(parsed)
	return true
}

func (style Style) GetContainString() (string, bool) {
	value, ok := style.GetContain()
	if !ok {
		return "", false
	}
	return containString(value), true
}

func (style *Style) SetWillChangeString(value string) bool {
	parsed, ok := ParseWillChange(value)
	if !ok {
		return false
	}
	style.SetWillChange(parsed)
	return true
}

func (style Style) GetWillChangeString() (string, bool) {
	value, ok := style.GetWillChange()
	if !ok {
		return "", false
	}
	return willChangeString(value), true
}

func containForStyle(style Style) ContainMode {
	if value, ok := resolveContain(style.contain); ok {
		return value
	}
	return ContainNone
}

func willChangeForStyle(style Style) WillChangeHints {
	if value, ok := resolveWillChange(style.willChange); ok {
		return value
	}
	return WillChangeAuto
}

func containIncludesLayout(value ContainMode) bool {
	return value&ContainLayout != 0
}

func containIncludesPaint(value ContainMode) bool {
	return value&ContainPaint != 0
}

func styleContainsPaint(style Style) bool {
	return containIncludesPaint(containForStyle(style))
}

func willChangePromotesRetainedLayer(value WillChangeHints) bool {
	return value&(WillChangeContents|WillChangeScrollPosition|WillChangeTransform|WillChangeOpacity) != 0
}

func styleWillChangePromotesRetainedLayer(style Style) bool {
	return willChangePromotesRetainedLayer(willChangeForStyle(style))
}
