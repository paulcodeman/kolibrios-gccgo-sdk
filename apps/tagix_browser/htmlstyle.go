package main

import (
	"kos"
	"strconv"
	"strings"
	"ui"
)

var htmlNamedColors = map[string]kos.Color{
	"black":       ui.Black,
	"gray":        ui.Gray,
	"grey":        ui.Gray,
	"silver":      ui.Silver,
	"white":       ui.White,
	"fuchsia":     ui.Fuchsia,
	"purple":      ui.Purple,
	"red":         ui.Red,
	"maroon":      ui.Maroon,
	"yellow":      ui.Yellow,
	"olive":       ui.Olive,
	"lime":        ui.Lime,
	"green":       ui.Green,
	"aqua":        ui.Aqua,
	"teal":        ui.Teal,
	"blue":        ui.Blue,
	"navy":        ui.Navy,
	"orange":      0xFFA500,
	"cyan":        ui.Aqua,
	"magenta":     ui.Fuchsia,
	"transparent": ui.White,
}

func applyNodeInlineStyle(style *ui.Style, node *Node) {
	if style == nil || node == nil {
		return
	}
	applyInlineStyle(style, attrValue(node, "style"))
}

func applyInlineStyle(style *ui.Style, declarations string) {
	if style == nil {
		return
	}
	for _, chunk := range strings.Split(declarations, ";") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		colon := strings.IndexByte(chunk, ':')
		if colon <= 0 || colon+1 >= len(chunk) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(chunk[:colon]))
		value := strings.TrimSpace(chunk[colon+1:])
		applyInlineStyleRule(style, name, value)
	}
}

func applyInlineStyleRule(style *ui.Style, name string, value string) {
	if style == nil {
		return
	}
	switch name {
	case "color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetForeground(color)
		}
	case "background", "background-color":
		if color, ok := parseHTMLColor(lastStyleToken(value)); ok {
			style.SetBackground(color)
		}
	case "display":
		style.SetDisplayString(value)
	case "position":
		style.SetPositionString(value)
	case "text-align":
		style.SetTextAlignString(value)
	case "text-decoration":
		style.SetTextDecorationString(value)
	case "white-space":
		style.SetWhiteSpaceString(value)
	case "overflow":
		style.SetOverflowString(value)
	case "overflow-x":
		style.SetOverflowXString(value)
	case "overflow-y":
		style.SetOverflowYString(value)
	case "contain":
		style.SetContainString(value)
	case "will-change":
		style.SetWillChangeString(value)
	case "opacity":
		if parsed, ok := parseHTMLOpacity(value); ok {
			style.SetOpacityFloat(parsed)
		}
	case "width":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetWidth(parsed)
		}
	case "height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetHeight(parsed)
		}
	case "min-width":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMinWidth(parsed)
		}
	case "max-width":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMaxWidth(parsed)
		}
	case "min-height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMinHeight(parsed)
		}
	case "max-height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMaxHeight(parsed)
		}
	case "margin":
		if values, ok := parseHTMLBoxLengths(value); ok {
			style.SetMargin(values...)
		}
	case "margin-top":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginTop(parsed)
		}
	case "margin-right":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginRight(parsed)
		}
	case "margin-bottom":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginBottom(parsed)
		}
	case "margin-left":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginLeft(parsed)
		}
	case "padding":
		if values, ok := parseHTMLBoxLengths(value); ok {
			style.SetPadding(values...)
		}
	case "padding-top":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingTop(parsed)
		}
	case "padding-right":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingRight(parsed)
		}
	case "padding-bottom":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingBottom(parsed)
		}
	case "padding-left":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingLeft(parsed)
		}
	case "border":
		applyHTMLBorder(style, value, "")
	case "border-top":
		applyHTMLBorder(style, value, "top")
	case "border-right":
		applyHTMLBorder(style, value, "right")
	case "border-bottom":
		applyHTMLBorder(style, value, "bottom")
	case "border-left":
		applyHTMLBorder(style, value, "left")
	case "border-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderColor(color)
		}
	case "border-top-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderTopColor(color)
		}
	case "border-right-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderRightColor(color)
		}
	case "border-bottom-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderBottomColor(color)
		}
	case "border-left-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderLeftColor(color)
		}
	case "border-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderColor()
			style.SetBorder(parsed, color)
		}
	case "border-top-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderTopColor()
			style.SetBorderTop(parsed, color)
		}
	case "border-right-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderRightColor()
			style.SetBorderRight(parsed, color)
		}
	case "border-bottom-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderBottomColor()
			style.SetBorderBottom(parsed, color)
		}
	case "border-left-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderLeftColor()
			style.SetBorderLeft(parsed, color)
		}
	case "border-radius":
		if values, ok := parseHTMLBoxLengths(value); ok {
			style.SetBorderRadius(values...)
		}
	case "font-family":
		if path := parseHTMLFontPath(value); path != "" {
			style.SetFontPath(path)
		}
	case "font-size":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetFontSize(parsed)
		}
	case "line-height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetLineHeight(parsed)
		}
	case "outline":
		if width, color, ok := parseHTMLBorder(value); ok {
			style.SetOutline(width, color)
		}
	case "outline-offset":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetOutlineOffset(parsed)
		}
	}
}

