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

type cssSimpleSelector struct {
	tag       string
	id        string
	classes   []string
	pseudos   []string
	universal bool
}

type cssSelectorStep struct {
	simple     cssSimpleSelector
	combinator byte
}

type cssSelector struct {
	steps []cssSelectorStep
}

type cssMediaCondition struct {
	minWidth int
	maxWidth int
	hasMin   bool
	hasMax   bool
}

type cssRule struct {
	selector     cssSelector
	declarations string
	specificity  int
	order        int
	media        cssMediaCondition
}

type pageStylesheet struct {
	rules []cssRule
	cache map[string]ui.Style
}

type cssAutoMargins struct {
	left  bool
	right bool
}

type cssListStyleState struct {
	listStyleType string
	hasType       bool
}

type cssBackgroundState struct {
	imageURL      string
	hasImage      bool
	repeat        string
	hasRepeat     bool
	attachment    ui.BackgroundAttachment
	hasAttachment bool
}

type cssVerticalAlignState struct {
	value    string
	hasValue bool
}

type cssPseudoState uint8

const (
	cssPseudoStateHover cssPseudoState = 1 << iota
	cssPseudoStateActive
	cssPseudoStateFocus
)

type cssStyleVariant uint8

const (
	cssStyleVariantBase cssStyleVariant = iota
	cssStyleVariantHover
	cssStyleVariantActive
	cssStyleVariantFocus
)

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
	if doc.stylesheetParsed {
		return doc.stylesheet
	}
	doc.stylesheetParsed = true
	styleNodes := doc.GetElementsByTagName("style")
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
		doc.stylesheet = &pageStylesheet{
			cache: map[string]ui.Style{},
		}
		return doc.stylesheet
	}
	doc.stylesheet = &pageStylesheet{
		rules: rules,
		cache: map[string]ui.Style{},
	}
	return doc.stylesheet
}

func cssResolvedStyleCacheKey(node *Node, layout cssLayoutContext, variant cssStyleVariant) string {
	if node == nil {
		return ""
	}
	return strconv.Itoa(node.ID) +
		"|" + strconv.Itoa(layout.viewportWidth) +
		"|" + strconv.Itoa(layout.viewportHeight) +
		"|" + strconv.Itoa(layout.fontSize) +
		"|v" + strconv.Itoa(int(variant))
}

func parseCSSRules(source string, order *int) []cssRule {
	source = stripCSSComments(source)
	if source == "" {
		return nil
	}
	return parseCSSRulesWithMedia(source, order, cssMediaCondition{})
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
	if strings.ContainsAny(source, "~[") {
		return selector, false
	}
	combinator := byte(0)
	for len(source) > 0 {
		source = strings.TrimLeft(source, " \t\r\n")
		if source == "" {
			break
		}
		step, rest, ok := parseCSSSelectorStep(source)
		if !ok {
			return selector, false
		}
		step.combinator = combinator
		selector.steps = append(selector.steps, step)
		source = rest
		nextCombinator := byte(0)
		sawSpace := false
		for len(source) > 0 {
			if isCSSSpace(source[0]) {
				sawSpace = true
				source = source[1:]
				continue
			}
			if source[0] == '>' || source[0] == '+' {
				nextCombinator = source[0]
				source = source[1:]
				for len(source) > 0 && isCSSSpace(source[0]) {
					source = source[1:]
				}
				break
			}
			if sawSpace {
				nextCombinator = ' '
			}
			break
		}
		if nextCombinator == 0 && sawSpace {
			nextCombinator = ' '
		}
		combinator = nextCombinator
	}
	if len(selector.steps) == 0 {
		return selector, false
	}
	return selector, true
}

func parseCSSSelectorStep(source string) (cssSelectorStep, string, bool) {
	step := cssSelectorStep{}
	source = strings.TrimLeft(source, " \t\r\n")
	if source == "" {
		return step, source, false
	}
	for len(source) > 0 {
		switch source[0] {
		case ' ', '\t', '\r', '\n', '>', '+', ',', '{':
			if step.simple.tag == "" && step.simple.id == "" && len(step.simple.classes) == 0 && len(step.simple.pseudos) == 0 && !step.simple.universal {
				return step, source, false
			}
			return step, source, true
		case '*':
			if step.simple.tag != "" || step.simple.universal {
				return step, source, false
			}
			step.simple.universal = true
			source = source[1:]
		case '#':
			source = source[1:]
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return step, source, false
			}
			step.simple.id = token
			source = rest
		case '.':
			source = source[1:]
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return step, source, false
			}
			step.simple.classes = append(step.simple.classes, token)
			source = rest
		case ':':
			if len(source) > 1 && source[1] == ':' {
				return step, source, false
			}
			source = source[1:]
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return step, source, false
			}
			step.simple.pseudos = append(step.simple.pseudos, normalizeCSSPseudo(token))
			source = rest
		default:
			if step.simple.tag != "" {
				return step, source, false
			}
			token, rest, ok := readCSSIdent(source)
			if !ok {
				return step, source, false
			}
			step.simple.tag = toLowerASCII(token)
			source = rest
		}
	}
	if step.simple.tag == "" && step.simple.id == "" && len(step.simple.classes) == 0 && len(step.simple.pseudos) == 0 && !step.simple.universal {
		return step, source, false
	}
	return step, source, true
}

