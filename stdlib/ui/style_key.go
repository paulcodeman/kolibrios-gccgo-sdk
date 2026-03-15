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
	if value, ok := resolveColor(style.Background); ok {
		v := value
		key.background = &v
	}
	if value, ok := resolveColor(style.Foreground); ok {
		v := value
		key.foreground = &v
	}
	if value, ok := resolveColor(style.BorderColor); ok {
		v := value
		key.borderColor = &v
	}
	if value, ok := resolveLength(style.BorderWidth); ok {
		v := value
		key.borderWidth = &v
	}
	if value, ok := resolveCornerRadii(style.BorderRadius); ok {
		if value != nil {
			v := *value
			key.borderRadius = &v
		}
	}
	if value, ok := resolveGradient(style.Gradient); ok {
		if value != nil {
			v := *value
			key.gradient = &v
		}
	}
	if value, ok := resolveBackgroundAttachment(style.BackgroundAttachment); ok {
		v := value
		key.backgroundAttachment = &v
	}
	if value, ok := resolveShadow(style.Shadow); ok {
		if value != nil {
			v := *value
			key.shadow = &v
		}
	}
	if value, ok := resolveTextAlign(style.TextAlign); ok {
		v := value
		key.textAlign = &v
	}
	if value, ok := resolveTextShadow(style.TextShadow); ok {
		if value != nil {
			v := *value
			key.textShadow = &v
		}
	}
	if value, ok := resolveFontPath(style.FontPath); ok {
		v := value
		key.fontPath = &v
	}
	if value, ok := resolveFontSize(style.FontSize); ok {
		v := value
		key.fontSize = &v
	}
	if value, ok := resolveSpacing(style.Padding); ok {
		if value != nil {
			v := *value
			key.padding = &v
		}
	}
	if value, ok := resolveOpacity(style.Opacity); ok {
		v := value
		key.opacity = &v
	}
	if value, ok := resolveOverflow(style.Overflow); ok {
		v := value
		key.overflow = &v
	}
	if value, ok := resolveOverflow(style.OverflowX); ok {
		v := value
		key.overflowX = &v
	}
	if value, ok := resolveOverflow(style.OverflowY); ok {
		v := value
		key.overflowY = &v
	}
	if value, ok := resolveScrollbarWidth(style.ScrollbarWidth); ok {
		v := value
		key.scrollbarWidth = &v
	}
	if value, ok := resolveColor(style.ScrollbarTrack); ok {
		v := value
		key.scrollbarTrack = &v
	}
	if value, ok := resolveColor(style.ScrollbarThumb); ok {
		v := value
		key.scrollbarThumb = &v
	}
	if value, ok := resolveScrollbarRadius(style.ScrollbarRadius); ok {
		v := value
		key.scrollbarRadius = &v
	}
	if value, ok := resolveSpacing(style.ScrollbarPadding); ok {
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
	if override.Background != nil {
		style.Background = override.Background
	}
	if override.Foreground != nil {
		style.Foreground = override.Foreground
	}
	if override.BorderColor != nil {
		style.BorderColor = override.BorderColor
	}
	if override.BorderWidth != nil {
		style.BorderWidth = override.BorderWidth
	}
	if override.BorderRadius != nil {
		style.BorderRadius = override.BorderRadius
	}
	if override.Gradient != nil {
		style.Gradient = override.Gradient
	}
	if override.BackgroundAttachment != nil {
		style.BackgroundAttachment = override.BackgroundAttachment
	}
	if override.Shadow != nil {
		style.Shadow = override.Shadow
	}
	if override.Display != nil {
		style.Display = override.Display
	}
	if override.TextAlign != nil {
		style.TextAlign = override.TextAlign
	}
	if override.TextShadow != nil {
		style.TextShadow = override.TextShadow
	}
	if override.FontPath != nil {
		style.FontPath = override.FontPath
	}
	if override.FontSize != nil {
		style.FontSize = override.FontSize
	}
	if override.Padding != nil {
		style.Padding = override.Padding
	}
	if override.Opacity != nil {
		style.Opacity = override.Opacity
	}
	if override.Position != nil {
		style.Position = override.Position
	}
	if override.Left != nil {
		style.Left = override.Left
	}
	if override.Top != nil {
		style.Top = override.Top
	}
	if override.Right != nil {
		style.Right = override.Right
	}
	if override.Bottom != nil {
		style.Bottom = override.Bottom
	}
	if override.Width != nil {
		style.Width = override.Width
	}
	if override.Height != nil {
		style.Height = override.Height
	}
	if override.Margin != nil {
		style.Margin = override.Margin
	}
	if override.Overflow != nil {
		style.Overflow = override.Overflow
	}
	if override.OverflowX != nil {
		style.OverflowX = override.OverflowX
	}
	if override.OverflowY != nil {
		style.OverflowY = override.OverflowY
	}
	if override.ScrollbarWidth != nil {
		style.ScrollbarWidth = override.ScrollbarWidth
	}
	if override.ScrollbarTrack != nil {
		style.ScrollbarTrack = override.ScrollbarTrack
	}
	if override.ScrollbarThumb != nil {
		style.ScrollbarThumb = override.ScrollbarThumb
	}
	if override.ScrollbarRadius != nil {
		style.ScrollbarRadius = override.ScrollbarRadius
	}
	if override.ScrollbarPadding != nil {
		style.ScrollbarPadding = override.ScrollbarPadding
	}
	return style
}