func applyHTMLBorder(style *ui.Style, value string, side string) {
	if style == nil {
		return
	}
	width, color, ok := parseHTMLBorder(value)
	if !ok {
		return
	}
	switch side {
	case "top":
		style.SetBorderTop(width, color)
	case "right":
		style.SetBorderRight(width, color)
	case "bottom":
		style.SetBorderBottom(width, color)
	case "left":
		style.SetBorderLeft(width, color)
	default:
		style.SetBorder(width, color)
	}
}

func parseHTMLBorder(value string) (int, kos.Color, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, 0, false
	}
	if value == "none" {
		return 0, 0, true
	}
	width := 1
	color := kos.Color(0xC9CFD5)
	seen := false
	for _, token := range strings.Fields(value) {
		if parsed, ok := parseHTMLLength(token); ok {
			width = parsed
			seen = true
			continue
		}
		if token == "solid" || token == "none" {
			seen = true
			continue
		}
		if parsed, ok := parseHTMLColor(token); ok {
			color = parsed
			seen = true
		}
	}
	return width, color, seen
}

func parseHTMLBoxLengths(value string) ([]int, bool) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return nil, false
	}
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		if strings.EqualFold(part, "auto") {
			values = append(values, 0)
			continue
		}
		parsed, ok := parseHTMLLength(part)
		if !ok {
			return nil, false
		}
		values = append(values, parsed)
	}
	return values, true
}

func parseHTMLLength(value string) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, false
	}
	switch value {
	case "0", "0px", "none":
		return 0, true
	case "auto", "100%":
		return 0, false
	}
	value = strings.TrimSuffix(value, "px")
	if strings.Contains(value, ".") {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return int(floatValue + 0.5), true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseHTMLOpacity(value string) (float64, bool) {
	raw := strings.TrimSpace(value)
	isPercent := strings.HasSuffix(raw, "%")
	value = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if isPercent {
		parsed = parsed / 100
	}
	return parsed, true
}

func parseHTMLColor(value string) (kos.Color, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, false
	}
	if named, ok := htmlNamedColors[value]; ok {
		return named, true
	}
	if strings.HasPrefix(value, "#") {
		hex := value[1:]
		switch len(hex) {
		case 3:
			r, okR := parseHTMLHexNibble(hex[0])
			g, okG := parseHTMLHexNibble(hex[1])
			b, okB := parseHTMLHexNibble(hex[2])
			if !okR || !okG || !okB {
				return 0, false
			}
			return kos.Color(uint32(r)<<20 | uint32(r)<<16 | uint32(g)<<12 | uint32(g)<<8 | uint32(b)<<4 | uint32(b)), true
		case 6:
			parsed, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return 0, false
			}
			return kos.Color(parsed), true
		}
	}
	return 0, false
}