func normalizeCSSPseudo(token string) string {
	token = toLowerASCII(strings.TrimSpace(token))
	switch token {
	case "action":
		return "active"
	case "focus-visible":
		return "focus"
	default:
		return token
	}
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

func isCSSSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func (selector cssSimpleSelector) specificity() int {
	score := 0
	if selector.id != "" {
		score += 100
	}
	score += len(selector.classes) * 10
	if len(selector.pseudos) > 0 {
		score += len(selector.pseudos) * 10
	}
	if selector.tag != "" {
		score++
	}
	return score
}

func (selector cssSelector) specificity() int {
	score := 0
	for _, step := range selector.steps {
		score += step.simple.specificity()
	}
	return score
}

func (selector cssSelector) matchesStatic(node *Node) bool {
	if node == nil || node.Type != ElementNode || len(selector.steps) == 0 {
		return false
	}
	current := node
	if !selector.steps[len(selector.steps)-1].simple.matchesStatic(current) {
		return false
	}
	for index := len(selector.steps) - 2; index >= 0; index-- {
		combinator := selector.steps[index+1].combinator
		target := selector.steps[index].simple
		switch combinator {
		case '>':
			current = nearestParentElement(current)
			if !target.matchesStatic(current) {
				return false
			}
		case '+':
			current = previousElementSibling(current)
			if !target.matchesStatic(current) {
				return false
			}
		case ' ':
			found := false
			for parent := nearestParentElement(current); parent != nil; parent = nearestParentElement(parent) {
				if target.matchesStatic(parent) {
					current = parent
					found = true
					break
				}
			}
			if !found {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (selector cssSimpleSelector) matchesStatic(node *Node) bool {
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
	for _, pseudo := range selector.pseudos {
		if isCSSInteractivePseudo(pseudo) {
			continue
		}
		if !matchesCSSStructuralPseudo(node, pseudo) {
			return false
		}
	}
	return true
}

func matchesCSSStructuralPseudo(node *Node, pseudo string) bool {
	switch pseudo {
	case "", "focus-visible":
		return true
	case "link", "visited", "any-link":
		return node.Tag == "a" && attrValue(node, "href") != ""
	case "root":
		return node.Tag == "html"
	case "first-child":
		return isFirstElementChild(node)
	case "last-child":
		return isLastElementChild(node)
	case "enabled":
		return !hasAttr(node, "disabled")
	case "disabled":
		return hasAttr(node, "disabled")
	case "checked":
		return hasAttr(node, "checked") || strings.EqualFold(strings.TrimSpace(attrValue(node, "aria-checked")), "true")
	default:
		return false
	}
}

func cssPseudoStateForName(name string) cssPseudoState {
	switch normalizeCSSPseudo(name) {
	case "hover":
		return cssPseudoStateHover
	case "active":
		return cssPseudoStateHover | cssPseudoStateActive
	case "focus":
		return cssPseudoStateFocus
	default:
		return 0
	}
}

func isCSSInteractivePseudo(name string) bool {
	return cssPseudoStateForName(name) != 0
}

func (selector cssSimpleSelector) dynamicMask() cssPseudoState {
	mask := cssPseudoState(0)
	for _, pseudo := range selector.pseudos {
		mask |= cssPseudoStateForName(pseudo)
	}
	return mask
}

func (selector cssSimpleSelector) hasDynamicPseudo() bool {
	return selector.dynamicMask() != 0
}

func (selector cssSelector) hasDynamicPseudo() bool {
	for _, step := range selector.steps {
		if step.simple.hasDynamicPseudo() {
			return true
		}
	}
	return false
}

func (selector cssSelector) hasDynamicPseudoOutsideTerminal() bool {
	if len(selector.steps) <= 1 {
		return false
	}
	for _, step := range selector.steps[:len(selector.steps)-1] {
		if step.simple.hasDynamicPseudo() {
			return true
		}
	}
	return false
}

func (selector cssSelector) dynamicTerminalMask() cssPseudoState {
	if len(selector.steps) == 0 {
		return 0
	}
	return selector.steps[len(selector.steps)-1].simple.dynamicMask()
}

func cssVariantStateMask(variant cssStyleVariant) cssPseudoState {
	switch variant {
	case cssStyleVariantHover:
		return cssPseudoStateHover
	case cssStyleVariantActive:
		return cssPseudoStateHover | cssPseudoStateActive
	case cssStyleVariantFocus:
		return cssPseudoStateFocus
	default:
		return 0
	}
}

func parseCSSRulesWithMedia(source string, order *int, media cssMediaCondition) []cssRule {
	rules := make([]cssRule, 0, 8)
	source = strings.TrimSpace(source)
	for source != "" {
		header, block, rest, ok := consumeCSSRuleBlock(source)
		if !ok {
			break
		}
		source = strings.TrimSpace(rest)
		header = strings.TrimSpace(header)
		block = strings.TrimSpace(block)
		if header == "" || block == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(header), "@media") {
			query := strings.TrimSpace(header[len("@media"):])
			condition, ok := parseCSSMediaCondition(query)
			if !ok {
				continue
			}
			rules = append(rules, parseCSSRulesWithMedia(block, order, media.merge(condition))...)
			continue
		}
		for _, rawSelector := range strings.Split(header, ",") {
			selector, ok := parseCSSSelector(rawSelector)
			if !ok {
				continue
			}
			rules = append(rules, cssRule{
				selector:     selector,
				declarations: block,
				specificity:  selector.specificity(),
				order:        *order,
				media:        media,
			})
			*order++
		}
	}
	return rules
}

func parseCSSAutoMargins(declarations string, current cssAutoMargins) cssAutoMargins {
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
		switch name {
		case "margin":
			values, ok := parseCSSBoxValues(value, cssLayoutContext{})
			if !ok {
				continue
			}
			current.right = values[1].auto
			current.left = values[3].auto
		case "margin-left":
			item, ok := parseCSSLengthValue(value, cssLayoutContext{})
			if !ok {
				continue
			}
			current.left = item.auto
		case "margin-right":
			item, ok := parseCSSLengthValue(value, cssLayoutContext{})
			if !ok {
				continue
			}
			current.right = item.auto
		}
	}
	return current
}

func parseCSSListStyleState(declarations string, current cssListStyleState) cssListStyleState {
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
		switch name {
		case "list-style-type":
			if normalized, ok := normalizeCSSListStyleType(value); ok {
				current.listStyleType = normalized
				current.hasType = true
			}
		case "list-style":
			for _, token := range strings.Fields(strings.ToLower(value)) {
				if strings.HasPrefix(token, "url(") || isCSSListStylePositionToken(token) {
					continue
				}
				if normalized, ok := normalizeCSSListStyleType(token); ok {
					current.listStyleType = normalized
					current.hasType = true
				}
			}
		}
	}
	return current
}

func normalizeCSSListStyleType(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "none",
		"disc",
		"circle",
		"square",
		"decimal",
		"decimal-leading-zero",
		"lower-alpha",
		"upper-alpha",
		"lower-latin",
		"upper-latin",
		"lower-roman",
		"upper-roman":
		return value, true
	default:
		return "", false
	}
}

func isCSSListStylePositionToken(value string) bool {
	switch value {
	case "inside", "outside":
		return true
	default:
		return false
	}
}

func parseCSSBackgroundState(declarations string, current cssBackgroundState) cssBackgroundState {
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
		switch name {
		case "background-image":
			if imageURL, ok := parseCSSBackgroundImageURL(value); ok {
				current.imageURL = imageURL
				current.hasImage = true
			} else if strings.EqualFold(strings.TrimSpace(value), "none") {
				current.imageURL = ""
				current.hasImage = true
			}
		case "background-repeat":
			if repeat, ok := normalizeCSSBackgroundRepeat(value); ok {
				current.repeat = repeat
				current.hasRepeat = true
			}
		case "background-attachment":
			if attachment, ok := ui.ParseBackgroundAttachment(value); ok {
				current.attachment = attachment
				current.hasAttachment = true
			}
		case "background":
			for _, token := range splitStyleValueTokens(value) {
				if token == "" {
					continue
				}
				if imageURL, ok := parseCSSBackgroundImageURL(token); ok {
					current.imageURL = imageURL
					current.hasImage = true
					continue
				}
				if repeat, ok := normalizeCSSBackgroundRepeat(token); ok {
					current.repeat = repeat
					current.hasRepeat = true
					continue
				}
				if attachment, ok := ui.ParseBackgroundAttachment(token); ok {
					current.attachment = attachment
					current.hasAttachment = true
				}
			}
		}
	}
	return current
}

