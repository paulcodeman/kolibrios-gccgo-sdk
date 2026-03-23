package ui

import (
	"kos"
	surfacepkg "surface"
)

type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

func (value TextAlign) String() string {
	switch value {
	case TextAlignLeft:
		return "left"
	case TextAlignCenter:
		return "center"
	case TextAlignRight:
		return "right"
	default:
		return ""
	}
}

type PositionMode int

const (
	PositionStatic PositionMode = iota
	PositionRelative
	PositionAbsolute
)

func (value PositionMode) String() string {
	switch value {
	case PositionStatic:
		return "static"
	case PositionRelative:
		return "relative"
	case PositionAbsolute:
		return "absolute"
	default:
		return ""
	}
}

type DisplayMode int

const (
	DisplayInline DisplayMode = iota
	DisplayInlineBlock
	DisplayBlock
	DisplayNone
)

func (value DisplayMode) String() string {
	switch value {
	case DisplayInline:
		return "inline"
	case DisplayInlineBlock:
		return "inline-block"
	case DisplayBlock:
		return "block"
	case DisplayNone:
		return "none"
	default:
		return ""
	}
}

type OverflowMode int

const (
	OverflowVisible OverflowMode = iota
	OverflowHidden
	OverflowScroll
	OverflowAuto
)

func (value OverflowMode) String() string {
	switch value {
	case OverflowVisible:
		return "visible"
	case OverflowHidden:
		return "hidden"
	case OverflowScroll:
		return "scroll"
	case OverflowAuto:
		return "auto"
	default:
		return ""
	}
}

type BackgroundAttachment int

const (
	BackgroundAttachmentScroll BackgroundAttachment = iota
	BackgroundAttachmentFixed
	BackgroundAttachmentLocal
)

func (value BackgroundAttachment) String() string {
	switch value {
	case BackgroundAttachmentScroll:
		return "scroll"
	case BackgroundAttachmentFixed:
		return "fixed"
	case BackgroundAttachmentLocal:
		return "local"
	default:
		return ""
	}
}

type VisibilityMode int

const (
	VisibilityVisible VisibilityMode = iota
	VisibilityHidden
)

func (value VisibilityMode) String() string {
	switch value {
	case VisibilityVisible:
		return "visible"
	case VisibilityHidden:
		return "hidden"
	default:
		return ""
	}
}

type BoxSizing int

const (
	BoxSizingBorderBox BoxSizing = iota
	BoxSizingContentBox
)

func (value BoxSizing) String() string {
	switch value {
	case BoxSizingBorderBox:
		return "border-box"
	case BoxSizingContentBox:
		return "content-box"
	default:
		return ""
	}
}

type TextDecoration int

const (
	TextDecorationNone TextDecoration = iota
	TextDecorationUnderline
)

func (value TextDecoration) String() string {
	switch value {
	case TextDecorationNone:
		return "none"
	case TextDecorationUnderline:
		return "underline"
	default:
		return ""
	}
}

type WhiteSpaceMode int

const (
	WhiteSpaceNormal WhiteSpaceMode = iota
	WhiteSpaceNoWrap
	WhiteSpacePre
	WhiteSpacePreWrap
	WhiteSpacePreLine
)

func (value WhiteSpaceMode) String() string {
	switch value {
	case WhiteSpaceNormal:
		return "normal"
	case WhiteSpaceNoWrap:
		return "nowrap"
	case WhiteSpacePre:
		return "pre"
	case WhiteSpacePreWrap:
		return "pre-wrap"
	case WhiteSpacePreLine:
		return "pre-line"
	default:
		return ""
	}
}

type OverflowWrapMode int

const (
	OverflowWrapNormal OverflowWrapMode = iota
	OverflowWrapBreakWord
)

func (value OverflowWrapMode) String() string {
	switch value {
	case OverflowWrapNormal:
		return "normal"
	case OverflowWrapBreakWord:
		return "break-word"
	default:
		return ""
	}
}

type WordBreakMode int

const (
	WordBreakNormal WordBreakMode = iota
	WordBreakBreakAll
)

func (value WordBreakMode) String() string {
	switch value {
	case WordBreakNormal:
		return "normal"
	case WordBreakBreakAll:
		return "break-all"
	default:
		return ""
	}
}

type ContainMode int

const (
	ContainNone   ContainMode = 0
	ContainLayout ContainMode = 1 << iota
	ContainPaint
)

const ContainContent = ContainLayout | ContainPaint

