package ui

import "kos"

type styleVisualKey struct {
	background           *kos.Color
	foreground           *kos.Color
	borderColor          *kos.Color
	borderWidth          *int
	borderRadius         *CornerRadii
	gradient             *Gradient
	backgroundAttachment *BackgroundAttachment
	shadow               *Shadow
	textAlign            *TextAlign
	textShadow           *TextShadow
	fontPath             *string
	fontSize             *int
	padding              *Spacing
	opacity              *uint8
	overflow             *OverflowMode
	overflowX            *OverflowMode
	overflowY            *OverflowMode
	scrollbarWidth       *int
	scrollbarTrack       *kos.Color
	scrollbarThumb       *kos.Color
	scrollbarRadius      *int
	scrollbarPadding     *Spacing
}

func visualKeyFor(style Style) styleVisualKey {
	key := styleVisualKey{}
	if value, ok := resolveColor(style.background); ok {
		v := value
		key.background = &v
	}
	if value, ok := resolveColor(style.foreground); ok {
		v := value
		key.foreground = &v
	}
	if value, ok := resolveColor(style.borderColor); ok {
		v := value
		key.borderColor = &v
	}
	if value, ok := resolveLength(style.borderWidth); ok {
		v := value
		key.borderWidth = &v
	}
	if value, ok := resolveCornerRadii(style.borderRadius); ok {
		if value != nil {
			v := *value
			key.borderRadius = &v
		}
	}
	if value, ok := resolveGradient(style.gradient); ok {
		if value != nil {
			v := *value
			key.gradient = &v
		}
	}
	if value, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok {
		v := value
		key.backgroundAttachment = &v
	}
	if value, ok := resolveShadow(style.shadow); ok {
		if value != nil {
			v := *value
			key.shadow = &v
		}
	}
	if value, ok := resolveTextAlign(style.textAlign); ok {
		v := value
		key.textAlign = &v
	}
	if value, ok := resolveTextShadow(style.textShadow); ok {
		if value != nil {
			v := *value
			key.textShadow = &v
		}
	}
	if value, ok := resolveFontPath(style.fontPath); ok {
		v := value
		key.fontPath = &v
	}
	if value, ok := resolveFontSize(style.fontSize); ok {
		v := value
		key.fontSize = &v
	}
	if value, ok := resolveSpacing(style.padding); ok {
		if value != nil {
			v := *value
			key.padding = &v
		}
	}
	if value, ok := resolveOpacity(style.opacity); ok {
		v := value
		key.opacity = &v
	}
	if value, ok := resolveOverflow(style.overflow); ok {
		v := value
		key.overflow = &v
	}
	if value, ok := resolveOverflow(style.overflowX); ok {
		v := value
		key.overflowX = &v
	}
	if value, ok := resolveOverflow(style.overflowY); ok {
		v := value
		key.overflowY = &v
	}
	if value, ok := resolveScrollbarWidth(style.scrollbarWidth); ok {
		v := value
		key.scrollbarWidth = &v
	}
	if value, ok := resolveColor(style.scrollbarTrack); ok {
		v := value
		key.scrollbarTrack = &v
	}
	if value, ok := resolveColor(style.scrollbarThumb); ok {
		v := value
		key.scrollbarThumb = &v
	}
	if value, ok := resolveScrollbarRadius(style.scrollbarRadius); ok {
		v := value
		key.scrollbarRadius = &v
	}
	if value, ok := resolveSpacing(style.scrollbarPadding); ok {
		if value != nil {
			v := *value
			key.scrollbarPadding = &v
		}
	}
	return key
}