func parseCSSVerticalAlignState(declarations string, current cssVerticalAlignState) cssVerticalAlignState {
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
		if name != "vertical-align" {
			continue
		}
		value := strings.TrimSpace(chunk[colon+1:])
		if normalized, ok := normalizeCSSVerticalAlign(value); ok {
			current.value = normalized
			current.hasValue = true
		}
	}
	return current
}

func parseCSSBackgroundImageURL(value string) (string, bool) {
	imageURL, ok := extractCSSURLValue(value)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(imageURL), imageURL != ""
}

func normalizeCSSBackgroundRepeat(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "repeat", "repeat-x", "repeat-y", "no-repeat":
		return strings.ToLower(strings.TrimSpace(value)), true
	default:
		return "", false
	}
}

func normalizeCSSVerticalAlign(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "top", "text-top":
		return "top", true
	case "middle", "center":
		return "middle", true
	case "bottom", "text-bottom":
		return "bottom", true
	default:
		return "", false
	}
}

func consumeCSSRuleBlock(source string) (string, string, string, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", "", "", false
	}
	brace := strings.IndexByte(source, '{')
	if brace <= 0 || brace+1 >= len(source) {
		return "", "", "", false
	}
	depth := 1
	index := brace + 1
	for index < len(source) {
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[:brace], source[brace+1 : index], source[index+1:], true
			}
		}
		index++
	}
	return "", "", "", false
}

func parseCSSMediaCondition(query string) (cssMediaCondition, bool) {
	condition := cssMediaCondition{}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return condition, false
	}
	for _, part := range strings.Split(query, "and") {
		part = strings.TrimSpace(strings.Trim(part, "()"))
		if part == "" {
			continue
		}
		colon := strings.IndexByte(part, ':')
		if colon <= 0 || colon+1 >= len(part) {
			continue
		}
		name := strings.TrimSpace(part[:colon])
		value := strings.TrimSpace(part[colon+1:])
		width, ok := parseCSSLength(value, cssLayoutContext{viewportWidth: 0, viewportHeight: 0, fontSize: defaultPageFontSize})
		if !ok {
			continue
		}
		switch name {
		case "max-width":
			condition.maxWidth = width
			condition.hasMax = true
		case "min-width":
			condition.minWidth = width
			condition.hasMin = true
		}
	}
	return condition, condition.hasMin || condition.hasMax
}

func (condition cssMediaCondition) merge(other cssMediaCondition) cssMediaCondition {
	merged := condition
	if other.hasMin && (!merged.hasMin || other.minWidth > merged.minWidth) {
		merged.minWidth = other.minWidth
		merged.hasMin = true
	}
	if other.hasMax && (!merged.hasMax || other.maxWidth < merged.maxWidth) {
		merged.maxWidth = other.maxWidth
		merged.hasMax = true
	}
	return merged
}

func (condition cssMediaCondition) matches(layout cssLayoutContext) bool {
	width := layout.viewportWidth
	if width <= 0 {
		return true
	}
	if condition.hasMin && width < condition.minWidth {
		return false
	}
	if condition.hasMax && width > condition.maxWidth {
		return false
	}
	return true
}

func nearestParentElement(node *Node) *Node {
	for current := node; current != nil; current = current.Parent {
		if current.Parent == nil {
			return nil
		}
		if current.Parent.Type == ElementNode {
			return current.Parent
		}
	}
	return nil
}

func previousElementSibling(node *Node) *Node {
	if node == nil || node.Parent == nil {
		return nil
	}
	children := node.Parent.Children
	for index := range children {
		if children[index] != node {
			continue
		}
		for prev := index - 1; prev >= 0; prev-- {
			if children[prev] != nil && children[prev].Type == ElementNode {
				return children[prev]
			}
		}
		break
	}
	return nil
}

