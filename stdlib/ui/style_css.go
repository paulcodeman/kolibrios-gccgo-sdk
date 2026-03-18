package ui

import (
	"kos"
	"strings"
)

func normalizeCSSKeyword(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ParseTextAlign(value string) (TextAlign, bool) {
	switch normalizeCSSKeyword(value) {
	case "left", "start":
		return TextAlignLeft, true
	case "center":
		return TextAlignCenter, true
	case "right", "end":
		return TextAlignRight, true
	default:
		return 0, false
	}
}

func ParsePosition(value string) (PositionMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "static":
		return PositionStatic, true
	case "relative":
		return PositionRelative, true
	case "absolute":
		return PositionAbsolute, true
	default:
		return 0, false
	}
}

func ParseDisplay(value string) (DisplayMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "inline":
		return DisplayInline, true
	case "inline-block":
		return DisplayInlineBlock, true
	case "block":
		return DisplayBlock, true
	case "none":
		return DisplayNone, true
	default:
		return 0, false
	}
}

func ParseVisibility(value string) (VisibilityMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "visible":
		return VisibilityVisible, true
	case "hidden":
		return VisibilityHidden, true
	default:
		return 0, false
	}
}

func ParseBoxSizing(value string) (BoxSizing, bool) {
	switch normalizeCSSKeyword(value) {
	case "border-box":
		return BoxSizingBorderBox, true
	case "content-box":
		return BoxSizingContentBox, true
	default:
		return 0, false
	}
}

func ParseOverflow(value string) (OverflowMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "visible":
		return OverflowVisible, true
	case "hidden":
		return OverflowHidden, true
	case "scroll":
		return OverflowScroll, true
	case "auto":
		return OverflowAuto, true
	default:
		return 0, false
	}
}

func ParseTextDecoration(value string) (TextDecoration, bool) {
	switch normalizeCSSKeyword(value) {
	case "none":
		return TextDecorationNone, true
	case "underline":
		return TextDecorationUnderline, true
	default:
		return 0, false
	}
}

func ParseWhiteSpace(value string) (WhiteSpaceMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "normal":
		return WhiteSpaceNormal, true
	case "nowrap":
		return WhiteSpaceNoWrap, true
	case "pre":
		return WhiteSpacePre, true
	case "pre-wrap":
		return WhiteSpacePreWrap, true
	case "pre-line":
		return WhiteSpacePreLine, true
	default:
		return 0, false
	}
}

func ParseOverflowWrap(value string) (OverflowWrapMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "normal":
		return OverflowWrapNormal, true
	case "break-word", "anywhere":
		return OverflowWrapBreakWord, true
	default:
		return 0, false
	}
}

func ParseWordBreak(value string) (WordBreakMode, bool) {
	switch normalizeCSSKeyword(value) {
	case "normal":
		return WordBreakNormal, true
	case "break-all":
		return WordBreakBreakAll, true
	default:
		return 0, false
	}
}

func ParseBackgroundAttachment(value string) (BackgroundAttachment, bool) {
	switch normalizeCSSKeyword(value) {
	case "scroll":
		return BackgroundAttachmentScroll, true
	case "fixed":
		return BackgroundAttachmentFixed, true
	case "local":
		return BackgroundAttachmentLocal, true
	default:
		return 0, false
	}
}

func ParseGradientDirection(value string) (GradientDirection, bool) {
	switch normalizeCSSKeyword(value) {
	case "vertical", "to bottom":
		return GradientVertical, true
	case "horizontal", "to right":
		return GradientHorizontal, true
	default:
		return 0, false
	}
}

func spacingOrZero(value *Spacing) Spacing {
	if value == nil {
		return Spacing{}
	}
	return *value
}

func cornerRadiiOrZero(value *CornerRadii) CornerRadii {
	if value == nil {
		return CornerRadii{}
	}
	return *value
}

func setSpacingSide(target **Spacing, side string, value int) {
	current := spacingOrZero(*target)
	switch side {
	case "top":
		current.Top = value
	case "right":
		current.Right = value
	case "bottom":
		current.Bottom = value
	case "left":
		current.Left = value
	}
	*target = &current
}

