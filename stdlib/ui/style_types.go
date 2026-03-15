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
	Background           *kos.Color
	Foreground           *kos.Color
	BorderColor          *kos.Color
	BorderWidth          *int
	BorderRadius         *CornerRadii
	Gradient             *Gradient
	BackgroundAttachment *BackgroundAttachment
	Shadow               *Shadow
	Display              *DisplayMode
	TextAlign            *TextAlign
	TextShadow           *TextShadow
	FontPath             *string
	FontSize             *int
	Padding              *Spacing
	Opacity              *uint8
	Position             *PositionMode
	Left                 *int
	Top                  *int
	Right                *int
	Bottom               *int
	Width                *int
	Height               *int
	Margin               *Spacing
	Overflow             *OverflowMode
	OverflowX            *OverflowMode
	OverflowY            *OverflowMode
	ScrollbarWidth       *int
	ScrollbarTrack       *kos.Color
	ScrollbarThumb       *kos.Color
	ScrollbarRadius      *int
	ScrollbarPadding     *Spacing
}

func (style Style) IsZero() bool {
	return style.Background == nil &&
		style.Foreground == nil &&
		style.BorderColor == nil &&
		style.BorderWidth == nil &&
		style.BorderRadius == nil &&
		style.Gradient == nil &&
		style.BackgroundAttachment == nil &&
		style.Shadow == nil &&
		style.Display == nil &&
		style.TextAlign == nil &&
		style.TextShadow == nil &&
		style.FontPath == nil &&
		style.FontSize == nil &&
		style.Padding == nil &&
		style.Opacity == nil &&
		style.Position == nil &&
		style.Left == nil &&
		style.Top == nil &&
		style.Right == nil &&
		style.Bottom == nil &&
		style.Width == nil &&
		style.Height == nil &&
		style.Margin == nil &&
		style.Overflow == nil &&
		style.OverflowX == nil &&
		style.OverflowY == nil &&
		style.ScrollbarWidth == nil &&
		style.ScrollbarTrack == nil &&
		style.ScrollbarThumb == nil &&
		style.ScrollbarRadius == nil &&
		style.ScrollbarPadding == nil
}

func (style Style) HasLayout() bool {
	return style.Display != nil ||
		style.Position != nil ||
		style.Left != nil ||
		style.Top != nil ||
		style.Right != nil ||
		style.Bottom != nil ||
		style.Width != nil ||
		style.Height != nil ||
		style.Margin != nil
}

func (style Style) HasVisual() bool {
	return style.Background != nil ||
		style.Foreground != nil ||
		style.BorderColor != nil ||
		style.BorderWidth != nil ||
		style.BorderRadius != nil ||
		style.Gradient != nil ||
		style.BackgroundAttachment != nil ||
		style.Shadow != nil ||
		style.TextAlign != nil ||
		style.TextShadow != nil ||
		style.FontPath != nil ||
		style.FontSize != nil ||
		style.Padding != nil ||
		style.Opacity != nil ||
		style.Overflow != nil ||
		style.OverflowX != nil ||
		style.OverflowY != nil ||
		style.ScrollbarWidth != nil ||
		style.ScrollbarTrack != nil ||
		style.ScrollbarThumb != nil ||
		style.ScrollbarRadius != nil ||
		style.ScrollbarPadding != nil
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