func isFirstElementChild(node *Node) bool {
	if node == nil || node.Parent == nil {
		return false
	}
	for _, child := range node.Parent.Children {
		if child == nil || child.Type != ElementNode {
			continue
		}
		return child == node
	}
	return false
}

func isLastElementChild(node *Node) bool {
	if node == nil || node.Parent == nil {
		return false
	}
	for index := len(node.Parent.Children) - 1; index >= 0; index-- {
		child := node.Parent.Children[index]
		if child == nil || child.Type != ElementNode {
			continue
		}
		return child == node
	}
	return false
}

func (sheet *pageStylesheet) resolvedStyle(node *Node, layout cssLayoutContext, variant cssStyleVariant) ui.Style {
	if sheet == nil || node == nil {
		return ui.Style{}
	}
	if sheet.cache == nil {
		sheet.cache = map[string]ui.Style{}
	}
	key := cssResolvedStyleCacheKey(node, layout, variant)
	if cached, ok := sheet.cache[key]; ok {
		return cached
	}
	style := ui.Style{}
	stateMask := cssVariantStateMask(variant)
	if len(sheet.rules) > 0 {
		matched := make([]cssRule, 0, 8)
		for _, rule := range sheet.rules {
			if !rule.media.matches(layout) {
				continue
			}
			if !rule.selector.matchesStatic(node) {
				continue
			}
			switch variant {
			case cssStyleVariantBase:
				if rule.selector.hasDynamicPseudo() {
					continue
				}
			default:
				if rule.selector.hasDynamicPseudoOutsideTerminal() {
					continue
				}
				required := rule.selector.dynamicTerminalMask()
				if required == 0 || required&^stateMask != 0 {
					continue
				}
			}
			matched = append(matched, rule)
		}
		if len(matched) > 0 {
			sort.SliceStable(matched, func(i int, j int) bool {
				if matched[i].specificity != matched[j].specificity {
					return matched[i].specificity < matched[j].specificity
				}
				return matched[i].order < matched[j].order
			})
			for _, rule := range matched {
				applyCSSDeclarations(&style, rule.declarations, layout)
			}
		}
	}
	if variant == cssStyleVariantBase {
		if inline := attrValue(node, "style"); inline != "" {
			applyCSSDeclarations(&style, inline, layout)
		}
	}
	sheet.cache[key] = style
	return style
}

func (sheet *pageStylesheet) apply(style *ui.Style, node *Node, layout cssLayoutContext) {
	if sheet == nil || style == nil || node == nil {
		return
	}
	applyResolvedStyle(style, sheet.resolvedStyle(node, layout, cssStyleVariantBase))
}

func mergeResolvedStyle(target ui.Style, source ui.Style) ui.Style {
	applyResolvedStyle(&target, source)
	return target
}

func (sheet *pageStylesheet) interactionStyles(node *Node, layout cssLayoutContext) (ui.Style, ui.Style, ui.Style) {
	if sheet == nil || node == nil {
		return ui.Style{}, ui.Style{}, ui.Style{}
	}
	return sheet.resolvedStyle(node, layout, cssStyleVariantHover),
		sheet.resolvedStyle(node, layout, cssStyleVariantActive),
		sheet.resolvedStyle(node, layout, cssStyleVariantFocus)
}

func applyPageInteractionStyles(target *ui.DocumentNode, node *Node, ctx *renderContext) {
	if target == nil || node == nil || ctx == nil || ctx.stylesheet == nil {
		return
	}
	layout := ctx.cssLayoutContext()
	hover, active, focus := ctx.stylesheet.interactionStyles(node, layout)
	if !hover.IsZero() {
		target.StyleHover = mergeResolvedStyle(target.StyleHover, hover)
	}
	if !active.IsZero() {
		target.StyleActive = mergeResolvedStyle(target.StyleActive, active)
	}
	if !focus.IsZero() {
		target.StyleFocus = mergeResolvedStyle(target.StyleFocus, focus)
	}
}

func applyShellInteractionStyles(target *ui.DocumentNode, node *Node, ctx *shellRenderContext) {
	if target == nil || node == nil || ctx == nil || ctx.stylesheet == nil {
		return
	}
	hover, active, focus := ctx.stylesheet.interactionStyles(node, ctx.layout)
	if !hover.IsZero() {
		target.StyleHover = mergeResolvedStyle(target.StyleHover, hover)
	}
	if !active.IsZero() {
		target.StyleActive = mergeResolvedStyle(target.StyleActive, active)
	}
	if !focus.IsZero() {
		target.StyleFocus = mergeResolvedStyle(target.StyleFocus, focus)
	}
}

func applyStateTextPropertiesToTextDescendants(node *ui.DocumentNode, hover ui.Style, active ui.Style, focus ui.Style) {
	if node == nil {
		return
	}
	if node.Kind == ui.DocumentNodeText {
		if !hover.IsZero() {
			updated := node.StyleHover
			copyPageTextProperties(&updated, hover)
			node.StyleHover = updated
		}
		if !active.IsZero() {
			updated := node.StyleActive
			copyPageTextProperties(&updated, active)
			node.StyleActive = updated
		}
		if !focus.IsZero() {
			updated := node.StyleFocus
			copyPageTextProperties(&updated, focus)
			node.StyleFocus = updated
		}
		return
	}
	if node.Focusable || !node.StyleHover.IsZero() || !node.StyleActive.IsZero() || !node.StyleFocus.IsZero() {
		return
	}
	for _, child := range node.Children {
		applyStateTextPropertiesToTextDescendants(child, hover, active, focus)
	}
}

func applyStateTextPropertiesToTextNodes(nodes []*ui.DocumentNode, hover ui.Style, active ui.Style, focus ui.Style) {
	for _, node := range nodes {
		applyStateTextPropertiesToTextDescendants(node, hover, active, focus)
	}
}