func parseHTMLHexNibble(value byte) (uint8, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func parseHTMLFontPath(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		family := strings.Trim(strings.TrimSpace(strings.ToLower(part)), `"'`)
		switch family {
		case "ui-sans", "sans", "sans-serif", "system-ui", "tagix-sans", "opensans":
			return webSansFontPath
		case "ui-mono", "mono", "monospace", "tagix-mono", "robotomono":
			return webMonoFontPath
		}
		if strings.HasSuffix(family, ".ttf") {
			return strings.Trim(strings.TrimSpace(part), `"'`)
		}
	}
	return ""
}

func lastStyleToken(value string) string {
	tokens := strings.Fields(strings.TrimSpace(value))
	if len(tokens) == 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

func applyShellFrameTemplate(app *App, node *Node, ctx *shellRenderContext) {
	if app == nil || node == nil || app.pageFrame == nil || app.pageView == nil {
		return
	}
	if src := strings.TrimSpace(attrValue(node, "src")); src != "" {
		app.startupURL = normalizeURL(src)
	}
	template := ui.Style{}
	applyShellNodeStyles(&template, node, ctx)

	if display, ok := template.GetDisplay(); ok {
		app.pageFrame.Style.SetDisplay(display)
	}
	if margin, ok := template.GetMargin(); ok {
		if ctx != nil && ctx.root != nil && !app.webViewMode {
			hostRoot := ui.Style{}
			applyShellNodeStyles(&hostRoot, ctx.root, ctx)
			if padding, paddingOK := hostRoot.GetPadding(); paddingOK {
				margin.Left += padding.Left
				margin.Right += padding.Right
			}
		}
		app.pageFrame.Style.SetMargin(margin.Top, margin.Right, margin.Bottom, margin.Left)
	}
	if padding, ok := template.GetPadding(); ok {
		app.pageFrame.Style.SetPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
	}
	if border, ok := template.GetBorderWidth(); ok {
		color, colorOK := template.GetBorderColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorder(border, color)
	}
	if border, ok := template.GetBorderTopWidth(); ok {
		color, colorOK := template.GetBorderTopColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderTop(border, color)
	}
	if border, ok := template.GetBorderRightWidth(); ok {
		color, colorOK := template.GetBorderRightColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderRight(border, color)
	}
	if border, ok := template.GetBorderBottomWidth(); ok {
		color, colorOK := template.GetBorderBottomColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderBottom(border, color)
	}
	if border, ok := template.GetBorderLeftWidth(); ok {
		color, colorOK := template.GetBorderLeftColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderLeft(border, color)
	}
	if radius, ok := template.GetBorderRadius(); ok {
		app.pageFrame.Style.SetBorderRadius(radius.TopLeft, radius.TopRight, radius.BottomRight, radius.BottomLeft)
	}
	if color, ok := template.GetBackground(); ok {
		app.pageFrame.Style.SetBackground(color)
		app.pageView.Style.SetBackground(color)
	}
	if opacity, ok := template.GetOpacity(); ok {
		app.pageFrame.Style.SetOpacity(opacity)
	}
	if overflow, ok := template.GetOverflow(); ok {
		app.pageView.Style.SetOverflow(overflow)
	}
	if overflowX, ok := template.GetOverflowX(); ok {
		app.pageView.Style.SetOverflowX(overflowX)
	}
	if overflowY, ok := template.GetOverflowY(); ok {
		app.pageView.Style.SetOverflowY(overflowY)
	}
	if minHeight, ok := template.GetMinHeight(); ok && minHeight > 0 {
		app.pageMinHeight = minHeight
		app.pageView.Style.SetMinHeight(minHeight)
	}
	if height, ok := template.GetHeight(); ok && height > 0 {
		if height > app.pageMinHeight {
			app.pageMinHeight = height
		}
	}
	if maxHeight, ok := template.GetMaxHeight(); ok && maxHeight > 0 {
		app.pageView.Style.SetMaxHeight(maxHeight)
	}
	if width, ok := template.GetMinWidth(); ok && width > 0 {
		app.pageFrame.Style.SetMinWidth(width)
	}
	if width, ok := template.GetMaxWidth(); ok && width > 0 {
		app.pageFrame.Style.SetMaxWidth(width)
	}
}
