package main

import (
	"sort"
	"strconv"
	"strings"
	"ui"
)

const (
	defaultPageFontSize   = 16
	defaultPageLineHeight = 24
	defaultBodyMargin     = 8
)

type cssLayoutContext struct {
	viewportWidth  int
	viewportHeight int
	fontSize       int
}

type cssLengthValue struct {
	pixels int
	auto   bool
}

type cssSelector struct {
	tag     string
	id      string
	classes []string
	pseudo  string
}

type cssRule struct {
	selector     cssSelector
	declarations string
	specificity  int
	order        int
}

type pageStylesheet struct {
	rules []cssRule
}

func (ctx *renderContext) cssLayoutContext() cssLayoutContext {
	if ctx == nil {
		return cssLayoutContext{fontSize: defaultPageFontSize}
	}
	return cssLayoutContext{
		viewportWidth:  ctx.viewportWidth,
		viewportHeight: ctx.viewportHeight,
		fontSize:       defaultPageFontSize,
	}
}

func parseDocumentStylesheet(doc *Document) *pageStylesheet {
	if doc == nil {
		return nil
	}
	styleNodes := doc.GetElementsByTagName("style")
	if len(styleNodes) == 0 {
		return nil
	}
	rules := make([]cssRule, 0, len(styleNodes)*4)
	order := 0
	for _, node := range styleNodes {
		source := strings.TrimSpace(collectText(node))
		if source == "" {
			continue
		}
		rules = append(rules, parseCSSRules(source, &order)...)
	}
	if len(rules) == 0 {
		return nil
	}
	return &pageStylesheet{rules: rules}
}

func parseCSSRules(source string, order *int) []cssRule {
	source = stripCSSComments(source)
	if source == "" {
		return nil
	}
	rules := make([]cssRule, 0, 8)
	for _, block := range strings.Split(source, "}") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		brace := strings.IndexByte(block, '{')
		if brace <= 0 || brace+1 >= len(block) {
			continue
		}
		selectors := strings.TrimSpace(block[:brace])
		declarations := strings.TrimSpace(block[brace+1:])
		if selectors == "" || declarations == "" {
			continue
		}
		for _, rawSelector := range strings.Split(selectors, ",") {
			selector, ok := parseCSSSelector(rawSelector)
			if !ok {
				continue
			}
			rules = append(rules, cssRule{
				selector:     selector,
				declarations: declarations,
				specificity:  selector.specificity(),
				order:        *order,
			})
			*order++
		}
	}
	return rules
}

func stripCSSComments(source string) string {
	if source == "" {
		return ""
	}
	var builder strings.Builder
	for {
		start := strings.Index(source, "/*")
		if start < 0 {
			builder.WriteString(source)
			break
		}
		builder.WriteString(source[:start])
		source = source[start+2:]
		end := strings.Index(source, "*/")
		if end < 0 {
			break
		}
		source = source[end+2:]
	}
	return builder.String()
}

func parseCSSSelector(source string) (cssSelector, bool) {
	selector := cssSelector{}
	source = strings.TrimSpace(source)
	if source == "" {
		return selector, false
	}
	if strings.ContainsAny(source, " >+~[") {
		return selector, false
	}
	for len(source) > 0 {
		switch source[0] {
		case '*':
			source = source[1:]
		case '#':
			source = source[1:]
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return selector, false
			}
			selector.id = token
			source = rest
		case '.':
			source = source[1:]
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return selector, false
			}
			selector.classes = append(selector.classes, token)
			source = rest
		case ':':
			source = source[1:]
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return selector, false
			}
			selector.pseudo = toLowerASCII(token)
			source = rest
		default:
			if selector.tag != "" {
				return selector, false
			}
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return selector, false
			}
			selector.tag = toLowerASCII(token)
			source = rest
		}
	}
	if selector.tag == "" && selector.id == "" && len(selector.classes) == 0 && selector.pseudo == "" {
		return selector, false
	}
	return selector, true
}