func applyPageInteractionTextStyles(target *ui.DocumentNode) {
	if target == nil {
		return
	}
	applyStateTextPropertiesToTextNodes(target.Children, target.StyleHover, target.StyleActive, target.StyleFocus)
}

func applyShellInteractionTextStyles(target *ui.DocumentNode) {
	if target == nil {
		return
	}
	applyStateTextPropertiesToTextNodes(target.Children, target.StyleHover, target.StyleActive, target.StyleFocus)
}

func (sheet *pageStylesheet) autoMargins(node *Node, layout cssLayoutContext) cssAutoMargins {
	state := cssAutoMargins{}
	if sheet == nil || node == nil {
		return state
	}
	if len(sheet.rules) > 0 {
		matched := make([]cssRule, 0, 8)
		for _, rule := range sheet.rules {
			if !rule.media.matches(layout) {
				continue
			}
			if rule.selector.hasDynamicPseudo() {
				continue
			}
			if rule.selector.matchesStatic(node) {
				matched = append(matched, rule)
			}
		}
		if len(matched) > 0 {
			sort.SliceStable(matched, func(i int, j int) bool {
				if matched[i].specificity != matched[j].specificity {
					return matched[i].specificity < matched[j].specificity
				}
				return matched[i].order < matched[j].order
			})
			for _, rule := range matched {
				state = parseCSSAutoMargins(rule.declarations, state)
			}
		}
	}
	if inline := attrValue(node, "style"); inline != "" {
		state = parseCSSAutoMargins(inline, state)
	}
	return state
}

func (sheet *pageStylesheet) listStyleType(node *Node, layout cssLayoutContext) (string, bool) {
	state := cssListStyleState{}
	if node == nil {
		return state.listStyleType, state.hasType
	}
	if sheet != nil && len(sheet.rules) > 0 {
		matched := make([]cssRule, 0, 8)
		for _, rule := range sheet.rules {
			if !rule.media.matches(layout) {
				continue
			}
			if rule.selector.hasDynamicPseudo() {
				continue
			}
			if rule.selector.matchesStatic(node) {
				matched = append(matched, rule)
			}
		}
		if len(matched) > 0 {
			sort.SliceStable(matched, func(i int, j int) bool {
				if matched[i].specificity != matched[j].specificity {
					return matched[i].specificity < matched[j].specificity
				}
				return matched[i].order < matched[j].order
			})
			for _, rule := range matched {
				state = parseCSSListStyleState(rule.declarations, state)
			}
		}
	}
	if inline := attrValue(node, "style"); inline != "" {
		state = parseCSSListStyleState(inline, state)
	}
	return state.listStyleType, state.hasType
}

func (sheet *pageStylesheet) backgroundState(node *Node, layout cssLayoutContext) cssBackgroundState {
	state := cssBackgroundState{}
	if node == nil {
		return state
	}
	if sheet != nil && len(sheet.rules) > 0 {
		matched := make([]cssRule, 0, 8)
		for _, rule := range sheet.rules {
			if !rule.media.matches(layout) {
				continue
			}
			if rule.selector.hasDynamicPseudo() {
				continue
			}
			if rule.selector.matchesStatic(node) {
				matched = append(matched, rule)
			}
		}
		if len(matched) > 0 {
			sort.SliceStable(matched, func(i int, j int) bool {
				if matched[i].specificity != matched[j].specificity {
					return matched[i].specificity < matched[j].specificity
				}
				return matched[i].order < matched[j].order
			})
			for _, rule := range matched {
				state = parseCSSBackgroundState(rule.declarations, state)
			}
		}
	}
	if inline := attrValue(node, "style"); inline != "" {
		state = parseCSSBackgroundState(inline, state)
	}
	return state
}

func (sheet *pageStylesheet) verticalAlign(node *Node, layout cssLayoutContext) (string, bool) {
	state := cssVerticalAlignState{}
	if node == nil {
		return state.value, state.hasValue
	}
	if sheet != nil && len(sheet.rules) > 0 {
		matched := make([]cssRule, 0, 8)
		for _, rule := range sheet.rules {
			if !rule.media.matches(layout) {
				continue
			}
			if rule.selector.hasDynamicPseudo() {
				continue
			}
			if rule.selector.matchesStatic(node) {
				matched = append(matched, rule)
			}
		}
		if len(matched) > 0 {
			sort.SliceStable(matched, func(i int, j int) bool {
				if matched[i].specificity != matched[j].specificity {
					return matched[i].specificity < matched[j].specificity
				}
				return matched[i].order < matched[j].order
			})
			for _, rule := range matched {
				state = parseCSSVerticalAlignState(rule.declarations, state)
			}
		}
	}
	if inline := attrValue(node, "style"); inline != "" {
		state = parseCSSVerticalAlignState(inline, state)
	}
	if state.hasValue {
		return state.value, true
	}
	if normalized, ok := normalizeCSSVerticalAlign(attrValue(node, "valign")); ok {
		return normalized, true
	}
	if node.Tag == "td" || node.Tag == "th" {
		return "middle", true
	}
	return "", false
}

func resolvedNodeBackgroundState(node *Node, ctx *renderContext) cssBackgroundState {
	if node == nil {
		return cssBackgroundState{}
	}
	layout := cssLayoutContext{}
	if ctx != nil {
		layout = ctx.cssLayoutContext()
	}
	if ctx != nil && ctx.stylesheet != nil {
		return ctx.stylesheet.backgroundState(node, layout)
	}
	state := cssBackgroundState{}
	if inline := attrValue(node, "style"); inline != "" {
		state = parseCSSBackgroundState(inline, state)
	}
	return state
}