func setCornerRadius(target **CornerRadii, corner string, value int) {
	if value < 0 {
		value = 0
	}
	current := cornerRadiiOrZero(*target)
	switch corner {
	case "top-left":
		current.TopLeft = value
	case "top-right":
		current.TopRight = value
	case "bottom-right":
		current.BottomRight = value
	case "bottom-left":
		current.BottomLeft = value
	}
	*target = &current
}

func spacingSide(value *Spacing, side string) (int, bool) {
	if value == nil {
		return 0, false
	}
	switch side {
	case "top":
		return value.Top, true
	case "right":
		return value.Right, true
	case "bottom":
		return value.Bottom, true
	case "left":
		return value.Left, true
	default:
		return 0, false
	}
}

func cornerRadiusValue(value *CornerRadii, corner string) (int, bool) {
	if value == nil {
		return 0, false
	}
	switch corner {
	case "top-left":
		return value.TopLeft, true
	case "top-right":
		return value.TopRight, true
	case "bottom-right":
		return value.BottomRight, true
	case "bottom-left":
		return value.BottomLeft, true
	default:
		return 0, false
	}
}

func (style *Style) SetColor(color kos.Color) {
	style.SetForeground(color)
}

func (style Style) GetColor() (kos.Color, bool) {
	return style.GetForeground()
}

func (style *Style) SetBackgroundColor(color kos.Color) {
	style.SetBackground(color)
}

func (style Style) GetBackgroundColor() (kos.Color, bool) {
	return style.GetBackground()
}

func (style *Style) SetBackgroundGradient(value Gradient) {
	style.SetGradient(value)
}

func (style Style) GetBackgroundGradient() (Gradient, bool) {
	return style.GetGradient()
}

func (style *Style) SetBoxShadow(value Shadow) {
	style.SetShadow(value)
}

func (style *Style) SetBoxShadowPtr(value *Shadow) {
	style.SetShadowPtr(value)
}

func (style Style) GetBoxShadow() (Shadow, bool) {
	return style.GetShadow()
}

func (style *Style) SetFontFamily(value string) {
	style.SetFontPath(value)
}

func (style Style) GetFontFamily() (string, bool) {
	return style.GetFontPath()
}

func (style *Style) SetOpacityFloat(value float64) bool {
	if style == nil {
		return false
	}
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	style.SetOpacity(uint8(value*255 + 0.5))
	return true
}

func (style Style) GetOpacityFloat() (float64, bool) {
	value, ok := style.GetOpacity()
	if !ok {
		return 0, false
	}
	return float64(value) / 255, true
}

func (style *Style) SetDisplayString(value string) bool {
	parsed, ok := ParseDisplay(value)
	if !ok || style == nil {
		return false
	}
	style.SetDisplay(parsed)
	return true
}