func styleVisualKeyEqual(a styleVisualKey, b styleVisualKey) bool {
	return equalColorPtr(a.background, b.background) &&
		equalColorPtr(a.foreground, b.foreground) &&
		equalColorPtr(a.borderColor, b.borderColor) &&
		equalIntPtr(a.borderWidth, b.borderWidth) &&
		equalCornerRadiiPtr(a.borderRadius, b.borderRadius) &&
		equalGradientPtr(a.gradient, b.gradient) &&
		equalBackgroundAttachmentPtr(a.backgroundAttachment, b.backgroundAttachment) &&
		equalShadowPtr(a.shadow, b.shadow) &&
		equalTextAlignPtr(a.textAlign, b.textAlign) &&
		equalTextShadowPtr(a.textShadow, b.textShadow) &&
		equalStringPtr(a.fontPath, b.fontPath) &&
		equalIntPtr(a.fontSize, b.fontSize) &&
		equalSpacingPtr(a.padding, b.padding) &&
		equalBytePtr(a.opacity, b.opacity) &&
		equalOverflowPtr(a.overflow, b.overflow) &&
		equalOverflowPtr(a.overflowX, b.overflowX) &&
		equalOverflowPtr(a.overflowY, b.overflowY) &&
		equalIntPtr(a.scrollbarWidth, b.scrollbarWidth) &&
		equalColorPtr(a.scrollbarTrack, b.scrollbarTrack) &&
		equalColorPtr(a.scrollbarThumb, b.scrollbarThumb) &&
		equalIntPtr(a.scrollbarRadius, b.scrollbarRadius) &&
		equalSpacingPtr(a.scrollbarPadding, b.scrollbarPadding)
}

func equalColorPtr(a *kos.Color, b *kos.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalIntPtr(a *int, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalBytePtr(a *uint8, b *uint8) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalTextAlignPtr(a *TextAlign, b *TextAlign) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalCornerRadiiPtr(a *CornerRadii, b *CornerRadii) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalGradientPtr(a *Gradient, b *Gradient) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalBackgroundAttachmentPtr(a *BackgroundAttachment, b *BackgroundAttachment) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalShadowPtr(a *Shadow, b *Shadow) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalTextShadowPtr(a *TextShadow, b *TextShadow) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalStringPtr(a *string, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalSpacingPtr(a *Spacing, b *Spacing) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalPositionPtr(a *PositionMode, b *PositionMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalDisplayPtr(a *DisplayMode, b *DisplayMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalOverflowPtr(a *OverflowMode, b *OverflowMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func mergeStyle(base Style, override Style) Style {
	if override.IsZero() {
		return base
	}
	style := base
	if override.background != nil {
		style.background = override.background
	}
	if override.foreground != nil {
		style.foreground = override.foreground
	}
	if override.borderColor != nil {
		style.borderColor = override.borderColor
	}
	if override.borderWidth != nil {
		style.borderWidth = override.borderWidth
	}
	if override.borderRadius != nil {
		style.borderRadius = override.borderRadius
	}
	if override.gradient != nil {
		style.gradient = override.gradient
	}
	if override.backgroundAttachment != nil {
		style.backgroundAttachment = override.backgroundAttachment
	}
	if override.shadow != nil {
		style.shadow = override.shadow
	}
	if override.display != nil {
		style.display = override.display
	}
	if override.textAlign != nil {
		style.textAlign = override.textAlign
	}
	if override.textShadow != nil {
		style.textShadow = override.textShadow
	}
	if override.fontPath != nil {
		style.fontPath = override.fontPath
	}
	if override.fontSize != nil {
		style.fontSize = override.fontSize
	}
	if override.padding != nil {
		style.padding = override.padding
	}
	if override.opacity != nil {
		style.opacity = override.opacity
	}
	if override.position != nil {
		style.position = override.position
	}
	if override.left != nil {
		style.left = override.left
	}
	if override.top != nil {
		style.top = override.top
	}
	if override.right != nil {
		style.right = override.right
	}
	if override.bottom != nil {
		style.bottom = override.bottom
	}
	if override.width != nil {
		style.width = override.width
	}
	if override.height != nil {
		style.height = override.height
	}
	if override.margin != nil {
		style.margin = override.margin
	}
	if override.overflow != nil {
		style.overflow = override.overflow
	}
	if override.overflowX != nil {
		style.overflowX = override.overflowX
	}
	if override.overflowY != nil {
		style.overflowY = override.overflowY
	}
	if override.scrollbarWidth != nil {
		style.scrollbarWidth = override.scrollbarWidth
	}
	if override.scrollbarTrack != nil {
		style.scrollbarTrack = override.scrollbarTrack
	}
	if override.scrollbarThumb != nil {
		style.scrollbarThumb = override.scrollbarThumb
	}
	if override.scrollbarRadius != nil {
		style.scrollbarRadius = override.scrollbarRadius
	}
	if override.scrollbarPadding != nil {
		style.scrollbarPadding = override.scrollbarPadding
	}
	return style
}