func resolvedDocumentBackgroundState(doc *Document, ctx *renderContext) cssBackgroundState {
	state := cssBackgroundState{}
	if doc == nil {
		return state
	}
	for _, tag := range []string{"html", "body"} {
		nodes := doc.GetElementsByTagName(tag)
		if len(nodes) == 0 || nodes[0] == nil {
			continue
		}
		next := resolvedNodeBackgroundState(nodes[0], ctx)
		if next.hasImage {
			state.imageURL = next.imageURL
			state.hasImage = true
		}
		if next.hasRepeat {
			state.repeat = next.repeat
			state.hasRepeat = true
		}
		if next.hasAttachment {
			state.attachment = next.attachment
			state.hasAttachment = true
		}
	}
	return state
}

func resolvedNodeVerticalAlign(node *Node, ctx *renderContext) string {
	if node == nil {
		return ""
	}
	layout := cssLayoutContext{}
	if ctx != nil {
		layout = ctx.cssLayoutContext()
	}
	if ctx != nil && ctx.stylesheet != nil {
		if value, ok := ctx.stylesheet.verticalAlign(node, layout); ok {
			return value
		}
	} else if normalized, ok := normalizeCSSVerticalAlign(attrValue(node, "valign")); ok {
		return normalized
	}
	if normalized, ok := normalizeCSSVerticalAlign(attrValue(node, "valign")); ok {
		return normalized
	}
	if node.Tag == "td" || node.Tag == "th" {
		return "middle"
	}
	return ""
}

func applyPageNodeStyles(style *ui.Style, node *Node, ctx *renderContext) {
	if style == nil || node == nil {
		return
	}
	layout := ctx.cssLayoutContext()
	if ctx != nil && ctx.stylesheet != nil {
		ctx.stylesheet.apply(style, node, layout)
	} else if inline := attrValue(node, "style"); inline != "" {
		applyCSSDeclarations(style, inline, layout)
	}
	applyPageAutoMargins(style, node, ctx, layout)
}

func applyPageNodeStylesExceptBackground(style *ui.Style, node *Node, ctx *renderContext) {
	if style == nil || node == nil {
		return
	}
	layout := ctx.cssLayoutContext()
	resolved := ui.Style{}
	if ctx != nil && ctx.stylesheet != nil {
		ctx.stylesheet.apply(&resolved, node, layout)
	} else if inline := attrValue(node, "style"); inline != "" {
		applyCSSDeclarations(&resolved, inline, layout)
	}
	applyResolvedStyleExceptBackground(style, resolved)
	applyPageAutoMargins(style, node, ctx, layout)
}

