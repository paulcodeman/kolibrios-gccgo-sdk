package ui

import "kos"

type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

type PositionMode int

const (
	PositionStatic PositionMode = iota
	PositionRelative
	PositionAbsolute
)

type DisplayMode int

const (
	DisplayInline DisplayMode = iota
	DisplayInlineBlock
	DisplayBlock
	DisplayNone
)

type OverflowMode int

const (
	OverflowVisible OverflowMode = iota
	OverflowHidden
	OverflowScroll
	OverflowAuto
)

type BackgroundAttachment int

const (
	BackgroundAttachmentScroll BackgroundAttachment = iota
	BackgroundAttachmentFixed
	BackgroundAttachmentLocal
)

type GradientDirection int

const (
	GradientVertical GradientDirection = iota
	GradientHorizontal
)

type Gradient struct {
	From      kos.Color
	To        kos.Color
	Direction GradientDirection
}

type Shadow struct {
	OffsetX int
	OffsetY int
	Blur    int
	Color   kos.Color
	Alpha   uint8
}

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

type CornerRadii struct {
	TopLeft     int
	TopRight    int
	BottomRight int
	BottomLeft  int
}

func (radii CornerRadii) Active() bool {
	return radii.TopLeft != 0 ||
		radii.TopRight != 0 ||
		radii.BottomRight != 0 ||
		radii.BottomLeft != 0
}

type Style struct {
	background           *kos.Color
	foreground           *kos.Color
	borderColor          *kos.Color
	borderWidth          *int
	borderRadius         *CornerRadii
	gradient             *Gradient
	backgroundAttachment *BackgroundAttachment
	shadow               *Shadow
	display              *DisplayMode
	textAlign            *TextAlign
	textShadow           *TextShadow
	fontPath             *string
	fontSize             *int
	padding              *Spacing
	opacity              *uint8
	position             *PositionMode
	left                 *int
	top                  *int
	right                *int
	bottom               *int
	width                *int
	height               *int
	margin               *Spacing
	overflow             *OverflowMode
	overflowX            *OverflowMode
	overflowY            *OverflowMode
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
		style.borderRadius == nil &&
		style.gradient == nil &&
		style.backgroundAttachment == nil &&
		style.shadow == nil &&
		style.display == nil &&
		style.textAlign == nil &&
		style.textShadow == nil &&
		style.fontPath == nil &&
		style.fontSize == nil &&
		style.padding == nil &&
		style.opacity == nil &&
		style.position == nil &&
		style.left == nil &&
		style.top == nil &&
		style.right == nil &&
		style.bottom == nil &&
		style.width == nil &&
		style.height == nil &&
		style.margin == nil &&
		style.overflow == nil &&
		style.overflowX == nil &&
		style.overflowY == nil &&
		style.scrollbarWidth == nil &&
		style.scrollbarTrack == nil &&
		style.scrollbarThumb == nil &&
		style.scrollbarRadius == nil &&
		style.scrollbarPadding == nil
}

func (style Style) HasLayout() bool {
	return style.display != nil ||
		style.position != nil ||
		style.left != nil ||
		style.top != nil ||
		style.right != nil ||
		style.bottom != nil ||
		style.width != nil ||
		style.height != nil ||
		style.margin != nil
}

func (style Style) HasVisual() bool {
	return style.background != nil ||
		style.foreground != nil ||
		style.borderColor != nil ||
		style.borderWidth != nil ||
		style.borderRadius != nil ||
		style.gradient != nil ||
		style.backgroundAttachment != nil ||
		style.shadow != nil ||
		style.textAlign != nil ||
		style.textShadow != nil ||
		style.fontPath != nil ||
		style.fontSize != nil ||
		style.padding != nil ||
		style.opacity != nil ||
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