func (value ContainMode) String() string {
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

type WillChangeHints int

const (
	WillChangeAuto     WillChangeHints = 0
	WillChangeContents WillChangeHints = 1 << iota
	WillChangeScrollPosition
	WillChangeTransform
	WillChangeOpacity
)

func (value WillChangeHints) String() string {
	return willChangeString(value)
}

type GradientDirection = surfacepkg.GradientDirection

const (
	GradientVertical   GradientDirection = surfacepkg.GradientVertical
	GradientHorizontal GradientDirection = surfacepkg.GradientHorizontal
)

type Gradient = surfacepkg.Gradient

type Shadow = surfacepkg.Shadow

type TextShadow struct {
	OffsetX int
	OffsetY int
	Color   kos.Color
}

type Spacing struct {
	Left   int
	Top    int
	Right  int
	Bottom int
}

type CornerRadii = surfacepkg.CornerRadii

type Style struct {
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
	display              *DisplayMode
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
	position             *PositionMode
	left                 *int
	top                  *int
	right                *int
	bottom               *int
	width                *int
	height               *int
	minWidth             *int
	maxWidth             *int
	minHeight            *int
	maxHeight            *int
	margin               *Spacing
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

func (style Style) IsZero() bool {
	return style.background == nil &&
		style.foreground == nil &&
		style.borderColor == nil &&
		style.borderWidth == nil &&
		style.borderTopColor == nil &&
		style.borderRightColor == nil &&
		style.borderBottomColor == nil &&
		style.borderLeftColor == nil &&
		style.borderTopWidth == nil &&
		style.borderRightWidth == nil &&
		style.borderBottomWidth == nil &&
		style.borderLeftWidth == nil &&
		style.borderRadius == nil &&
		style.gradient == nil &&
		style.backgroundAttachment == nil &&
		style.shadow == nil &&
		style.display == nil &&
		style.visibility == nil &&
		style.textAlign == nil &&
		style.textDecoration == nil &&
		style.whiteSpace == nil &&
		style.overflowWrap == nil &&
		style.wordBreak == nil &&
		style.textShadow == nil &&
		style.fontPath == nil &&
		style.fontSize == nil &&
		style.lineHeight == nil &&
		style.padding == nil &&
		style.opacity == nil &&
		style.boxSizing == nil &&
		style.outlineColor == nil &&
		style.outlineWidth == nil &&
		style.outlineOffset == nil &&
		style.outlineRadius == nil &&
		style.position == nil &&
		style.left == nil &&
		style.top == nil &&
		style.right == nil &&
		style.bottom == nil &&
		style.width == nil &&
		style.height == nil &&
		style.minWidth == nil &&
		style.maxWidth == nil &&
		style.minHeight == nil &&
		style.maxHeight == nil &&
		style.margin == nil &&
		style.overflow == nil &&
		style.overflowX == nil &&
		style.overflowY == nil &&
		style.contain == nil &&
		style.willChange == nil &&
		style.scrollbarWidth == nil &&
		style.scrollbarTrack == nil &&
		style.scrollbarThumb == nil &&
		style.scrollbarRadius == nil &&
		style.scrollbarPadding == nil
}

func (style Style) HasLayout() bool {
	return style.display != nil ||
		style.visibility != nil ||
		style.position != nil ||
		style.left != nil ||
		style.top != nil ||
		style.right != nil ||
		style.bottom != nil ||
		style.width != nil ||
		style.height != nil ||
		style.minWidth != nil ||
		style.maxWidth != nil ||
		style.minHeight != nil ||
		style.maxHeight != nil ||
		style.boxSizing != nil ||
		style.lineHeight != nil ||
		style.whiteSpace != nil ||
		style.overflowWrap != nil ||
		style.wordBreak != nil ||
		style.borderWidth != nil ||
		style.borderTopWidth != nil ||
		style.borderRightWidth != nil ||
		style.borderBottomWidth != nil ||
		style.borderLeftWidth != nil ||
		style.padding != nil ||
		style.margin != nil
}

func (style Style) HasVisual() bool {
	return style.background != nil ||
		style.foreground != nil ||
		style.borderColor != nil ||
		style.borderWidth != nil ||
		style.borderTopColor != nil ||
		style.borderRightColor != nil ||
		style.borderBottomColor != nil ||
		style.borderLeftColor != nil ||
		style.borderTopWidth != nil ||
		style.borderRightWidth != nil ||
		style.borderBottomWidth != nil ||
		style.borderLeftWidth != nil ||
		style.borderRadius != nil ||
		style.gradient != nil ||
		style.backgroundAttachment != nil ||
		style.shadow != nil ||
		style.visibility != nil ||
		style.textAlign != nil ||
		style.textDecoration != nil ||
		style.whiteSpace != nil ||
		style.overflowWrap != nil ||
		style.wordBreak != nil ||
		style.textShadow != nil ||
		style.fontPath != nil ||
		style.fontSize != nil ||
		style.lineHeight != nil ||
		style.padding != nil ||
		style.opacity != nil ||
		style.outlineColor != nil ||
		style.outlineWidth != nil ||
		style.outlineOffset != nil ||
		style.outlineRadius != nil ||
		style.overflow != nil ||
		style.overflowX != nil ||
		style.overflowY != nil ||
		style.scrollbarWidth != nil ||
		style.scrollbarTrack != nil ||
		style.scrollbarThumb != nil ||
		style.scrollbarRadius != nil ||
		style.scrollbarPadding != nil
}

func ColorPtr(value kos.Color) *kos.Color {
	v := value
	return &v
}

func IntPtr(value int) *int {
	v := value
	return &v
}

func BytePtr(value uint8) *uint8 {
	v := value
	return &v
}

func StringPtr(value string) *string {
	v := value
	return &v
}

func AlignPtr(value TextAlign) *TextAlign {
	v := value
	return &v
}

func PositionPtr(value PositionMode) *PositionMode {
	v := value
	return &v
}

func DisplayPtr(value DisplayMode) *DisplayMode {
	v := value
	return &v
}

func OverflowPtr(value OverflowMode) *OverflowMode {
	v := value
	return &v
}

func BackgroundAttachmentPtr(value BackgroundAttachment) *BackgroundAttachment {
	v := value
	return &v
}

func VisibilityPtr(value VisibilityMode) *VisibilityMode {
	v := value
	return &v
}

func BoxSizingPtr(value BoxSizing) *BoxSizing {
	v := value
	return &v
}

func TextDecorationPtr(value TextDecoration) *TextDecoration {
	v := value
	return &v
}

func WhiteSpacePtr(value WhiteSpaceMode) *WhiteSpaceMode {
	v := value
	return &v
}

func OverflowWrapPtr(value OverflowWrapMode) *OverflowWrapMode {
	v := value
	return &v
}

func WordBreakPtr(value WordBreakMode) *WordBreakMode {
	v := value
	return &v
}

func ContainPtr(value ContainMode) *ContainMode {
	v := value
	return &v
}

func WillChangePtr(value WillChangeHints) *WillChangeHints {
	v := value
	return &v
}