func (style Style) GetDisplayString() (string, bool) {
	value, ok := style.GetDisplay()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetVisibilityString(value string) bool {
	parsed, ok := ParseVisibility(value)
	if !ok || style == nil {
		return false
	}
	style.SetVisibility(parsed)
	return true
}

func (style Style) GetVisibilityString() (string, bool) {
	value, ok := style.GetVisibility()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetPositionString(value string) bool {
	parsed, ok := ParsePosition(value)
	if !ok || style == nil {
		return false
	}
	style.SetPosition(parsed)
	return true
}

func (style Style) GetPositionString() (string, bool) {
	value, ok := style.GetPosition()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetBoxSizingString(value string) bool {
	parsed, ok := ParseBoxSizing(value)
	if !ok || style == nil {
		return false
	}
	style.SetBoxSizing(parsed)
	return true
}

func (style Style) GetBoxSizingString() (string, bool) {
	value, ok := style.GetBoxSizing()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetTextAlignString(value string) bool {
	parsed, ok := ParseTextAlign(value)
	if !ok || style == nil {
		return false
	}
	style.SetTextAlign(parsed)
	return true
}

func (style Style) GetTextAlignString() (string, bool) {
	value, ok := style.GetTextAlign()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetTextDecorationString(value string) bool {
	parsed, ok := ParseTextDecoration(value)
	if !ok || style == nil {
		return false
	}
	style.SetTextDecoration(parsed)
	return true
}

func (style Style) GetTextDecorationString() (string, bool) {
	value, ok := style.GetTextDecoration()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetWhiteSpaceString(value string) bool {
	parsed, ok := ParseWhiteSpace(value)
	if !ok || style == nil {
		return false
	}
	style.SetWhiteSpace(parsed)
	return true
}

func (style Style) GetWhiteSpaceString() (string, bool) {
	value, ok := style.GetWhiteSpace()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetOverflowWrapString(value string) bool {
	parsed, ok := ParseOverflowWrap(value)
	if !ok || style == nil {
		return false
	}
	style.SetOverflowWrap(parsed)
	return true
}

func (style Style) GetOverflowWrapString() (string, bool) {
	value, ok := style.GetOverflowWrap()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetWordBreakString(value string) bool {
	parsed, ok := ParseWordBreak(value)
	if !ok || style == nil {
		return false
	}
	style.SetWordBreak(parsed)
	return true
}

func (style Style) GetWordBreakString() (string, bool) {
	value, ok := style.GetWordBreak()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetOverflowString(value string) bool {
	parsed, ok := ParseOverflow(value)
	if !ok || style == nil {
		return false
	}
	style.SetOverflow(parsed)
	return true
}

func (style Style) GetOverflowString() (string, bool) {
	value, ok := style.GetOverflow()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetOverflowXString(value string) bool {
	parsed, ok := ParseOverflow(value)
	if !ok || style == nil {
		return false
	}
	style.SetOverflowX(parsed)
	return true
}

func (style Style) GetOverflowXString() (string, bool) {
	value, ok := style.GetOverflowX()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetOverflowYString(value string) bool {
	parsed, ok := ParseOverflow(value)
	if !ok || style == nil {
		return false
	}
	style.SetOverflowY(parsed)
	return true
}

func (style Style) GetOverflowYString() (string, bool) {
	value, ok := style.GetOverflowY()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetOverflowAxes(x OverflowMode, y OverflowMode) {
	if style == nil {
		return
	}
	style.SetOverflowX(x)
	style.SetOverflowY(y)
}

func (style *Style) SetOverflowAxesString(x string, y string) bool {
	if style == nil {
		return false
	}
	xValue, ok := ParseOverflow(x)
	if !ok {
		return false
	}
	yValue, ok := ParseOverflow(y)
	if !ok {
		return false
	}
	style.SetOverflowAxes(xValue, yValue)
	return true
}

func (style Style) GetEffectiveOverflowX() OverflowMode {
	return overflowModeFor(style, "x")
}

func (style Style) GetEffectiveOverflowY() OverflowMode {
	return overflowModeFor(style, "y")
}

func (style Style) GetEffectiveOverflowXString() string {
	return style.GetEffectiveOverflowX().String()
}

func (style Style) GetEffectiveOverflowYString() string {
	return style.GetEffectiveOverflowY().String()
}

func (style *Style) SetBackgroundAttachmentString(value string) bool {
	parsed, ok := ParseBackgroundAttachment(value)
	if !ok || style == nil {
		return false
	}
	style.SetBackgroundAttachment(parsed)
	return true
}

func (style Style) GetBackgroundAttachmentString() (string, bool) {
	value, ok := style.GetBackgroundAttachment()
	if !ok {
		return "", false
	}
	return value.String(), true
}

func (style *Style) SetPaddingTop(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.padding, "top", value)
}

func (style *Style) SetPaddingRight(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.padding, "right", value)
}

func (style *Style) SetPaddingBottom(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.padding, "bottom", value)
}

func (style *Style) SetPaddingLeft(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.padding, "left", value)
}

func (style *Style) SetPaddingX(value int) {
	if style == nil {
		return
	}
	style.SetPaddingLeft(value)
	style.SetPaddingRight(value)
}

func (style *Style) SetPaddingY(value int) {
	if style == nil {
		return
	}
	style.SetPaddingTop(value)
	style.SetPaddingBottom(value)
}

func (style Style) GetPaddingTop() (int, bool) {
	return spacingSide(style.padding, "top")
}

func (style Style) GetPaddingRight() (int, bool) {
	return spacingSide(style.padding, "right")
}

func (style Style) GetPaddingBottom() (int, bool) {
	return spacingSide(style.padding, "bottom")
}

func (style Style) GetPaddingLeft() (int, bool) {
	return spacingSide(style.padding, "left")
}

func (style *Style) SetMarginTop(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.margin, "top", value)
}

func (style *Style) SetMarginRight(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.margin, "right", value)
}

func (style *Style) SetMarginBottom(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.margin, "bottom", value)
}

func (style *Style) SetMarginLeft(value int) {
	if style == nil {
		return
	}
	setSpacingSide(&style.margin, "left", value)
}

func (style *Style) SetMarginX(value int) {
	if style == nil {
		return
	}
	style.SetMarginLeft(value)
	style.SetMarginRight(value)
}

func (style *Style) SetMarginY(value int) {
	if style == nil {
		return
	}
	style.SetMarginTop(value)
	style.SetMarginBottom(value)
}

func (style Style) GetMarginTop() (int, bool) {
	return spacingSide(style.margin, "top")
}

func (style Style) GetMarginRight() (int, bool) {
	return spacingSide(style.margin, "right")
}

func (style Style) GetMarginBottom() (int, bool) {
	return spacingSide(style.margin, "bottom")
}

func (style Style) GetMarginLeft() (int, bool) {
	return spacingSide(style.margin, "left")
}

func (style *Style) SetInset(values ...int) {
	if style == nil || len(values) == 0 {
		return
	}
	top, right, bottom, left := expandBoxShorthand(values)
	style.SetTop(top)
	style.SetRight(right)
	style.SetBottom(bottom)
	style.SetLeft(left)
}

func (style *Style) SetInsetX(value int) {
	if style == nil {
		return
	}
	style.SetLeft(value)
	style.SetRight(value)
}

func (style *Style) SetInsetY(value int) {
	if style == nil {
		return
	}
	style.SetTop(value)
	style.SetBottom(value)
}

func (style *Style) SetBorderTopLeftRadius(value int) {
	if style == nil {
		return
	}
	setCornerRadius(&style.borderRadius, "top-left", value)
}

func (style *Style) SetBorderTopRightRadius(value int) {
	if style == nil {
		return
	}
	setCornerRadius(&style.borderRadius, "top-right", value)
}

func (style *Style) SetBorderBottomRightRadius(value int) {
	if style == nil {
		return
	}
	setCornerRadius(&style.borderRadius, "bottom-right", value)
}

func (style *Style) SetBorderBottomLeftRadius(value int) {
	if style == nil {
		return
	}
	setCornerRadius(&style.borderRadius, "bottom-left", value)
}

func (style Style) GetBorderTopLeftRadius() (int, bool) {
	return cornerRadiusValue(style.borderRadius, "top-left")
}

func (style Style) GetBorderTopRightRadius() (int, bool) {
	return cornerRadiusValue(style.borderRadius, "top-right")
}

func (style Style) GetBorderBottomRightRadius() (int, bool) {
	return cornerRadiusValue(style.borderRadius, "bottom-right")
}

func (style Style) GetBorderBottomLeftRadius() (int, bool) {
	return cornerRadiusValue(style.borderRadius, "bottom-left")
}

func (style *Style) SetBorderTop(width int, color kos.Color) {
	if style == nil {
		return
	}
	style.SetBorderTopWidth(width)
	style.SetBorderTopColor(color)
}

func (style *Style) SetBorderRight(width int, color kos.Color) {
	if style == nil {
		return
	}
	style.SetBorderRightWidth(width)
	style.SetBorderRightColor(color)
}

func (style *Style) SetBorderBottom(width int, color kos.Color) {
	if style == nil {
		return
	}
	style.SetBorderBottomWidth(width)
	style.SetBorderBottomColor(color)
}

func (style *Style) SetBorderLeft(width int, color kos.Color) {
	if style == nil {
		return
	}
	style.SetBorderLeftWidth(width)
	style.SetBorderLeftColor(color)
}