func readCSSIdent(source string) (string, string, bool) {
	if source == "" || !isCSSIdentStart(source[0]) {
		return "", source, false
	}
	index := 1
	for index < len(source) && isCSSIdentPart(source[index]) {
		index++
	}
	return source[:index], source[index:], true
}

func isCSSIdentStart(value byte) bool {
	return (value >= 'a' && value <= 'z') ||
		(value >= 'A' && value <= 'Z') ||
		value == '_' || value == '-'
}

func isCSSIdentPart(value byte) bool {
	return isCSSIdentStart(value) || (value >= '0' && value <= '9')
}

func (selector cssSelector) specificity() int {
	score := 0
	if selector.id != "" {
		score += 100
	}
	score += len(selector.classes) * 10
	if selector.pseudo != "" {
		score += 10
	}
	if selector.tag != "" {
		score++
	}
	return score
}

func (selector cssSelector) matches(node *Node) bool {
	if node == nil || node.Type != ElementNode {
		return false
	}
	if selector.tag != "" && node.Tag != selector.tag {
		return false
	}
	if selector.id != "" && attrValue(node, "id") != selector.id {
		return false
	}
	if len(selector.classes) > 0 {
		classAttr := attrValue(node, "class")
		for _, className := range selector.classes {
			if !classListContains(classAttr, className) {
				return false
			}
		}
	}
	switch selector.pseudo {
	case "":
		return true
	case "link", "visited", "any-link":
		return node.Tag == "a" && attrValue(node, "href") != ""
	case "root":
		return node.Tag == "html"
	default:
		return false
	}
}

func (sheet *pageStylesheet) apply(style *ui.Style, node *Node, layout cssLayoutContext) {
	if sheet == nil || style == nil || node == nil || len(sheet.rules) == 0 {
		return
	}
	matched := make([]cssRule, 0, 8)
	for _, rule := range sheet.rules {
		if rule.selector.matches(node) {
			matched = append(matched, rule)
		}
	}
	if len(matched) == 0 {
		return
	}
	sort.SliceStable(matched, func(i int, j int) bool {
		if matched[i].specificity != matched[j].specificity {
			return matched[i].specificity < matched[j].specificity
		}
		return matched[i].order < matched[j].order
	})
	for _, rule := range matched {
		applyCSSDeclarations(style, rule.declarations, layout)
	}
}

func applyPageNodeStyles(style *ui.Style, node *Node, ctx *renderContext) {
	if style == nil || node == nil {
		return
	}
	layout := ctx.cssLayoutContext()
	if ctx != nil && ctx.stylesheet != nil {
		ctx.stylesheet.apply(style, node, layout)
	}
	if inline := attrValue(node, "style"); inline != "" {
		applyCSSDeclarations(style, inline, layout)
	}
}

func applyPageTextProperties(style *ui.Style, node *Node, ctx *renderContext) {
	if style == nil || node == nil {
		return
	}
	resolved := ui.Style{}
	applyPageNodeStyles(&resolved, node, ctx)
	copyPageTextProperties(style, resolved)
}

func copyPageTextProperties(target *ui.Style, source ui.Style) {
	if target == nil {
		return
	}
	if color, ok := source.GetForeground(); ok {
		target.SetForeground(color)
	}
	if path, ok := source.GetFontPath(); ok {
		target.SetFontPath(path)
	}
	if size, ok := source.GetFontSize(); ok {
		target.SetFontSize(size)
	}
	if lineHeight, ok := source.GetLineHeight(); ok {
		target.SetLineHeight(lineHeight)
	}
	if decoration, ok := source.GetTextDecoration(); ok {
		target.SetTextDecoration(decoration)
	}
	if align, ok := source.GetTextAlign(); ok {
		target.SetTextAlign(align)
	}
	if whiteSpace, ok := source.GetWhiteSpace(); ok {
		target.SetWhiteSpace(whiteSpace)
	}
}

