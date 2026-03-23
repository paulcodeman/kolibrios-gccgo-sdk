package ui

import "kos"

type styleVisualKey struct {
	background           *kos.Color
	foreground           *kos.Color
	borderColor          *kos.Color
	borderWidth          *int
	borderTopColor       *kos.Color
	borderRightColor     *kos.Color
	borderBottomColor    *kos.Color
	borderLeftColor      *kos.Color
	borderTopWidth       *int
	borderRightWidth     *int
	borderBottomWidth    *int
	borderLeftWidth      *int
	borderRadius         *CornerRadii
	gradient             *Gradient
	backgroundAttachment *BackgroundAttachment
	shadow               *Shadow
	visibility           *VisibilityMode
	textAlign            *TextAlign
	textDecoration       *TextDecoration
	whiteSpace           *WhiteSpaceMode
	overflowWrap         *OverflowWrapMode
	wordBreak            *WordBreakMode
	textShadow           *TextShadow
	fontPath             *string
	fontSize             *int
	lineHeight           *int
	padding              *Spacing
	opacity              *uint8
	boxSizing            *BoxSizing
	outlineColor         *kos.Color
	outlineWidth         *int
	outlineOffset        *int
	outlineRadius        *int
	overflow             *OverflowMode
	overflowX            *OverflowMode
	overflowY            *OverflowMode
	contain              *ContainMode
	willChange           *WillChangeHints
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
	if value, ok := resolveColor(style.borderTopColor); ok {
		v := value
		key.borderTopColor = &v
	}
	if value, ok := resolveColor(style.borderRightColor); ok {
		v := value
		key.borderRightColor = &v
	}
	if value, ok := resolveColor(style.borderBottomColor); ok {
		v := value
		key.borderBottomColor = &v
	}
	if value, ok := resolveColor(style.borderLeftColor); ok {
		v := value
		key.borderLeftColor = &v
	}
	if value, ok := resolveLength(style.borderTopWidth); ok {
		v := value
		key.borderTopWidth = &v
	}
	if value, ok := resolveLength(style.borderRightWidth); ok {
		v := value
		key.borderRightWidth = &v
	}
	if value, ok := resolveLength(style.borderBottomWidth); ok {
		v := value
		key.borderBottomWidth = &v
	}
	if value, ok := resolveLength(style.borderLeftWidth); ok {
		v := value
		key.borderLeftWidth = &v
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
	if value, ok := resolveVisibility(style.visibility); ok {
		v := value
		key.visibility = &v
	}
	if value, ok := resolveTextAlign(style.textAlign); ok {
		v := value
		key.textAlign = &v
	}
	if value, ok := resolveTextDecoration(style.textDecoration); ok {
		v := value
		key.textDecoration = &v
	}
	if value, ok := resolveWhiteSpace(style.whiteSpace); ok {
		v := value
		key.whiteSpace = &v
	}
	if value, ok := resolveOverflowWrap(style.overflowWrap); ok {
		v := value
		key.overflowWrap = &v
	}
	if value, ok := resolveWordBreak(style.wordBreak); ok {
		v := value
		key.wordBreak = &v
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
	if value, ok := resolveLineHeight(style.lineHeight); ok {
		v := value
		key.lineHeight = &v
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
	if value, ok := resolveBoxSizing(style.boxSizing); ok {
		v := value
		key.boxSizing = &v
	}
	if value, ok := resolveColor(style.outlineColor); ok {
		v := value
		key.outlineColor = &v
	}
	if value, ok := resolveLength(style.outlineWidth); ok {
		v := value
		key.outlineWidth = &v
	}
	if value, ok := resolveLength(style.outlineOffset); ok {
		v := value
		key.outlineOffset = &v
	}
	if value, ok := resolveLength(style.outlineRadius); ok {
		v := value
		key.outlineRadius = &v
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
	if value, ok := resolveContain(style.contain); ok {
		v := value
		key.contain = &v
	}
	if value, ok := resolveWillChange(style.willChange); ok {
		v := value
		key.willChange = &v
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
		equalColorPtr(a.borderTopColor, b.borderTopColor) &&
		equalColorPtr(a.borderRightColor, b.borderRightColor) &&
		equalColorPtr(a.borderBottomColor, b.borderBottomColor) &&
		equalColorPtr(a.borderLeftColor, b.borderLeftColor) &&
		equalIntPtr(a.borderTopWidth, b.borderTopWidth) &&
		equalIntPtr(a.borderRightWidth, b.borderRightWidth) &&
		equalIntPtr(a.borderBottomWidth, b.borderBottomWidth) &&
		equalIntPtr(a.borderLeftWidth, b.borderLeftWidth) &&
		equalCornerRadiiPtr(a.borderRadius, b.borderRadius) &&
		equalGradientPtr(a.gradient, b.gradient) &&
		equalBackgroundAttachmentPtr(a.backgroundAttachment, b.backgroundAttachment) &&
		equalShadowPtr(a.shadow, b.shadow) &&
		equalVisibilityPtr(a.visibility, b.visibility) &&
		equalTextAlignPtr(a.textAlign, b.textAlign) &&
		equalTextDecorationPtr(a.textDecoration, b.textDecoration) &&
		equalWhiteSpacePtr(a.whiteSpace, b.whiteSpace) &&
		equalOverflowWrapPtr(a.overflowWrap, b.overflowWrap) &&
		equalWordBreakPtr(a.wordBreak, b.wordBreak) &&
		equalTextShadowPtr(a.textShadow, b.textShadow) &&
		equalStringPtr(a.fontPath, b.fontPath) &&
		equalIntPtr(a.fontSize, b.fontSize) &&
		equalIntPtr(a.lineHeight, b.lineHeight) &&
		equalSpacingPtr(a.padding, b.padding) &&
		equalBytePtr(a.opacity, b.opacity) &&
		equalBoxSizingPtr(a.boxSizing, b.boxSizing) &&
		equalColorPtr(a.outlineColor, b.outlineColor) &&
		equalIntPtr(a.outlineWidth, b.outlineWidth) &&
		equalIntPtr(a.outlineOffset, b.outlineOffset) &&
		equalIntPtr(a.outlineRadius, b.outlineRadius) &&
		equalOverflowPtr(a.overflow, b.overflow) &&
		equalOverflowPtr(a.overflowX, b.overflowX) &&
		equalOverflowPtr(a.overflowY, b.overflowY) &&
		equalContainPtr(a.contain, b.contain) &&
		equalWillChangePtr(a.willChange, b.willChange) &&
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

func equalVisibilityPtr(a *VisibilityMode, b *VisibilityMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalTextDecorationPtr(a *TextDecoration, b *TextDecoration) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalWhiteSpacePtr(a *WhiteSpaceMode, b *WhiteSpaceMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalOverflowWrapPtr(a *OverflowWrapMode, b *OverflowWrapMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalWordBreakPtr(a *WordBreakMode, b *WordBreakMode) bool {
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

func equalBoxSizingPtr(a *BoxSizing, b *BoxSizing) bool {
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

func equalContainPtr(a *ContainMode, b *ContainMode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalWillChangePtr(a *WillChangeHints, b *WillChangeHints) bool {
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
	if override.borderTopColor != nil {
		style.borderTopColor = override.borderTopColor
	}
	if override.borderRightColor != nil {
		style.borderRightColor = override.borderRightColor
	}
	if override.borderBottomColor != nil {
		style.borderBottomColor = override.borderBottomColor
	}
	if override.borderLeftColor != nil {
		style.borderLeftColor = override.borderLeftColor
	}
	if override.borderTopWidth != nil {
		style.borderTopWidth = override.borderTopWidth
	}
	if override.borderRightWidth != nil {
		style.borderRightWidth = override.borderRightWidth
	}
	if override.borderBottomWidth != nil {
		style.borderBottomWidth = override.borderBottomWidth
	}
	if override.borderLeftWidth != nil {
		style.borderLeftWidth = override.borderLeftWidth
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
	if override.visibility != nil {
		style.visibility = override.visibility
	}
	if override.textAlign != nil {
		style.textAlign = override.textAlign
	}
	if override.textDecoration != nil {
		style.textDecoration = override.textDecoration
	}
	if override.whiteSpace != nil {
		style.whiteSpace = override.whiteSpace
	}
	if override.overflowWrap != nil {
		style.overflowWrap = override.overflowWrap
	}
	if override.wordBreak != nil {
		style.wordBreak = override.wordBreak
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
	if override.lineHeight != nil {
		style.lineHeight = override.lineHeight
	}
	if override.padding != nil {
		style.padding = override.padding
	}
	if override.opacity != nil {
		style.opacity = override.opacity
	}
	if override.boxSizing != nil {
		style.boxSizing = override.boxSizing
	}
	if override.outlineColor != nil {
		style.outlineColor = override.outlineColor
	}
	if override.outlineWidth != nil {
		style.outlineWidth = override.outlineWidth
	}
	if override.outlineOffset != nil {
		style.outlineOffset = override.outlineOffset
	}
	if override.outlineRadius != nil {
		style.outlineRadius = override.outlineRadius
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
	if override.minWidth != nil {
		style.minWidth = override.minWidth
	}
	if override.maxWidth != nil {
		style.maxWidth = override.maxWidth
	}
	if override.minHeight != nil {
		style.minHeight = override.minHeight
	}
	if override.maxHeight != nil {
		style.maxHeight = override.maxHeight
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
	if override.contain != nil {
		style.contain = override.contain
	}
	if override.willChange != nil {
		style.willChange = override.willChange
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
