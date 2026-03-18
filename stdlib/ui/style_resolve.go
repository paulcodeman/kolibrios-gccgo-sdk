package ui

import "kos"

func resolveLength(value *int) (int, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveScrollbarWidth(value *int) (int, bool) {
	if value == nil {
		return 0, false
	}
	parsed := *value
	if parsed < 0 {
		parsed = 0
	}
	return parsed, true
}

func resolveScrollbarRadius(value *int) (int, bool) {
	if value == nil {
		return 0, false
	}
	parsed := *value
	if parsed < 0 {
		parsed = 0
	}
	return parsed, true
}

func resolveOpacity(value *uint8) (uint8, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveColor(value *kos.Color) (kos.Color, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveGradient(value *Gradient) (*Gradient, bool) {
	if value == nil {
		return nil, false
	}
	return value, true
}

func resolveShadow(value *Shadow) (*Shadow, bool) {
	if value == nil {
		return nil, false
	}
	return value, true
}

func resolveTextShadow(value *TextShadow) (*TextShadow, bool) {
	if value == nil {
		return nil, false
	}
	return value, true
}

func resolveFontPath(value *string) (string, bool) {
	if value == nil || *value == "" {
		return "", false
	}
	return *value, true
}

func resolveFontSize(value *int) (int, bool) {
	if value == nil || *value <= 0 {
		return 0, false
	}
	return *value, true
}

func resolveLineHeight(value *int) (int, bool) {
	if value == nil || *value <= 0 {
		return 0, false
	}
	return *value, true
}

func resolveTextAlign(value *TextAlign) (TextAlign, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveTextDecoration(value *TextDecoration) (TextDecoration, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveDisplay(value *DisplayMode) (DisplayMode, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveVisibility(value *VisibilityMode) (VisibilityMode, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveOverflow(value *OverflowMode) (OverflowMode, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveBackgroundAttachment(value *BackgroundAttachment) (BackgroundAttachment, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolvePosition(value *PositionMode) (PositionMode, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveBoxSizing(value *BoxSizing) (BoxSizing, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

func resolveSpacing(value *Spacing) (*Spacing, bool) {
	if value == nil {
		return nil, false
	}
	return value, true
}

func resolveCornerRadii(value *CornerRadii) (*CornerRadii, bool) {
	if value == nil {
		return nil, false
	}
	return value, true
}