func applyPageCanvasStyles(style *ui.Style, doc *Document, ctx *renderContext) {
	if style == nil || doc == nil {
		return
	}
	for _, tag := range []string{"html", "body"} {
		nodes := doc.GetElementsByTagName(tag)
		if len(nodes) == 0 {
			continue
		}
		resolved := ui.Style{}
		applyPageNodeStyles(&resolved, nodes[0], ctx)
		if color, ok := resolved.GetBackground(); ok {
			style.SetBackground(color)
		}
		if color, ok := resolved.GetForeground(); ok {
			style.SetForeground(color)
		}
		if path, ok := resolved.GetFontPath(); ok {
			style.SetFontPath(path)
		}
		if size, ok := resolved.GetFontSize(); ok {
			style.SetFontSize(size)
		}
		if lineHeight, ok := resolved.GetLineHeight(); ok {
			style.SetLineHeight(lineHeight)
		}
	}
}

func documentCanvasStyle(doc *Document, viewportWidth int, viewportHeight int) ui.Style {
	style := ui.Style{}
	style.SetBackground(ui.White)
	style.SetForeground(0x333333)
	style.SetFontPath(webSansFontPath)
	style.SetFontSize(defaultPageFontSize)
	style.SetLineHeight(defaultPageLineHeight)
	ctx := &renderContext{
		stylesheet:     parseDocumentStylesheet(doc),
		viewportWidth:  viewportWidth,
		viewportHeight: viewportHeight,
	}
	applyPageCanvasStyles(&style, doc, ctx)
	return style
}

func applyCSSDeclarations(style *ui.Style, declarations string, layout cssLayoutContext) {
	if style == nil {
		return
	}
	layout = normalizeCSSLayoutContext(layout)
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
		applyCSSDeclaration(style, name, value, &layout)
	}
}

func normalizeCSSLayoutContext(layout cssLayoutContext) cssLayoutContext {
	if layout.fontSize <= 0 {
		layout.fontSize = defaultPageFontSize
	}
	return layout
}

func applyCSSDeclaration(style *ui.Style, name string, value string, layout *cssLayoutContext) {
	if style == nil || layout == nil {
		return
	}
	switch name {
	case "width":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetWidth(parsed)
			return
		}
	case "height":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetHeight(parsed)
			return
		}
	case "min-width":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMinWidth(parsed)
			return
		}
	case "max-width":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMaxWidth(parsed)
			return
		}
	case "min-height":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMinHeight(parsed)
			return
		}
	case "max-height":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMaxHeight(parsed)
			return
		}
	case "margin":
		if applyCSSBoxSpacing(style, value, *layout, true) {
			return
		}
	case "padding":
		if applyCSSBoxSpacing(style, value, *layout, false) {
			return
		}
	case "margin-top":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMarginTop(parsed)
			return
		}
	case "margin-right":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMarginRight(parsed)
			return
		}
	case "margin-bottom":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMarginBottom(parsed)
			return
		}
	case "margin-left":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetMarginLeft(parsed)
			return
		}
	case "padding-top":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetPaddingTop(parsed)
			return
		}
	case "padding-right":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetPaddingRight(parsed)
			return
		}
	case "padding-bottom":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetPaddingBottom(parsed)
			return
		}
	case "padding-left":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetPaddingLeft(parsed)
			return
		}
	case "font-size":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetFontSize(parsed)
			layout.fontSize = parsed
			return
		}
	case "line-height":
		if parsed, ok := parseCSSLength(value, *layout); ok {
			style.SetLineHeight(parsed)
			return
		}
	}
	applyInlineStyleRule(style, name, value)
}

func applyCSSBoxSpacing(style *ui.Style, value string, layout cssLayoutContext, margin bool) bool {
	values, ok := parseCSSBoxValues(value, layout)
	if !ok {
		return false
	}
	if margin {
		values = resolveCSSAutoMargins(style, values, layout)
		style.SetMargin(values[0].pixels, values[1].pixels, values[2].pixels, values[3].pixels)
		return true
	}
	style.SetPadding(values[0].pixels, values[1].pixels, values[2].pixels, values[3].pixels)
	return true
}