func applyPageAutoMargins(style *ui.Style, node *Node, ctx *renderContext, layout cssLayoutContext) {
	if style == nil || node == nil || layout.viewportWidth <= 0 {
		return
	}
	auto := cssAutoMargins{}
	if ctx != nil && ctx.stylesheet != nil {
		auto = ctx.stylesheet.autoMargins(node, layout)
	} else {
		auto = parseCSSAutoMargins(attrValue(node, "style"), auto)
	}
	if !auto.left && !auto.right {
		return
	}
	targetWidth, ok := style.GetWidth()
	if !ok || targetWidth <= 0 {
		targetWidth, ok = style.GetMaxWidth()
	}
	if !ok || targetWidth <= 0 {
		return
	}
	margin, _ := style.GetMargin()
	remaining := layout.viewportWidth - targetWidth
	if remaining < 0 {
		remaining = 0
	}
	switch {
	case auto.left && auto.right:
		margin.Left = remaining / 2
		margin.Right = remaining - margin.Left
	case auto.left:
		margin.Left = remaining
	case auto.right:
		margin.Right = remaining
	}
	style.SetMargin(margin.Top, margin.Right, margin.Bottom, margin.Left)
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

func applyResolvedStyle(target *ui.Style, source ui.Style) {
	if target == nil {
		return
	}
	if color, ok := source.GetBackground(); ok {
		target.SetBackground(color)
	}
	if color, ok := source.GetForeground(); ok {
		target.SetForeground(color)
	}
	if color, ok := source.GetBorderColor(); ok {
		target.SetBorderColor(color)
	}
	if color, ok := source.GetBorderTopColor(); ok {
		target.SetBorderTopColor(color)
	}
	if color, ok := source.GetBorderRightColor(); ok {
		target.SetBorderRightColor(color)
	}
	if color, ok := source.GetBorderBottomColor(); ok {
		target.SetBorderBottomColor(color)
	}
	if color, ok := source.GetBorderLeftColor(); ok {
		target.SetBorderLeftColor(color)
	}
	if width, ok := source.GetBorderWidth(); ok {
		if color, colorOK := source.GetBorderColor(); colorOK {
			target.SetBorder(width, color)
		} else {
			target.SetBorderWidth(width)
		}
	}
	if width, ok := source.GetBorderTopWidth(); ok {
		if color, colorOK := source.GetBorderTopColor(); colorOK {
			target.SetBorderTop(width, color)
		} else {
			target.SetBorderTopWidth(width)
		}
	}
	if width, ok := source.GetBorderRightWidth(); ok {
		if color, colorOK := source.GetBorderRightColor(); colorOK {
			target.SetBorderRight(width, color)
		} else {
			target.SetBorderRightWidth(width)
		}
	}
	if width, ok := source.GetBorderBottomWidth(); ok {
		if color, colorOK := source.GetBorderBottomColor(); colorOK {
			target.SetBorderBottom(width, color)
		} else {
			target.SetBorderBottomWidth(width)
		}
	}
	if width, ok := source.GetBorderLeftWidth(); ok {
		if color, colorOK := source.GetBorderLeftColor(); colorOK {
			target.SetBorderLeft(width, color)
		} else {
			target.SetBorderLeftWidth(width)
		}
	}
	if radius, ok := source.GetBorderRadius(); ok {
		target.SetBorderRadius(radius.TopLeft, radius.TopRight, radius.BottomRight, radius.BottomLeft)
	}
	if gradient, ok := source.GetGradient(); ok {
		target.SetGradient(gradient)
	}
	if attachment, ok := source.GetBackgroundAttachment(); ok {
		target.SetBackgroundAttachment(attachment)
	}
	if shadow, ok := source.GetShadow(); ok {
		target.SetShadow(shadow)
	}
	if display, ok := source.GetDisplay(); ok {
		target.SetDisplay(display)
	}
	if alignItems, ok := source.GetAlignItems(); ok {
		target.SetAlignItems(alignItems)
	}
	if visibility, ok := source.GetVisibility(); ok {
		target.SetVisibility(visibility)
	}
	if align, ok := source.GetTextAlign(); ok {
		target.SetTextAlign(align)
	}
	if decoration, ok := source.GetTextDecoration(); ok {
		target.SetTextDecoration(decoration)
	}
	if whiteSpace, ok := source.GetWhiteSpace(); ok {
		target.SetWhiteSpace(whiteSpace)
	}
	if overflowWrap, ok := source.GetOverflowWrap(); ok {
		target.SetOverflowWrap(overflowWrap)
	}
	if wordBreak, ok := source.GetWordBreak(); ok {
		target.SetWordBreak(wordBreak)
	}
	if textShadow, ok := source.GetTextShadow(); ok {
		target.SetTextShadow(textShadow)
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
	if padding, ok := source.GetPadding(); ok {
		target.SetPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
	}
	if opacity, ok := source.GetOpacity(); ok {
		target.SetOpacity(opacity)
	}
	if boxSizing, ok := source.GetBoxSizing(); ok {
		target.SetBoxSizing(boxSizing)
	}
	if color, ok := source.GetOutlineColor(); ok {
		target.SetOutlineColor(color)
	}
	if width, ok := source.GetOutlineWidth(); ok {
		target.SetOutlineWidth(width)
	}
	if offset, ok := source.GetOutlineOffset(); ok {
		target.SetOutlineOffset(offset)
	}
	if radius, ok := source.GetOutlineRadius(); ok {
		target.SetOutlineRadius(radius)
	}
	if position, ok := source.GetPosition(); ok {
		target.SetPosition(position)
	}
	if value, ok := source.GetLeft(); ok {
		target.SetLeft(value)
	}
	if value, ok := source.GetTop(); ok {
		target.SetTop(value)
	}
	if value, ok := source.GetRight(); ok {
		target.SetRight(value)
	}
	if value, ok := source.GetBottom(); ok {
		target.SetBottom(value)
	}
	if value, ok := source.GetWidth(); ok {
		target.SetWidth(value)
	}
	if value, ok := source.GetFlexGrow(); ok {
		target.SetFlexGrowFloat(value)
	}
	if value, ok := source.GetHeight(); ok {
		target.SetHeight(value)
	}
	if value, ok := source.GetMinWidth(); ok {
		target.SetMinWidth(value)
	}
	if value, ok := source.GetMaxWidth(); ok {
		target.SetMaxWidth(value)
	}
	if value, ok := source.GetMinHeight(); ok {
		target.SetMinHeight(value)
	}
	if value, ok := source.GetMaxHeight(); ok {
		target.SetMaxHeight(value)
	}
	if margin, ok := source.GetMargin(); ok {
		target.SetMargin(margin.Top, margin.Right, margin.Bottom, margin.Left)
	}
	if overflow, ok := source.GetOverflow(); ok {
		target.SetOverflow(overflow)
	}
	if overflowX, ok := source.GetOverflowX(); ok {
		target.SetOverflowX(overflowX)
	}
	if overflowY, ok := source.GetOverflowY(); ok {
		target.SetOverflowY(overflowY)
	}
	if contain, ok := source.GetContain(); ok {
		target.SetContain(contain)
	}
	if willChange, ok := source.GetWillChange(); ok {
		target.SetWillChange(willChange)
	}
	if value, ok := source.GetScrollbarWidth(); ok {
		target.SetScrollbarWidth(value)
	}
	if color, ok := source.GetScrollbarTrack(); ok {
		target.SetScrollbarTrack(color)
	}
	if color, ok := source.GetScrollbarThumb(); ok {
		target.SetScrollbarThumb(color)
	}
	if value, ok := source.GetScrollbarRadius(); ok {
		target.SetScrollbarRadius(value)
	}
	if padding, ok := source.GetScrollbarPadding(); ok {
		target.SetScrollbarPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
	}
}

func applyResolvedStyleExceptBackground(target *ui.Style, source ui.Style) {
	if target == nil {
		return
	}
	if color, ok := source.GetForeground(); ok {
		target.SetForeground(color)
	}
	if color, ok := source.GetBorderColor(); ok {
		target.SetBorderColor(color)
	}
	if color, ok := source.GetBorderTopColor(); ok {
		target.SetBorderTopColor(color)
	}
	if color, ok := source.GetBorderRightColor(); ok {
		target.SetBorderRightColor(color)
	}
	if color, ok := source.GetBorderBottomColor(); ok {
		target.SetBorderBottomColor(color)
	}
	if color, ok := source.GetBorderLeftColor(); ok {
		target.SetBorderLeftColor(color)
	}
	if width, ok := source.GetBorderWidth(); ok {
		if color, colorOK := source.GetBorderColor(); colorOK {
			target.SetBorder(width, color)
		} else {
			target.SetBorderWidth(width)
		}
	}
	if width, ok := source.GetBorderTopWidth(); ok {
		if color, colorOK := source.GetBorderTopColor(); colorOK {
			target.SetBorderTop(width, color)
		} else {
			target.SetBorderTopWidth(width)
		}
	}
	if width, ok := source.GetBorderRightWidth(); ok {
		if color, colorOK := source.GetBorderRightColor(); colorOK {
			target.SetBorderRight(width, color)
		} else {
			target.SetBorderRightWidth(width)
		}
	}
	if width, ok := source.GetBorderBottomWidth(); ok {
		if color, colorOK := source.GetBorderBottomColor(); colorOK {
			target.SetBorderBottom(width, color)
		} else {
			target.SetBorderBottomWidth(width)
		}
	}
	if width, ok := source.GetBorderLeftWidth(); ok {
		if color, colorOK := source.GetBorderLeftColor(); colorOK {
			target.SetBorderLeft(width, color)
		} else {
			target.SetBorderLeftWidth(width)
		}
	}
	if radius, ok := source.GetBorderRadius(); ok {
		target.SetBorderRadius(radius.TopLeft, radius.TopRight, radius.BottomRight, radius.BottomLeft)
	}
	if display, ok := source.GetDisplay(); ok {
		target.SetDisplay(display)
	}
	if alignItems, ok := source.GetAlignItems(); ok {
		target.SetAlignItems(alignItems)
	}
	if position, ok := source.GetPosition(); ok {
		target.SetPosition(position)
	}
	if value, ok := source.GetTop(); ok {
		target.SetTop(value)
	}
	if value, ok := source.GetRight(); ok {
		target.SetRight(value)
	}
	if value, ok := source.GetBottom(); ok {
		target.SetBottom(value)
	}
	if value, ok := source.GetLeft(); ok {
		target.SetLeft(value)
	}
	if margin, ok := source.GetMargin(); ok {
		target.SetMargin(margin.Top, margin.Right, margin.Bottom, margin.Left)
	}
	if padding, ok := source.GetPadding(); ok {
		target.SetPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
	}
	if value, ok := source.GetWidth(); ok {
		target.SetWidth(value)
	}
	if value, ok := source.GetFlexGrow(); ok {
		target.SetFlexGrowFloat(value)
	}
	if value, ok := source.GetHeight(); ok {
		target.SetHeight(value)
	}
	if value, ok := source.GetMinWidth(); ok {
		target.SetMinWidth(value)
	}
	if value, ok := source.GetMaxWidth(); ok {
		target.SetMaxWidth(value)
	}
	if value, ok := source.GetMinHeight(); ok {
		target.SetMinHeight(value)
	}
	if value, ok := source.GetMaxHeight(); ok {
		target.SetMaxHeight(value)
	}
	if value, ok := source.GetForeground(); ok {
		target.SetForeground(value)
	}
	if path, ok := source.GetFontPath(); ok {
		target.SetFontPath(path)
	}
	if value, ok := source.GetFontSize(); ok {
		target.SetFontSize(value)
	}
	if value, ok := source.GetLineHeight(); ok {
		target.SetLineHeight(value)
	}
	if value, ok := source.GetTextAlign(); ok {
		target.SetTextAlign(value)
	}
	if value, ok := source.GetTextDecoration(); ok {
		target.SetTextDecoration(value)
	}
	if value, ok := source.GetWhiteSpace(); ok {
		target.SetWhiteSpace(value)
	}
	if value, ok := source.GetOverflow(); ok {
		target.SetOverflow(value)
	}
	if value, ok := source.GetOverflowX(); ok {
		target.SetOverflowX(value)
	}
	if value, ok := source.GetOverflowY(); ok {
		target.SetOverflowY(value)
	}
	if value, ok := source.GetContain(); ok {
		target.SetContain(value)
	}
	if shadow, ok := source.GetShadow(); ok {
		target.SetShadow(shadow)
	}
	if shadow, ok := source.GetTextShadow(); ok {
		target.SetTextShadow(shadow)
	}
	if value, ok := source.GetOpacity(); ok {
		target.SetOpacity(value)
	}
	if value, ok := source.GetOutlineColor(); ok {
		target.SetOutlineColor(value)
	}
	if value, ok := source.GetOutlineWidth(); ok {
		target.SetOutlineWidth(value)
	}
	if value, ok := source.GetOutlineOffset(); ok {
		target.SetOutlineOffset(value)
	}
	if value, ok := source.GetOutlineRadius(); ok {
		target.SetOutlineRadius(value)
	}
	if value, ok := source.GetWillChange(); ok {
		target.SetWillChange(value)
	}
	if value, ok := source.GetScrollbarWidth(); ok {
		target.SetScrollbarWidth(value)
	}
	if color, ok := source.GetScrollbarTrack(); ok {
		target.SetScrollbarTrack(color)
	}
	if color, ok := source.GetScrollbarThumb(); ok {
		target.SetScrollbarThumb(color)
	}
	if value, ok := source.GetScrollbarRadius(); ok {
		target.SetScrollbarRadius(value)
	}
	if padding, ok := source.GetScrollbarPadding(); ok {
		target.SetScrollbarPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
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
		if parsed, ok := parseCSSFontSize(value, *layout); ok {
			style.SetFontSize(parsed)
			layout.fontSize = parsed
			return
		}
	case "line-height":
		if parsed, ok := parseCSSLineHeight(value, *layout); ok {
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

func parseCSSFontSize(value string, layout cssLayoutContext) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if strings.HasSuffix(value, "%") {
		percent, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "%")), 64)
		if err != nil {
			return 0, false
		}
		return roundCSSPixels(float64(layout.fontSize) * percent / 100), true
	}
	return parseCSSLength(value, layout)
}

func parseCSSLineHeight(value string, layout cssLayoutContext) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "normal" {
		return 0, false
	}
	if strings.HasSuffix(value, "%") {
		percent, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "%")), 64)
		if err != nil {
			return 0, false
		}
		return roundCSSPixels(float64(layout.fontSize) * percent / 100), true
	}
	if strings.ContainsAny(value, "abcdefghijklmnopqrstuvwxyz") {
		return parseCSSLength(value, layout)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return roundCSSPixels(float64(layout.fontSize) * parsed), true
}

func roundCSSPixels(value float64) int {
	if value < 0 {
		return int(value - 0.5)
	}
	return int(value + 0.5)
}