func parseCSSBoxValues(value string, layout cssLayoutContext) ([4]cssLengthValue, bool) {
	values := [4]cssLengthValue{}
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 || len(parts) > 4 {
		return values, false
	}
	parsed := make([]cssLengthValue, 0, len(parts))
	for _, part := range parts {
		item, ok := parseCSSLengthValue(part, layout)
		if !ok {
			return values, false
		}
		parsed = append(parsed, item)
	}
	switch len(parsed) {
	case 1:
		values[0] = parsed[0]
		values[1] = parsed[0]
		values[2] = parsed[0]
		values[3] = parsed[0]
	case 2:
		values[0] = parsed[0]
		values[1] = parsed[1]
		values[2] = parsed[0]
		values[3] = parsed[1]
	case 3:
		values[0] = parsed[0]
		values[1] = parsed[1]
		values[2] = parsed[2]
		values[3] = parsed[1]
	case 4:
		copy(values[:], parsed)
	}
	return values, true
}

func parseCSSLengthValue(value string, layout cssLayoutContext) (cssLengthValue, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return cssLengthValue{}, false
	}
	if value == "auto" {
		return cssLengthValue{auto: true}, true
	}
	parsed, ok := parseCSSLength(value, layout)
	if !ok {
		return cssLengthValue{}, false
	}
	return cssLengthValue{pixels: parsed}, true
}

func resolveCSSAutoMargins(style *ui.Style, values [4]cssLengthValue, layout cssLayoutContext) [4]cssLengthValue {
	if !values[1].auto && !values[3].auto {
		for i := range values {
			if values[i].auto {
				values[i].pixels = 0
				values[i].auto = false
			}
		}
		return values
	}
	width, ok := style.GetWidth()
	if !ok || width < 0 || layout.viewportWidth <= 0 {
		for i := range values {
			if values[i].auto {
				values[i].pixels = 0
				values[i].auto = false
			}
		}
		return values
	}
	left := values[3].pixels
	right := values[1].pixels
	remaining := layout.viewportWidth - width - left - right
	if remaining < 0 {
		remaining = 0
	}
	switch {
	case values[1].auto && values[3].auto:
		values[3].pixels = remaining / 2
		values[1].pixels = remaining - values[3].pixels
	case values[3].auto:
		values[3].pixels = remaining
	case values[1].auto:
		values[1].pixels = remaining
	}
	for i := range values {
		values[i].auto = false
	}
	return values
}

func parseCSSLength(value string, layout cssLayoutContext) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, false
	}
	switch value {
	case "0", "0px", "0em", "0rem", "0vw", "0vh", "none":
		return 0, true
	case "auto", "normal", "inherit", "initial", "unset":
		return 0, false
	}
	amount := value
	unit := ""
	for _, suffix := range []string{"rem", "px", "em", "vw", "vh"} {
		if strings.HasSuffix(amount, suffix) {
			amount = strings.TrimSpace(strings.TrimSuffix(amount, suffix))
			unit = suffix
			break
		}
	}
	if strings.HasSuffix(amount, "%") {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case "", "px":
		return roundCSSPixels(parsed), true
	case "em":
		return roundCSSPixels(parsed * float64(layout.fontSize)), true
	case "rem":
		return roundCSSPixels(parsed * float64(defaultPageFontSize)), true
	case "vw":
		if layout.viewportWidth <= 0 {
			return 0, false
		}
		return roundCSSPixels(parsed * float64(layout.viewportWidth) / 100), true
	case "vh":
		if layout.viewportHeight <= 0 {
			return 0, false
		}
		return roundCSSPixels(parsed * float64(layout.viewportHeight) / 100), true
	default:
		return 0, false
	}
}

func roundCSSPixels(value float64) int {
	if value < 0 {
		return int(value - 0.5)
	}
	return int(value + 0.5)
}
