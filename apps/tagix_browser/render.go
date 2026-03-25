package main

import (
	"kos"
	neturl "net/url"
	"os"
	pathpkg "path"
	"strconv"
	"strings"
	"ui"
	"unicode/utf8"
)

const (
	webSansFontPath              = "assets/fonts/Go.ttf"
	webSansBoldFontPath          = "assets/fonts/GoBold.ttf"
	webSansItalicFontPath        = "assets/fonts/GoItalic.ttf"
	webSansBoldItalicFontPath    = "assets/fonts/GoBoldItalic.ttf"
	webMonoFontPath              = "assets/fonts/GoMono.ttf"
	webMonoBoldFontPath          = "assets/fonts/GoMonoBold.ttf"
	webMonoItalicFontPath        = "assets/fonts/GoMonoItalic.ttf"
	webMonoBoldItalicFontPath    = "assets/fonts/GoMonoBoldItalic.ttf"
	webIconFontPath              = "assets/fonts/MaterialDesignIconsDesktop.ttf"
	webShellHTML                 = "assets/shell.html"
	pageBackgroundMinTiledHeight = 8192
)

const (
	mdiIconChevronLeft          = 0xF0141
	mdiIconChevronRight         = 0xF0142
	mdiIconRefresh              = 0xF0450
	mdiIconHomeVariant          = 0xF02DE
	mdiIconCircleSmall          = 0xF09DF
	mdiIconCircleOutline        = 0xF0766
	mdiIconSquareOutline        = 0xF0763
	mdiIconCheckboxBlankOutline = 0xF0131
	mdiIconCheckboxMarked       = 0xF0132
	mdiIconRadioboxBlank        = 0xF043D
	mdiIconRadioboxMarked       = 0xF043E
)

var (
	cachedShellTemplate        string
	cachedShellTemplateRead    bool
	cachedShellTemplateSource  string
)

type renderContext struct {
	baseURL        string
	openURL        func(string)
	submitForm     func(string, string, neturl.Values)
	loadImage      func(string) *ui.DocumentImage
	imageError     func(string) string
	setStatusHint  func(string)
	requestPaint   func()
	requestLayout  func()
	stylesheet     *pageStylesheet
	viewportWidth  int
	viewportHeight int
	radioGroups    map[string][]*radioControlState
	forms          map[*Node]*formState
}

type shellRenderContext struct {
	stylesheet *pageStylesheet
	layout     cssLayoutContext
	body       *Node
	root       *Node
}

type radioControlState struct {
	node      *ui.DocumentNode
	indicator *ui.DocumentNode
	checked   bool
}

type formField struct {
	name  string
	value string
}

type formControlState struct {
	node   *Node
	fields func(*Node) []formField
	reset  func() bool
}

type formState struct {
	node     *Node
	ctx      *renderContext
	controls []*formControlState
}

func tagixIconFontPath() string {
	if path := lookupBundledFontFamilyPath("materialdesigniconsdesktop"); path != "" {
		return path
	}
	if info, err := os.Stat(webIconFontPath); err == nil && info != nil && !info.IsDir() {
		return webIconFontPath
	}
	return ""
}

func tagixIconGlyph(codepoint int, fallback string) string {
	if tagixIconFontPath() != "" && codepoint > 0 {
		return string(rune(codepoint))
	}
	return fallback
}

func isPrivateUseIconText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	r, size := utf8.DecodeRuneInString(value)
	return size == len(value) && r >= 0xF0000 && r <= 0x10FFFD
}

func applyTagixIconTextStyle(style *ui.Style, size int, lineHeight int, color kos.Color) {
	if style == nil {
		return
	}
	style.SetDisplay(ui.DisplayInline)
	if path := tagixIconFontPath(); path != "" {
		style.SetFontPath(path)
	} else {
		style.SetFontPath(webSansFontPath)
	}
	if size > 0 {
		style.SetFontSize(size)
	}
	if lineHeight > 0 {
		style.SetLineHeight(lineHeight)
	}
	style.SetForeground(color)
}

func shellActionGlyph(action string) string {
	switch action {
	case "back":
		return tagixIconGlyph(mdiIconChevronLeft, "←")
	case "forward":
		return tagixIconGlyph(mdiIconChevronRight, "→")
	case "reload":
		return tagixIconGlyph(mdiIconRefresh, "↻")
	case "home":
		return tagixIconGlyph(mdiIconHomeVariant, "⌂")
	default:
		return ""
	}
}

func htmlCheckboxGlyph(checked bool) string {
	if checked {
		return tagixIconGlyph(mdiIconCheckboxMarked, "☑")
	}
	return tagixIconGlyph(mdiIconCheckboxBlankOutline, "☐")
}

func htmlRadioGlyph(checked bool) string {
	if checked {
		return tagixIconGlyph(mdiIconRadioboxMarked, "◉")
	}
	return tagixIconGlyph(mdiIconRadioboxBlank, "◯")
}

func nearestAncestorTag(node *Node, tag string) *Node {
	for current := node; current != nil; current = current.Parent {
		if current.Type == ElementNode && current.Tag == tag {
			return current
		}
	}
	return nil
}

func findNodeByAttr(node *Node, key string, value string) *Node {
	if node == nil {
		return nil
	}
	if node.Type == ElementNode && attrValue(node, key) == value {
		return node
	}
	for _, child := range node.Children {
		if found := findNodeByAttr(child, key, value); found != nil {
			return found
		}
	}
	return nil
}

func (ctx *renderContext) formForControl(node *Node) *formState {
	if ctx == nil || node == nil {
		return nil
	}
	formNode := nearestAncestorTag(node, "form")
	if formNode == nil {
		return nil
	}
	if ctx.forms == nil {
		ctx.forms = map[*Node]*formState{}
	}
	if existing, ok := ctx.forms[formNode]; ok {
		return existing
	}
	form := &formState{
		node: formNode,
		ctx:  ctx,
	}
	ctx.forms[formNode] = form
	return form
}

func (form *formState) addControl(control *formControlState) {
	if form == nil || control == nil {
		return
	}
	form.controls = append(form.controls, control)
}

func (form *formState) submit(submitter *Node) {
	if form == nil || form.ctx == nil {
		return
	}
	method := strings.ToLower(attrValue(form.node, "method"))
	if method == "" {
		method = "get"
	}
	actionURL := resolveFormAction(form.ctx.baseURL, form.node)
	if actionURL == "" {
		return
	}
	values := make(neturl.Values)
	for _, control := range form.controls {
		if control == nil || control.fields == nil {
			continue
		}
		for _, field := range control.fields(submitter) {
			name := strings.TrimSpace(field.name)
			if name == "" {
				continue
			}
			values.Add(name, field.value)
		}
	}
	if form.ctx.submitForm != nil {
		form.ctx.submitForm(actionURL, method, values)
		return
	}
	if method == "get" && form.ctx.openURL != nil {
		form.ctx.openURL(appendURLQuery(actionURL, values.Encode()))
	}
}

func (form *formState) reset() {
	if form == nil {
		return
	}
	changed := false
	for _, control := range form.controls {
		if control == nil || control.reset == nil {
			continue
		}
		if control.reset() {
			changed = true
		}
	}
	if changed {
		requestRenderedPageLayout(form.ctx)
	}
}

func resolveFormAction(baseURL string, formNode *Node) string {
	action := attrValue(formNode, "action")
	if action == "" {
		return baseURL
	}
	resolved := resolveURL(baseURL, action)
	if resolved == "" {
		return ""
	}
	return resolved
}

func styled(update func(*ui.Style)) ui.Style {
	value := ui.Style{}
	if update != nil {
		update(&value)
	}
	return value
}

func newShellRenderContext(app *App, doc *Document) *shellRenderContext {
	width := defaultWindowWidth
	height := defaultWindowHeight
	if app != nil && app.window != nil {
		client := app.window.ClientRect()
		if client.Width > 0 {
			width = client.Width
		}
		if client.Height > 0 {
			height = client.Height
		}
	}
	ctx := &shellRenderContext{
		stylesheet: parseDocumentStylesheet(doc),
		layout: cssLayoutContext{
			viewportWidth:  width,
			viewportHeight: height,
			fontSize:       13,
		},
	}
	if doc != nil {
		nodes := doc.GetElementsByTagName("body")
		if len(nodes) > 0 {
			ctx.body = nodes[0]
		}
		ctx.root = findNodeByAttr(doc.Root, "data-role", "shell-root")
		if ctx.root == nil {
			ctx.root = findNodeByAttr(doc.Root, "id", "browser-shell")
		}
	}
	return ctx
}

func applyShellNodeStyles(style *ui.Style, node *Node, ctx *shellRenderContext) {
	if style == nil || node == nil {
		return
	}
	layout := cssLayoutContext{fontSize: 13}
	if ctx != nil {
		layout = ctx.layout
	}
	if ctx != nil && ctx.stylesheet != nil {
		ctx.stylesheet.apply(style, node, layout)
		return
	}
	if inline := attrValue(node, "style"); inline != "" {
		applyCSSDeclarations(style, inline, layout)
	}
}

func applyShellHostNodeStyles(style *ui.Style, node *Node, ctx *shellRenderContext) {
	if style == nil || node == nil {
		return
	}
	resolved := ui.Style{}
	applyShellNodeStyles(&resolved, node, ctx)
	if display, ok := resolved.GetDisplay(); ok {
		style.SetDisplay(display)
	}
	if alignItems, ok := resolved.GetAlignItems(); ok {
		style.SetAlignItems(alignItems)
	}
	if color, ok := resolved.GetBackground(); ok {
		style.SetBackground(color)
	}
	if margin, ok := resolved.GetMargin(); ok {
		style.SetMargin(margin.Top, margin.Right, margin.Bottom, margin.Left)
	}
	if padding, ok := resolved.GetPadding(); ok {
		style.SetPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
	}
	if border, ok := resolved.GetBorderWidth(); ok {
		if color, colorOK := resolved.GetBorderColor(); colorOK {
			style.SetBorder(border, color)
		} else {
			style.SetBorderWidth(border)
		}
	}
	if border, ok := resolved.GetBorderTopWidth(); ok {
		if color, colorOK := resolved.GetBorderTopColor(); colorOK {
			style.SetBorderTop(border, color)
		} else {
			style.SetBorderTopWidth(border)
		}
	}
	if border, ok := resolved.GetBorderRightWidth(); ok {
		if color, colorOK := resolved.GetBorderRightColor(); colorOK {
			style.SetBorderRight(border, color)
		} else {
			style.SetBorderRightWidth(border)
		}
	}
	if border, ok := resolved.GetBorderBottomWidth(); ok {
		if color, colorOK := resolved.GetBorderBottomColor(); colorOK {
			style.SetBorderBottom(border, color)
		} else {
			style.SetBorderBottomWidth(border)
		}
	}
	if border, ok := resolved.GetBorderLeftWidth(); ok {
		if color, colorOK := resolved.GetBorderLeftColor(); colorOK {
			style.SetBorderLeft(border, color)
		} else {
			style.SetBorderLeftWidth(border)
		}
	}
	if radius, ok := resolved.GetBorderRadius(); ok {
		style.SetBorderRadius(radius.TopLeft, radius.TopRight, radius.BottomRight, radius.BottomLeft)
	}
	if opacity, ok := resolved.GetOpacity(); ok {
		style.SetOpacity(opacity)
	}
	if boxSizing, ok := resolved.GetBoxSizing(); ok {
		style.SetBoxSizing(boxSizing)
	}
	if width, ok := resolved.GetWidth(); ok {
		style.SetWidth(width)
	}
	if grow, ok := resolved.GetFlexGrow(); ok {
		style.SetFlexGrowFloat(grow)
	}
	if width, ok := resolved.GetMinWidth(); ok {
		style.SetMinWidth(width)
	}
	if width, ok := resolved.GetMaxWidth(); ok {
		style.SetMaxWidth(width)
	}
	if overflow, ok := resolved.GetOverflow(); ok {
		style.SetOverflow(overflow)
	}
	if overflowX, ok := resolved.GetOverflowX(); ok {
		style.SetOverflowX(overflowX)
	}
	if overflowY, ok := resolved.GetOverflowY(); ok {
		style.SetOverflowY(overflowY)
	}
	copyPageTextProperties(style, resolved)
}

func applyShellTextProperties(style *ui.Style, node *Node, ctx *shellRenderContext) {
	if style == nil || node == nil {
		return
	}
	resolved := ui.Style{}
	applyShellNodeStyles(&resolved, node, ctx)
	copyPageTextProperties(style, resolved)
}

func applyShellViewportStyle(app *App, doc *Document, ctx *shellRenderContext) {
	if app == nil || app.shellView == nil || doc == nil {
		return
	}
	style := ui.Style{}
	style.SetBackground(0xF1F3F4)
	if ctx != nil && ctx.body != nil {
		applyShellNodeStyles(&style, ctx.body, ctx)
	}
	if color, ok := style.GetBackground(); ok {
		app.shellView.Style.SetBackground(color)
	}
}

func shellRootStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(0x202124)
		style.SetContain(ui.ContainPaint)
	})
}

func shellContainerStyleForTag(tag string) ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetContain(ui.ContainPaint)
	})
}

func shellTextStyleForTag(tag string) ui.Style {
	return styled(func(style *ui.Style) {
		style.SetFontPath(webSansFontPath)
		style.SetForeground(0x202124)
		switch tag {
		case "h1":
			style.SetDisplay(ui.DisplayBlock)
			style.SetFontSize(20)
			style.SetLineHeight(24)
			style.SetMargin(0, 0, 4, 0)
		case "h2":
			style.SetDisplay(ui.DisplayBlock)
			style.SetFontSize(16)
			style.SetLineHeight(20)
			style.SetMargin(0, 0, 4, 0)
		case "p":
			style.SetDisplay(ui.DisplayBlock)
			style.SetFontSize(13)
			style.SetLineHeight(18)
			style.SetForeground(0x3C4043)
			style.SetMargin(0, 0, 6, 0)
		case "small":
			style.SetDisplay(ui.DisplayBlock)
			style.SetFontSize(11)
			style.SetLineHeight(15)
			style.SetForeground(0x5F6368)
		default:
			style.SetDisplay(ui.DisplayInline)
			style.SetFontSize(13)
			style.SetLineHeight(18)
		}
	})
}

func shellRoleText(app *App, role string, fallback string) string {
	switch role {
	case "title":
		if app != nil {
			if value := strings.TrimSpace(app.pageTitle); value != "" {
				return value
			}
		}
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return "Tagix Browser"
	case "status":
		if app != nil {
			return app.currentShellStatus()
		}
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return "Ready"
	default:
		return fallback
	}
}

func shellLooseTextNode(text string) *ui.DocumentNode {
	text = normalizeBlockText(text)
	if text == "" {
		return nil
	}
	return ui.NewDocumentText(text, shellTextStyleForTag("span"))
}

func shellTextElementNode(app *App, node *Node) *ui.DocumentNode {
	return shellTextElementNodeWithContext(app, node, nil)
}

func shellTextElementNodeWithContext(app *App, node *Node, ctx *shellRenderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	text := collectNodeText(node, false)
	if text == "" {
		return nil
	}
	value := ui.NewDocumentText(text, shellTextStyleForTag(node.Tag))
	applyShellHostNodeStyles(&value.Style, node, ctx)
	applyShellInteractionStyles(value, node, ctx)
	return app.registerShellNode(node, value, shellNodeRef{})
}

func shellBoundTextNode(app *App, node *Node, role string, ctx *shellRenderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	text := shellRoleText(app, role, collectNodeText(node, false))
	value := ui.NewDocumentText(text, shellTextStyleForTag(node.Tag))
	applyShellHostNodeStyles(&value.Style, node, ctx)
	applyShellInteractionStyles(value, node, ctx)
	return app.registerShellNode(node, value, shellNodeRef{role: role})
}

func shellContainerNode(app *App, node *Node, name string, baseStyle ui.Style, ctx *shellRenderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildShellChildren(app, node, ctx)
	container := ui.NewDocumentElement(name, baseStyle, children...)
	if ctx != nil && ctx.body != nil && ctx.body != node && name == "browser-shell" {
		applyShellTextProperties(&container.Style, ctx.body, ctx)
	}
	applyShellHostNodeStyles(&container.Style, node, ctx)
	applyShellInteractionStyles(container, node, ctx)
	applyShellInteractionTextStyles(container)
	return app.registerShellNode(node, container, shellNodeRef{})
}

func buildShellDocument(app *App) *ui.DocumentNode {
	currentURL := defaultURL
	canBack := false
	canForward := false
	if app != nil {
		if value := strings.TrimSpace(app.addressText); value != "" {
			currentURL = value
		}
		canBack = app.historyIndex > 0
		canForward = app.historyIndex+1 < len(app.history)
	}
	root := ui.NewDocumentElement("browser-shell", shellRootStyle())
	root.Style.SetPadding(10, 12, 0, 12)
	root.Style.SetBackground(0xF1F3F4)

	actions := ui.NewDocumentElement("browser-shell-actions", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayFlex)
		style.SetAlignItems(ui.AlignItemsCenter)
		style.SetMargin(0, 0, 8, 0)
		style.SetContain(ui.ContainPaint)
	}),
		shellButtonNode(app, "Back", "back", canBack),
		shellButtonNode(app, "Forward", "forward", canForward),
		shellButtonNode(app, "Reload", "reload", true),
		shellButtonNode(app, "Home", "home", true),
	)

	address := shellAddressNode(app, currentURL)
	address.Style.SetMargin(0, 0, 0, 0)
	address.Style.SetFlexGrowFloat(1)
	actions.Append(address)
	root.Append(actions)
	return root
}

func renderShellRoot(app *App) *ui.DocumentNode {
	if app != nil {
		app.resetShellNodeRegistry()
	}
	doc := Parse(loadShellTemplateSource(app))
	if doc != nil && doc.Root != nil {
		ctx := newShellRenderContext(app, doc)
		applyShellViewportStyle(app, doc, ctx)
		if root := buildShellTemplateRoot(app, doc.Root, ctx); root != nil {
			return root
		}
	}
	return buildShellDocument(app)
}

func loadShellTemplateSource(app *App) string {
	sourcePath := webShellHTML
	if app != nil {
		if value := strings.TrimSpace(app.shellTemplatePath); value != "" {
			sourcePath = value
		}
	}
	if cachedShellTemplateRead && cachedShellTemplateSource == sourcePath {
		return cachedShellTemplate
	}
	cachedShellTemplateRead = true
	cachedShellTemplateSource = sourcePath
	data, err := os.ReadFile(sourcePath)
	if err != nil || len(data) == 0 {
		_, _, missing := missingBuiltinAssetPage(sourcePath)
		cachedShellTemplate = missing
		return cachedShellTemplate
	}
	cachedShellTemplate = string(data)
	return cachedShellTemplate
}

func buildShellTemplateRoot(app *App, node *Node, ctx *shellRenderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	if node.Type == ElementNode {
		role := strings.TrimSpace(node.Attrs["data-role"])
		if role == "shell-root" {
			return shellContainerNode(app, node, "browser-shell", shellRootStyle(), ctx)
		}
	}
	for _, child := range node.Children {
		if built := buildShellTemplateRoot(app, child, ctx); built != nil {
			return built
		}
	}
	if node.Type == ElementNode && node.Tag == "body" {
		return shellContainerNode(app, node, "browser-shell", shellRootStyle(), ctx)
	}
	return nil
}

func buildShellTemplateNode(app *App, node *Node, ctx *shellRenderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	switch node.Type {
	case CommentNode, DocumentNode:
		return nil
	case TextNode:
		return shellLooseTextNode(node.Text)
	case ElementNode:
	default:
		return nil
	}

	role := strings.TrimSpace(node.Attrs["data-role"])
	switch role {
	case "shell-root":
		return shellContainerNode(app, node, "browser-shell", shellRootStyle(), ctx)
	case "title":
		return shellBoundTextNode(app, node, "title", ctx)
	case "status-bar":
		applyShellStatusTemplate(app, node, ctx)
		return nil
	case "status":
		return shellBoundTextNode(app, node, "status", ctx)
	case "button":
		return shellButtonNodeWithSource(app, collectNodeText(node, false), strings.TrimSpace(node.Attrs["data-action"]), true, node, ctx)
	case "address":
		return shellAddressNodeWithSource(app, strings.TrimSpace(node.Attrs["value"]), node, ctx)
	case "page-frame":
		applyShellFrameTemplate(app, node, ctx)
		return nil
	}

	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link":
		return nil
	case "iframe":
		applyShellFrameTemplate(app, node, ctx)
		return nil
	case "footer":
		if strings.TrimSpace(attrValue(node, "data-role")) == "status-bar" {
			applyShellStatusTemplate(app, node, ctx)
			return nil
		}
		return shellContainerNode(app, node, node.Tag, shellContainerStyleForTag(node.Tag), ctx)
	case "button":
		return shellButtonNodeWithSource(app, collectNodeText(node, false), strings.TrimSpace(node.Attrs["data-action"]), true, node, ctx)
	case "input":
		if role == "address" || strings.TrimSpace(node.Attrs["type"]) == "" || strings.EqualFold(strings.TrimSpace(node.Attrs["type"]), "text") {
			return shellAddressNodeWithSource(app, strings.TrimSpace(node.Attrs["value"]), node, ctx)
		}
		return nil
	case "h1", "h2", "p", "small", "span", "strong", "em", "label":
		return shellTextElementNodeWithContext(app, node, ctx)
	default:
		return shellContainerNode(app, node, node.Tag, shellContainerStyleForTag(node.Tag), ctx)
	}
}

func buildShellChildren(app *App, node *Node, ctx *shellRenderContext) []*ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := make([]*ui.DocumentNode, 0, len(node.Children))
	for _, child := range node.Children {
		built := buildShellTemplateNode(app, child, ctx)
		if built != nil {
			children = append(children, built)
		}
	}
	return children
}

func shellButtonNode(app *App, label string, action string, enabled bool) *ui.DocumentNode {
	return shellButtonNodeWithSource(app, label, action, enabled, nil, nil)
}

func shellButtonNodeWithSource(app *App, label string, action string, enabled bool, source *Node, ctx *shellRenderContext) *ui.DocumentNode {
	label = normalizeBlockText(label)
	if icon := shellActionGlyph(action); icon != "" {
		label = icon
	}
	if label == "" {
		label = "Action"
	}
	button := ui.NewDocumentElement("shell-button-"+action, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 6, 0, 0)
		style.SetPadding(5, 0)
		style.SetWidth(30)
		style.SetHeight(28)
		style.SetBoxSizing(ui.BoxSizingBorderBox)
		style.SetBorderRadius(8)
		style.SetBorder(1, 0xC3CAD2)
		style.SetBackground(0xE5E9EE)
		style.SetTextAlign(ui.TextAlignCenter)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(label, styled(func(style *ui.Style) {
		if isPrivateUseIconText(label) {
			applyTagixIconTextStyle(style, 16, 18, 0x202124)
			return
		}
		style.SetDisplay(ui.DisplayInline)
		style.SetFontPath(webSansFontPath)
		style.SetForeground(0x202124)
		style.SetFontSize(12)
		style.SetLineHeight(16)
	})))
	applyShellHostNodeStyles(&button.Style, source, ctx)
	if len(button.Children) > 0 && button.Children[0] != nil {
		copyPageTextProperties(&button.Children[0].Style, button.Style)
		button.Children[0].Style.SetDisplay(ui.DisplayInline)
		if isPrivateUseIconText(label) {
			applyTagixIconTextStyle(&button.Children[0].Style, 16, 18, 0x202124)
		}
	}
	button.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(0x96A6B8)
		style.SetBackground(0xD7E1ED)
	})
	button.StyleActive = styled(func(style *ui.Style) {
		style.SetBorderColor(0x7E90A5)
		style.SetBackground(0xC7D2DE)
	})
	button.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(0x1A73E8)
		style.SetOutline(2, 0x1A73E8)
		style.SetOutlineOffset(1)
	})
	applyShellInteractionStyles(button, source, ctx)
	applyShellInteractionTextStyles(button)
	switch action {
	case "back":
		button.OnClick = func() {
			if app != nil {
				app.goBack()
			}
		}
	case "forward":
		button.OnClick = func() {
			if app != nil {
				app.goForward()
			}
		}
	case "reload":
		button.OnClick = func() {
			if app != nil {
				app.reloadCurrent()
			}
		}
	case "home":
		button.OnClick = func() {
			if app != nil {
				app.goHome()
			}
		}
	}
	applyShellButtonState(button, enabled)
	return app.registerShellNode(source, button, shellNodeRef{role: "button", action: action})
}

func applyShellButtonState(node *ui.DocumentNode, enabled bool) {
	if node == nil {
		return
	}
	node.Focusable = enabled
	if enabled {
		node.Style.SetOpacity(255)
	} else {
		node.Style.SetOpacity(220)
	}
	if len(node.Children) > 0 && node.Children[0] != nil {
		if enabled {
			node.Children[0].Style.SetOpacity(255)
		} else {
			node.Children[0].Style.SetOpacity(220)
		}
	}
}

func shellAddressNode(app *App, value string) *ui.DocumentNode {
	return shellAddressNodeWithSource(app, value, nil, nil)
}

func shellAddressNodeWithSource(app *App, value string, source *Node, ctx *shellRenderContext) *ui.DocumentNode {
	input := ui.NewDocumentElement("shell-address", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetPadding(7, 12)
		style.SetBorderRadius(10)
		style.SetBorder(1, 0x808A96)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
		style.SetOverflow(ui.OverflowHidden)
		style.SetWhiteSpace(ui.WhiteSpaceNoWrap)
		style.SetMinWidth(180)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}), nil)
	applyShellHostNodeStyles(&input.Style, source, ctx)
	input.Editable = true
	input.Focusable = true
	input.Value = value
	input.Placeholder = "Type URL here"
	input.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(0x7E8B98)
	})
	input.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(0x1A73E8)
	})
	applyShellInteractionStyles(input, source, ctx)
	input.OnInput = func(node *ui.DocumentNode) {
		if app != nil && node != nil {
			app.addressText = node.Value
		}
	}
	input.OnChange = func(node *ui.DocumentNode) {
		if app != nil && node != nil {
			app.addressText = node.Value
			app.submitAddress()
		}
	}
	return app.registerShellNode(source, input, shellNodeRef{role: "address"})
}

func syncShellDocument(app *App, title string, status string) {
	if app == nil || app.shellDocument == nil {
		return
	}
	layoutDirty := false
	paintDirty := false

	if app.setShellTextByID("browser-title", title) {
		layoutDirty = true
	}
	if app.setShellTextByRole("title", title) {
		layoutDirty = true
	}
	if setShellNodeText(app.shellTitleNode, title) {
		layoutDirty = true
	}

	if app.setShellTextByID("browser-status", status) {
		layoutDirty = true
	}
	if app.setShellTextByRole("status", status) {
		layoutDirty = true
	}
	if setShellNodeText(app.shellStatusNode, status) {
		layoutDirty = true
	}

	if app.setShellEnabledByID("browser-back", app.historyIndex > 0) {
		paintDirty = true
	}
	if app.setShellEnabledByAction("back", app.historyIndex > 0) {
		paintDirty = true
	}
	if setShellButtonEnabled(app.shellBackNode, app.historyIndex > 0) {
		paintDirty = true
	}

	if app.setShellEnabledByID("browser-forward", app.historyIndex+1 < len(app.history)) {
		paintDirty = true
	}
	if app.setShellEnabledByAction("forward", app.historyIndex+1 < len(app.history)) {
		paintDirty = true
	}
	if setShellButtonEnabled(app.shellForwardNode, app.historyIndex+1 < len(app.history)) {
		paintDirty = true
	}

	if app.setShellEnabledByID("browser-reload", true) {
		paintDirty = true
	}
	if app.setShellEnabledByAction("reload", true) {
		paintDirty = true
	}
	if setShellButtonEnabled(app.shellReloadNode, true) {
		paintDirty = true
	}

	if app.setShellEnabledByID("browser-home", true) {
		paintDirty = true
	}
	if app.setShellEnabledByAction("home", true) {
		paintDirty = true
	}
	if setShellButtonEnabled(app.shellHomeNode, true) {
		paintDirty = true
	}

	showStatus := strings.TrimSpace(status) != ""
	if app.setShellVisibleByID("browser-status", showStatus) {
		layoutDirty = true
	}

	address := strings.TrimSpace(app.addressText)
	if address == "" {
		address = defaultURL
	}
	if app.setShellValueByID("browser-address", address) {
		paintDirty = true
	}
	if app.setShellValueByRole("address", address) {
		paintDirty = true
	}
	if setShellNodeValue(app.shellAddressNode, address) {
		paintDirty = true
	}
	if app.shellAddressNode != nil {
		app.shellAddressNode.Placeholder = "Type URL here"
	}
	if layoutDirty {
		app.shellDocument.MarkLayoutDirty()
	} else if paintDirty {
		app.shellDocument.MarkDirty()
	}
}

const defaultShellTemplateHTML = `<html>
<head>
<style>
html,body{margin:0}
body{background:#f1f3f4}
#browser-shell{display:block;padding:10px 12px 0;background:#f1f3f4;font-family:system-ui,sans-serif;font-size:13px;line-height:18px;color:#202124;box-sizing:border-box}
#browser-toolbar{display:block;box-sizing:border-box}
#browser-controls{display:flex;align-items:center;margin:0}
.browser-button{display:inline-block;width:30px;height:28px;margin:0 6px 0 0;padding:5px 0;border:1px solid #c3cad2;border-radius:8px;background:#e5e9ee;color:#202124;font-size:16px;line-height:18px;text-align:center;box-sizing:border-box}
#browser-address{display:block;flex-grow:1;padding:7px 12px;border:1px solid #808a96;border-radius:10px;background:#fff;color:#202124;font-size:13px;line-height:18px;min-width:220px;box-sizing:border-box}
#browser-page-frame{display:block;width:100%;margin:8px 0 0;border:1px solid #d7dee7;border-radius:16px;background:#fff;min-height:280px;overflow:auto;box-sizing:border-box}
#browser-status-bar{display:block;margin:8px 0 0;padding:4px 10px;border:1px solid #d7dee7;background:#f8f9fa;color:#5f6368;box-sizing:border-box}
#browser-status{display:block;margin:0;font-size:11px;line-height:14px;color:#5f6368}
</style>
</head>
<body>
<header id="browser-shell" data-role="shell-root">
<section id="browser-toolbar">
<nav id="browser-controls">
<button id="browser-back" class="browser-button" data-role="button" data-action="back">&#x2190;</button>
<button id="browser-forward" class="browser-button" data-role="button" data-action="forward">&#x2192;</button>
<button id="browser-reload" class="browser-button" data-role="button" data-action="reload">&#x21bb;</button>
<button id="browser-home" class="browser-button" data-role="button" data-action="home">&#x2302;</button>
<input id="browser-address" data-role="address" value="">
</nav>
</section>
<iframe id="browser-page-frame" data-role="page-frame" src="about:tagix"></iframe>
<footer id="browser-status-bar" data-role="status-bar"><small id="browser-status" data-role="status">Ready</small></footer>
</header>
</body>
</html>`

func buildRenderedDocument(title string, currentURL string, doc *Document, viewportWidth int, viewportHeight int, openURL func(string), submitForm func(string, string, neturl.Values), loadImage func(string) *ui.DocumentImage, imageError func(string) string, setStatusHint func(string), requestLayout func(), requestPaint func()) *ui.DocumentNode {
	ctx := &renderContext{
		baseURL:        currentURL,
		openURL:        openURL,
		submitForm:     submitForm,
		loadImage:      loadImage,
		imageError:     imageError,
		setStatusHint:  setStatusHint,
		requestLayout:  requestLayout,
		requestPaint:   requestPaint,
		stylesheet:     parseDocumentStylesheet(doc),
		viewportWidth:  viewportWidth,
		viewportHeight: viewportHeight,
		radioGroups:    map[string][]*radioControlState{},
		forms:          map[*Node]*formState{},
	}
	var content *ui.DocumentNode
	contentNodes := buildDocumentNodes(doc, ctx)
	if len(contentNodes) == 0 {
		content = messageCard("No renderable content", "The HTML5 parser returned a tree, but the current browser-host adapter did not find readable nodes yet.")
	} else {
		content = ui.NewDocumentElement("content", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0)
			style.SetContain(ui.ContainPaint)
		}), contentNodes...)
	}
	children := make([]*ui.DocumentNode, 0, 24)
	if background := pageBackgroundLayerNode(doc, ctx, estimateDocumentNodeOuterHeight(content)); background != nil {
		children = append(children, background)
	}
	if content != nil {
		children = append(children, content)
	}

	pageStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPosition(ui.PositionRelative)
		style.SetPadding(0)
		style.SetContain(ui.ContainPaint)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(defaultPageFontSize)
		style.SetLineHeight(defaultPageLineHeight)
		style.SetForeground(0x333333)
		style.SetBackground(ui.White)
	})
	applyPageCanvasStyles(&pageStyle, doc, ctx)
	page := ui.NewDocumentElement("page", pageStyle, children...)
	stripImplicitPagePaintContain(page)
	return page
}

func stripImplicitPagePaintContain(node *ui.DocumentNode) {
	if node == nil {
		return
	}
	if contain, ok := node.Style.GetContain(); ok && contain&ui.ContainPaint != 0 {
		node.Style.SetContain(contain &^ ui.ContainPaint)
	}
	if contain, ok := node.StyleHover.GetContain(); ok && contain&ui.ContainPaint != 0 {
		node.StyleHover.SetContain(contain &^ ui.ContainPaint)
	}
	if contain, ok := node.StyleActive.GetContain(); ok && contain&ui.ContainPaint != 0 {
		node.StyleActive.SetContain(contain &^ ui.ContainPaint)
	}
	if contain, ok := node.StyleFocus.GetContain(); ok && contain&ui.ContainPaint != 0 {
		node.StyleFocus.SetContain(contain &^ ui.ContainPaint)
	}
	for _, child := range node.Children {
		stripImplicitPagePaintContain(child)
	}
}

func pageBackgroundLayerNode(doc *Document, ctx *renderContext, contentHeight int) *ui.DocumentNode {
	if doc == nil || ctx == nil || ctx.loadImage == nil {
		return nil
	}
	state := resolvedDocumentBackgroundState(doc, ctx)
	if !state.hasImage || strings.TrimSpace(state.imageURL) == "" {
		return nil
	}
	image := ctx.loadImage(state.imageURL)
	if image == nil || !image.Valid() {
		return nil
	}
	repeat := state.repeat
	if repeat == "" {
		repeat = "repeat"
	}
	width := ctx.viewportWidth
	height := ctx.viewportHeight
	if width <= 0 {
		width = defaultWindowWidth
	}
	if height <= 0 {
		height = defaultPageHeight
	}
	if contentHeight > height {
		height = contentHeight
	}
	if height < pageBackgroundMinTiledHeight {
		height = pageBackgroundMinTiledHeight
	}
	layer := ui.NewDocumentElement("page-background", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPosition(ui.PositionAbsolute)
		style.SetLeft(0)
		style.SetTop(0)
		style.SetWidth(width)
		style.SetHeight(height)
		style.SetContain(ui.ContainPaint)
	}))
	tileWidth := image.Width
	tileHeight := image.Height
	if tileWidth <= 0 || tileHeight <= 0 {
		return nil
	}
	xCount := 1
	yCount := 1
	switch repeat {
	case "repeat", "repeat-x":
		xCount = (width + tileWidth - 1) / tileWidth
		if xCount < 1 {
			xCount = 1
		}
	}
	switch repeat {
	case "repeat", "repeat-y":
		yCount = (height + tileHeight - 1) / tileHeight
		if yCount < 1 {
			yCount = 1
		}
	}
	for y := 0; y < yCount; y++ {
		for x := 0; x < xCount; x++ {
			tile := ui.NewDocumentElement("page-background-tile", styled(func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetPosition(ui.PositionAbsolute)
				style.SetLeft(x * tileWidth)
				style.SetTop(y * tileHeight)
				style.SetWidth(tileWidth)
				style.SetHeight(tileHeight)
				style.SetContain(ui.ContainPaint)
			}))
			tile.Image = image
			layer.Append(tile)
		}
	}
	return layer
}

func buildDocumentNodes(doc *Document, ctx *renderContext) []*ui.DocumentNode {
	if doc == nil || doc.Root == nil {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, 16)
	appendDocumentNodes(&nodes, doc.Root, ctx)
	return nodes
}

func buildFlowNodes(node *Node, ctx *renderContext) []*ui.DocumentNode {
	if node == nil {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, len(node.Children))
	appendFlowContentNodes(&nodes, node, ctx)
	return nodes
}

func appendFlowContentNodes(out *[]*ui.DocumentNode, node *Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	inlineStyle := inlineTextStyleFromStyle(paragraphInlineStyle())
	applyPageTextPropertiesToInlineStyle(&inlineStyle, node, ctx)
	builder := inlinePieceBuilder{}
	flushParagraph := func() {
		if paragraph := flowParagraphNode(builder.pieces, node, ctx); paragraph != nil {
			*out = append(*out, paragraph)
		}
		builder = inlinePieceBuilder{}
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		switch child.Type {
		case CommentNode:
			continue
		case TextNode:
			builder.appendText(child.Text, inlineStyle)
		case DocumentNode:
			flushParagraph()
			appendFlowContentNodes(out, child, ctx)
		case ElementNode:
			if isSkippableElementTag(child.Tag) {
				continue
			}
			if pageNodeDisplayNone(child, ctx) {
				continue
			}
			if nodeParticipatesInInlineFlow(child) {
				collectInlinePieces(&builder, child, ctx, inlineStyle)
				continue
			}
			flushParagraph()
			appendDocumentNodes(out, child, ctx)
		}
	}
	flushParagraph()
}

func isSkippableElementTag(tag string) bool {
	switch tag {
	case "script", "style", "head", "title", "meta", "link", "option", "source", "template":
		return true
	default:
		return false
	}
}

func nodeParticipatesInInlineFlow(node *Node) bool {
	if node == nil {
		return false
	}
	switch node.Type {
	case TextNode:
		return normalizeBlockText(node.Text) != ""
	case CommentNode, DocumentNode:
		return false
	case ElementNode:
	default:
		return false
	}
	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link", "option", "source", "template":
		return false
	case "br", "a", "code", "img", "span", "strong", "em", "b", "i", "u", "small", "big", "abbr", "cite", "q", "s", "sub", "sup", "mark", "time", "kbd", "samp", "var", "wbr", "label", "button", "input", "textarea", "select", "progress":
		return true
	case "html", "body", "main", "section", "article", "aside", "nav", "header", "footer", "div", "form", "fieldset", "legend", "figure", "figcaption", "details", "summary", "dl", "dt", "dd", "table", "caption", "tbody", "thead", "tfoot", "tr", "td", "th", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "p", "blockquote", "pre", "ul", "ol", "li", "iframe":
		return false
	default:
		return true
	}
}

func pageNodeDisplayNone(node *Node, ctx *renderContext) bool {
	if node == nil || node.Type != ElementNode {
		return false
	}
	if hasAttr(node, "hidden") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(attrValue(node, "aria-hidden")), "true") {
		return true
	}
	resolved := ui.Style{}
	applyPageNodeStyles(&resolved, node, ctx)
	if display, ok := resolved.GetDisplay(); ok && display == ui.DisplayNone {
		return true
	}
	return false
}

func appendDocumentNodes(out *[]*ui.DocumentNode, node *Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}

	switch node.Type {
	case CommentNode:
		return
	case TextNode:
		text := normalizeBlockText(node.Text)
		if text != "" {
			*out = append(*out, paragraphNode(text))
		}
		return
	case DocumentNode:
		appendFlowContentNodes(out, node, ctx)
		return
	case ElementNode:
	default:
		return
	}
	if pageNodeDisplayNone(node, ctx) {
		return
	}

	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link", "option", "source", "template":
		return
	case "html", "body", "main", "section", "article", "aside", "nav", "header", "footer", "div", "form":
		if container := genericBlockContainerNode(node, ctx); container != nil {
			*out = append(*out, container)
		}
		return
	case "label", "dl", "tbody", "thead", "tfoot", "tr", "td", "th":
		appendFlowContentNodes(out, node, ctx)
		return
	case "fieldset":
		if fieldset := fieldsetBlockNode(node, ctx); fieldset != nil {
			*out = append(*out, fieldset)
		}
		return
	case "legend":
		if legend := legendBlockNode(node, ctx); legend != nil {
			*out = append(*out, legend)
		}
		return
	case "figure":
		if figure := figureBlockNode(node, ctx); figure != nil {
			*out = append(*out, figure)
		}
		return
	case "figcaption", "caption":
		if caption := figureCaptionBlockNode(node, ctx); caption != nil {
			*out = append(*out, caption)
		}
		return
	case "details":
		if details := detailsBlockNode(node, ctx); details != nil {
			*out = append(*out, details)
		}
		return
	case "summary":
		if summary := summaryBlockNode(node, ctx); summary != nil {
			*out = append(*out, summary)
		}
		return
	case "dt":
		if term := definitionTermBlockNode(node, ctx); term != nil {
			*out = append(*out, term)
		}
		return
	case "dd":
		if detail := definitionDetailBlockNode(node, ctx); detail != nil {
			*out = append(*out, detail)
		}
		return
	case "table":
		if table := tableBlockNode(node, ctx); table != nil {
			*out = append(*out, table)
		}
		return
	case "hr":
		*out = append(*out, separatorNode())
		return
	case "br":
		return
	case "h1", "h2", "h3", "h4", "h5", "h6":
		if heading := headingBlockNode(node, ctx); heading != nil {
			*out = append(*out, heading)
		}
		return
	case "p":
		if paragraph := paragraphBlockNode(node, ctx); paragraph != nil {
			*out = append(*out, paragraph)
		}
		return
	case "blockquote":
		if quote := blockquoteBlockNode(node, ctx); quote != nil {
			*out = append(*out, quote)
		}
		return
	case "pre":
		text := collectNodeTextPreserve(node, true)
		if text != "" {
			*out = append(*out, preformattedNode(text))
		}
		return
	case "code":
		if code := codeBlockNode(node, ctx); code != nil {
			*out = append(*out, code)
		}
		return
	case "ul", "ol":
		appendListNodes(out, node, ctx)
		return
	case "li":
		if item := listItemBlockNode(node, ctx, "-"); item != nil {
			*out = append(*out, item)
		}
		return
	case "a":
		if anchorHasStructuredContent(node) {
			if link := structuredLinkContainerNode(node, ctx); link != nil {
				*out = append(*out, link)
			}
			return
		}
		if link := standaloneLinkNode(node, ctx); link != nil {
			*out = append(*out, link)
		}
		return
	case "button":
		if button := htmlButtonNode(node, ctx); button != nil {
			*out = append(*out, button)
		}
		return
	case "input":
		if input := htmlInputNode(node, ctx); input != nil {
			*out = append(*out, input)
		}
		return
	case "textarea":
		if area := htmlTextareaNode(node, ctx); area != nil {
			*out = append(*out, area)
		}
		return
	case "select":
		if selectNode := htmlSelectNode(node, ctx); selectNode != nil {
			*out = append(*out, selectNode)
		}
		return
	case "progress":
		if progress := htmlProgressNode(node); progress != nil {
			*out = append(*out, progress)
		}
		return
	case "iframe":
		if frame := iframeFallbackNode(node, ctx); frame != nil {
			*out = append(*out, frame)
		}
		return
	case "option":
		return
	case "img":
		if image := imageFallbackNode(node, ctx); image != nil {
			*out = append(*out, image)
		}
		return
	default:
		appendFlowContentNodes(out, node, ctx)
	}
}

func appendListNodes(out *[]*ui.DocumentNode, node *Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	ordered := node.Tag == "ol"
	items := listItemElements(node)
	index := orderedListStart(node, len(items))
	step := 1
	if ordered && hasAttr(node, "reversed") {
		step = -1
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		if child.Type == ElementNode && child.Tag == "li" {
			itemIndex := index
			if ordered && hasAttr(child, "value") {
				itemIndex = attrInt(child, "value", index)
				index = itemIndex
			}
			marker := resolvedListItemMarker(node, child, ctx, ordered, itemIndex)
			if item := listItemBlockNode(child, ctx, marker); item != nil {
				*out = append(*out, item)
			}
			if ordered {
				index += step
			}
			continue
		}
		appendDocumentNodes(out, child, ctx)
	}
}

func listItemElements(node *Node) []*Node {
	if node == nil {
		return nil
	}
	items := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		if child != nil && child.Type == ElementNode && child.Tag == "li" {
			items = append(items, child)
		}
	}
	return items
}

func orderedListStart(node *Node, itemCount int) int {
	if node == nil {
		return 1
	}
	if hasAttr(node, "start") {
		return attrInt(node, "start", 1)
	}
	if hasAttr(node, "reversed") {
		if itemCount > 0 {
			return itemCount
		}
		return 0
	}
	return 1
}

func resolvedListItemMarker(listNode *Node, itemNode *Node, ctx *renderContext, ordered bool, index int) string {
	styleType := resolvedListStyleType(listNode, itemNode, ctx, ordered)
	if styleType == "none" {
		return ""
	}
	if ordered {
		return formatOrderedListMarker(styleType, index)
	}
	return formatUnorderedListMarker(styleType)
}

func resolvedListStyleType(listNode *Node, itemNode *Node, ctx *renderContext, ordered bool) string {
	if itemStyle := resolvedNodeListStyleType(itemNode, ctx, ordered); itemStyle != "" {
		return itemStyle
	}
	if listStyle := resolvedNodeListStyleType(listNode, ctx, ordered); listStyle != "" {
		return listStyle
	}
	if ordered {
		return "decimal"
	}
	return "disc"
}

func resolvedNodeListStyleType(node *Node, ctx *renderContext, ordered bool) string {
	if node == nil {
		return ""
	}
	if ctx != nil && ctx.stylesheet != nil {
		if value, ok := ctx.stylesheet.listStyleType(node, ctx.cssLayoutContext()); ok {
			return value
		}
	} else if inline := attrValue(node, "style"); inline != "" {
		if value, ok := parseInlineListStyleType(inline); ok {
			return value
		}
	}
	return listStyleTypeFromHTMLAttrs(node, ordered)
}

func parseInlineListStyleType(inline string) (string, bool) {
	state := parseCSSListStyleState(inline, cssListStyleState{})
	return state.listStyleType, state.hasType
}

func listStyleTypeFromHTMLAttrs(node *Node, ordered bool) string {
	value := strings.TrimSpace(attrValue(node, "type"))
	if value == "" {
		return ""
	}
	if ordered {
		switch value {
		case "1":
			return "decimal"
		case "a":
			return "lower-alpha"
		case "A":
			return "upper-alpha"
		case "i":
			return "lower-roman"
		case "I":
			return "upper-roman"
		}
		return ""
	}
	switch strings.ToLower(value) {
	case "disc", "circle", "square":
		return strings.ToLower(value)
	default:
		return ""
	}
}

func formatUnorderedListMarker(styleType string) string {
	switch styleType {
	case "", "disc":
		return tagixIconGlyph(mdiIconCircleSmall, "•")
	case "circle":
		return tagixIconGlyph(mdiIconCircleOutline, "◦")
	case "square":
		return tagixIconGlyph(mdiIconSquareOutline, "▪")
	default:
		return tagixIconGlyph(mdiIconCircleSmall, "•")
	}
}

func formatOrderedListMarker(styleType string, index int) string {
	switch styleType {
	case "", "decimal":
		return strconv.Itoa(index) + "."
	case "decimal-leading-zero":
		if index >= 0 && index < 10 {
			return "0" + strconv.Itoa(index) + "."
		}
		return strconv.Itoa(index) + "."
	case "lower-alpha", "lower-latin":
		if value := formatAlphabeticListIndex(index, false); value != "" {
			return value + "."
		}
	case "upper-alpha", "upper-latin":
		if value := formatAlphabeticListIndex(index, true); value != "" {
			return value + "."
		}
	case "lower-roman":
		if value := formatRomanListIndex(index, false); value != "" {
			return value + "."
		}
	case "upper-roman":
		if value := formatRomanListIndex(index, true); value != "" {
			return value + "."
		}
	case "none":
		return ""
	}
	return strconv.Itoa(index) + "."
}

func formatAlphabeticListIndex(index int, upper bool) string {
	if index <= 0 {
		return ""
	}
	index--
	buf := make([]byte, 0, 4)
	base := byte('a')
	if upper {
		base = 'A'
	}
	for {
		buf = append(buf, base+byte(index%26))
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}
	return string(buf)
}

func formatRomanListIndex(index int, upper bool) string {
	if index <= 0 || index > 3999 {
		return ""
	}
	type romanPart struct {
		value  int
		symbol string
	}
	parts := []romanPart{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}
	var builder strings.Builder
	for _, part := range parts {
		for index >= part.value {
			builder.WriteString(part.symbol)
			index -= part.value
		}
	}
	value := builder.String()
	if !upper {
		value = strings.ToLower(value)
	}
	return value
}

func appendNestedDocumentLinks(out *[]*ui.DocumentNode, node *Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	for _, child := range node.Children {
		appendDirectAnchorNodes(out, child, ctx)
	}
}

func appendDirectAnchorNodes(out *[]*ui.DocumentNode, node *Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	if node.Type == ElementNode && node.Tag == "a" {
		if link := documentLinkNodeFromAnchor(node, ctx); link != nil {
			*out = append(*out, link)
		}
		return
	}
	for _, child := range node.Children {
		appendDirectAnchorNodes(out, child, ctx)
	}
}

func collectNodeText(node *Node, skipLinks bool) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	collectNodeTextInto(&builder, node, skipLinks, false)
	return normalizeBlockText(builder.String())
}

func collectNodeTextPreserve(node *Node, skipLinks bool) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	collectNodeTextInto(&builder, node, skipLinks, true)
	return strings.TrimSpace(builder.String())
}

func collectNodeTextInto(builder *strings.Builder, node *Node, skipLinks bool, preserve bool) {
	if builder == nil || node == nil {
		return
	}
	switch node.Type {
	case CommentNode:
		return
	case TextNode:
		text := node.Text
		if !preserve {
			text = normalizeBlockText(text)
		}
		if text == "" {
			return
		}
		if builder.Len() > 0 && !preserve {
			_ = builder.WriteByte(' ')
		}
		builder.WriteString(text)
		return
	case ElementNode:
		if skipLinks && node.Tag == "a" {
			return
		}
		if node.Tag == "br" {
			if preserve {
				_ = builder.WriteByte('\n')
			}
			return
		}
	}
	for _, child := range node.Children {
		collectNodeTextInto(builder, child, skipLinks, preserve)
	}
}

func defaultLineHeightForFontSize(fontSize int) int {
	if fontSize <= 0 {
		fontSize = defaultPageFontSize
	}
	lineHeight := roundCSSPixels(float64(fontSize) * 1.25)
	if lineHeight < fontSize+3 {
		lineHeight = fontSize + 3
	}
	return lineHeight
}

func defaultBlockMarginBottom(fontSize int, scale float64) int {
	if fontSize <= 0 {
		fontSize = defaultPageFontSize
	}
	margin := roundCSSPixels(float64(fontSize) * scale)
	if margin < 0 {
		return 0
	}
	return margin
}

func resolvedPageTextMetrics(node *Node, ctx *renderContext, fallbackFontSize int, marginScale float64) (ui.Style, int, int, int) {
	resolved := ui.Style{}
	if node != nil {
		applyPageNodeStyles(&resolved, node, ctx)
	}
	fontSize := fallbackFontSize
	if fontSize <= 0 {
		fontSize = defaultPageFontSize
	}
	if value, ok := resolved.GetFontSize(); ok && value > 0 {
		fontSize = value
	}
	lineHeight := defaultLineHeightForFontSize(fontSize)
	if value, ok := resolved.GetLineHeight(); ok && value > 0 {
		lineHeight = value
	}
	marginBottom := defaultBlockMarginBottom(fontSize, marginScale)
	return resolved, fontSize, lineHeight, marginBottom
}

func headingNode(tag string, text string) *ui.DocumentNode {
	size := 14
	switch tag {
	case "h1":
		size = 22
	case "h2":
		size = 18
	case "h3":
		size = 16
	}
	marginBottom := defaultBlockMarginBottom(size, 0.67)
	return ui.NewDocumentText(text, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 4, marginBottom, 0)
		style.SetForeground(0x333333)
		style.SetFontSize(size)
		style.SetLineHeight(defaultLineHeightForFontSize(size))
	}))
}

type inlinePieceKind uint8

const (
	inlinePieceText inlinePieceKind = iota
	inlinePieceLink
	inlinePieceCode
	inlinePieceImage
	inlinePieceBreak
	inlinePieceControl
)

type inlinePiece struct {
	kind  inlinePieceKind
	text  string
	href  string
	node  *Node
	style inlineTextStyle
}

type inlinePieceBuilder struct {
	pieces    []inlinePiece
	needSpace bool
}

type inlineTextStyle struct {
	fontPath          string
	fontSize          int
	lineHeight        int
	foreground        kos.Color
	hasForeground     bool
	textDecoration    ui.TextDecoration
	hasTextDecoration bool
	whiteSpace        ui.WhiteSpaceMode
	hasWhiteSpace     bool
}

func paragraphInlineStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(0x333333)
		style.SetFontSize(defaultPageFontSize)
		style.SetLineHeight(defaultLineHeightForFontSize(defaultPageFontSize))
	})
}

func paragraphBlockStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, defaultBlockMarginBottom(defaultPageFontSize, 1.0), 0)
		style.SetForeground(0x333333)
		style.SetFontSize(defaultPageFontSize)
		style.SetLineHeight(defaultLineHeightForFontSize(defaultPageFontSize))
		style.SetContain(ui.ContainPaint)
	})
}

func paragraphNode(text string) *ui.DocumentNode {
	return ui.NewDocumentText(text, paragraphBlockStyle())
}

func flowParagraphNode(pieces []inlinePiece, owner *Node, ctx *renderContext) *ui.DocumentNode {
	children := inlineNodesFromPieces(pieces, ctx)
	if len(children) == 0 {
		return nil
	}
	blockStyle := paragraphBlockStyle()
	if owner != nil {
		applyPageTextProperties(&blockStyle, owner, ctx)
	}
	return ui.NewDocumentElement("flow-paragraph", blockStyle, children...)
}

func headingBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	size := 14
	switch node.Tag {
	case "h1":
		size = 22
	case "h2":
		size = 18
	case "h3":
		size = 16
	}
	resolved, fontSize, lineHeight, marginBottom := resolvedPageTextMetrics(node, ctx, size, 0.67)
	inlineStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(0x333333)
		style.SetFontSize(fontSize)
		style.SetLineHeight(lineHeight)
	})
	copyPageTextProperties(&inlineStyle, resolved)
	children := buildInlineNodes(node, ctx, inlineStyle)
	if len(children) == 0 {
		return nil
	}
	blockStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, marginBottom, 0)
		style.SetForeground(0x333333)
		style.SetFontSize(fontSize)
		style.SetLineHeight(lineHeight)
		style.SetContain(ui.ContainPaint)
	})
	copyPageTextProperties(&blockStyle, inlineStyle)
	applyPageNodeStyles(&blockStyle, node, ctx)
	block := ui.NewDocumentElement("heading-"+node.Tag, blockStyle, children...)
	applyPageInteractionStyles(block, node, ctx)
	applyPageInteractionTextStyles(block)
	return block
}

func paragraphBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	resolved, fontSize, lineHeight, marginBottom := resolvedPageTextMetrics(node, ctx, defaultPageFontSize, 1.0)
	inlineStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(0x333333)
		style.SetFontSize(fontSize)
		style.SetLineHeight(lineHeight)
	})
	copyPageTextProperties(&inlineStyle, resolved)
	children := buildInlineNodes(node, ctx, inlineStyle)
	if len(children) == 0 {
		return nil
	}
	blockStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, marginBottom, 0)
		style.SetForeground(0x333333)
		style.SetFontSize(fontSize)
		style.SetLineHeight(lineHeight)
		style.SetContain(ui.ContainPaint)
	})
	copyPageTextProperties(&blockStyle, inlineStyle)
	applyPageNodeStyles(&blockStyle, node, ctx)
	block := ui.NewDocumentElement("paragraph", blockStyle, children...)
	applyPageInteractionStyles(block, node, ctx)
	applyPageInteractionTextStyles(block)
	return block
}

func codeBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	resolved, fontSize, lineHeight, marginBottom := resolvedPageTextMetrics(node, ctx, defaultPageFontSize, 1.0)
	inlineStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(0x333333)
		style.SetFontSize(fontSize)
		style.SetLineHeight(lineHeight)
	})
	copyPageTextProperties(&inlineStyle, resolved)
	inlineStyle.SetFontPath(resolveRegularSemanticFontPath(styleFontPath(inlineStyle), true))
	children := buildInlineNodes(node, ctx, inlineStyle)
	if len(children) == 0 {
		return nil
	}
	blockStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, marginBottom, 0)
		style.SetForeground(0x333333)
		style.SetFontSize(fontSize)
		style.SetLineHeight(lineHeight)
		style.SetContain(ui.ContainPaint)
	})
	copyPageTextProperties(&blockStyle, inlineStyle)
	applyPageNodeStyles(&blockStyle, node, ctx)
	blockStyle.SetFontPath(resolveRegularSemanticFontPath(styleFontPath(blockStyle), true))
	block := ui.NewDocumentElement("code", blockStyle, children...)
	applyPageInteractionStyles(block, node, ctx)
	applyPageInteractionTextStyles(block)
	return block
}

func blockquoteBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("blockquote", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(2, 0, 2, 12)
		style.SetBorder(0, ui.White)
		style.SetBorderLeft(3, 0xC0C6CC)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func semanticLabelInlineStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Navy)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	})
}

func semanticLabelBlockStyle(marginBottom int) ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, marginBottom, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	})
}

func legendBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, semanticLabelInlineStyle())
	if len(children) == 0 {
		return nil
	}
	value := ui.NewDocumentElement("legend", semanticLabelBlockStyle(6), children...)
	applyPageInteractionStyles(value, node, ctx)
	applyPageInteractionTextStyles(value)
	return value
}

func figureCaptionBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
		style.SetLineHeight(17)
	}))
	if len(children) == 0 {
		return nil
	}
	value := ui.NewDocumentElement("figcaption", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(4, 0, 8, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
		style.SetLineHeight(17)
		style.SetContain(ui.ContainPaint)
	}), children...)
	applyPageInteractionStyles(value, node, ctx)
	applyPageInteractionTextStyles(value)
	return value
}

func summaryBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, semanticLabelInlineStyle())
	if len(children) == 0 {
		return nil
	}
	value := ui.NewDocumentElement("summary", semanticLabelBlockStyle(6), children...)
	applyPageInteractionStyles(value, node, ctx)
	applyPageInteractionTextStyles(value)
	return value
}

func definitionTermBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, semanticLabelInlineStyle())
	if len(children) == 0 {
		return nil
	}
	value := ui.NewDocumentElement("definition-term", semanticLabelBlockStyle(2), children...)
	applyPageInteractionStyles(value, node, ctx)
	applyPageInteractionTextStyles(value)
	return value
}

func definitionDetailBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("definition-detail", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 14)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func fieldsetBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("fieldset", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(10, 12)
		style.SetBorderRadius(8)
		style.SetBorder(1, 0xD8DEE4)
		style.SetBackground(0xFAFBFC)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func figureBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("figure", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func detailsBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("details", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(10, 12)
		style.SetBorderRadius(8)
		style.SetBorder(1, 0xD8DEE4)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func listMarkerNode(marker string) *ui.DocumentNode {
	return ui.NewDocumentText(marker+" ", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Navy)
		if isPrivateUseIconText(marker) {
			if path := tagixIconFontPath(); path != "" {
				style.SetFontPath(path)
			}
			style.SetFontSize(14)
			style.SetLineHeight(18)
			return
		}
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}))
}

func listLeadChildren(node *ui.DocumentNode) ([]*ui.DocumentNode, bool) {
	if node == nil || node.Kind != ui.DocumentNodeElement {
		return nil, false
	}
	switch node.Name {
	case "flow-paragraph", "paragraph":
		if len(node.Children) == 0 {
			return nil, false
		}
		return node.Children, true
	default:
		return nil, false
	}
}

func listItemBlockNode(node *Node, ctx *renderContext, marker string) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	hasMarker := strings.TrimSpace(marker) != ""
	if !hasMarker {
		if len(children) == 0 {
			return nil
		}
		return ui.NewDocumentElement("list-item", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 6, 10)
			style.SetForeground(ui.Black)
			style.SetFontSize(13)
			style.SetLineHeight(18)
			style.SetContain(ui.ContainPaint)
		}), children...)
	}
	itemChildren := make([]*ui.DocumentNode, 0, len(children)+2)
	bodyChildren := children
	if len(children) > 0 {
		if lead, ok := listLeadChildren(children[0]); ok {
			rowChildren := make([]*ui.DocumentNode, 0, len(lead)+1)
			rowChildren = append(rowChildren, listMarkerNode(marker))
			rowChildren = append(rowChildren, lead...)
			itemChildren = append(itemChildren, ui.NewDocumentElement("list-item-row", styled(func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetContain(ui.ContainPaint)
			}), rowChildren...))
			bodyChildren = children[1:]
		}
	}
	if len(itemChildren) == 0 {
		itemChildren = append(itemChildren, ui.NewDocumentElement("list-item-row", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetContain(ui.ContainPaint)
		}), listMarkerNode(marker)))
	}
	if len(bodyChildren) > 0 {
		itemChildren = append(itemChildren, ui.NewDocumentElement("list-item-body", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 0, 18)
			style.SetContain(ui.ContainPaint)
		}), bodyChildren...))
	}
	return ui.NewDocumentElement("list-item", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 10)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	}), itemChildren...)
}

func standaloneLinkNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	link := inlineLinkNodeFromAnchor(node, ctx)
	if link == nil {
		return nil
	}
	return ui.NewDocumentElement("standalone-link", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 16, 0)
		style.SetForeground(0x333333)
		style.SetFontSize(defaultPageFontSize)
		style.SetLineHeight(defaultPageLineHeight)
		style.SetContain(ui.ContainPaint)
	}), link)
}

func anchorHasStructuredContent(node *Node) bool {
	if node == nil {
		return false
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		if child.Type == ElementNode && !nodeParticipatesInInlineFlow(child) {
			return true
		}
	}
	return false
}

func structuredLinkContainerNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	baseURL := ""
	if ctx != nil {
		baseURL = ctx.baseURL
	}
	href := resolveURL(baseURL, node.Attrs["href"])
	if href == "" {
		return genericBlockContainerNode(node, ctx)
	}
	children := buildFlowNodes(node, ctx)
	style := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetContain(ui.ContainPaint)
		style.SetForeground(ui.Blue)
	})
	applyPageNodeStyles(&style, node, ctx)
	inheritTextPropertiesFromStyle(children, style)
	link := ui.NewDocumentElement("structured-link", style, children...)
	link.Focusable = true
	link.StyleHover = styled(func(style *ui.Style) {
		style.SetForeground(ui.Teal)
	})
	link.StyleActive = styled(func(style *ui.Style) {
		style.SetForeground(ui.Navy)
	})
	link.StyleFocus = styled(func(style *ui.Style) {
		style.SetOutline(1, ui.Blue)
		style.SetOutlineOffset(1)
	})
	applyPageInteractionStyles(link, node, ctx)
	applyPageInteractionTextStyles(link)
	if ctx != nil && ctx.openURL != nil {
		link.OnClick = func() {
			ctx.openURL(href)
		}
	}
	bindLinkStatusHint(link, href, ctx)
	return link
}

func inheritTextPropertiesFromStyle(nodes []*ui.DocumentNode, inherited ui.Style) {
	for _, node := range nodes {
		inheritDocumentNodeTextProperties(node, inherited)
	}
}

func inheritDocumentNodeTextProperties(node *ui.DocumentNode, inherited ui.Style) {
	if node == nil {
		return
	}
	if color, ok := inherited.GetForeground(); ok {
		if _, set := node.Style.GetForeground(); !set {
			node.Style.SetForeground(color)
		}
	}
	if path, ok := inherited.GetFontPath(); ok {
		if _, set := node.Style.GetFontPath(); !set {
			node.Style.SetFontPath(path)
		}
	}
	if size, ok := inherited.GetFontSize(); ok {
		if _, set := node.Style.GetFontSize(); !set {
			node.Style.SetFontSize(size)
		}
	}
	if lineHeight, ok := inherited.GetLineHeight(); ok {
		if _, set := node.Style.GetLineHeight(); !set {
			node.Style.SetLineHeight(lineHeight)
		}
	}
	if align, ok := inherited.GetTextAlign(); ok {
		if _, set := node.Style.GetTextAlign(); !set {
			node.Style.SetTextAlign(align)
		}
	}
	if decoration, ok := inherited.GetTextDecoration(); ok {
		if _, set := node.Style.GetTextDecoration(); !set {
			node.Style.SetTextDecoration(decoration)
		}
	}
	if whiteSpace, ok := inherited.GetWhiteSpace(); ok {
		if _, set := node.Style.GetWhiteSpace(); !set {
			node.Style.SetWhiteSpace(whiteSpace)
		}
	}
	next := inherited
	copyPageTextProperties(&next, node.Style)
	for _, child := range node.Children {
		inheritDocumentNodeTextProperties(child, next)
	}
}

func genericBlockContainerNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
	style := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetContain(ui.ContainPaint)
		if node.Tag == "body" {
			style.SetMargin(defaultBodyMargin)
		}
	})
	if background := resolvedNodeBackgroundState(node, ctx); background.hasImage && (node.Tag == "html" || node.Tag == "body") {
		applyPageNodeStylesExceptBackground(&style, node, ctx)
	} else {
		applyPageNodeStyles(&style, node, ctx)
	}
	inheritTextPropertiesFromStyle(children, style)
	container := ui.NewDocumentElement("container-"+node.Tag, style, children...)
	applyPageInteractionStyles(container, node, ctx)
	return container
}

func buildInlineNodes(node *Node, ctx *renderContext, baseStyle ui.Style) []*ui.DocumentNode {
	builder := inlinePieceBuilder{}
	inlineStyle := inlineTextStyleFromStyle(baseStyle)
	for _, child := range node.Children {
		collectInlinePieces(&builder, child, ctx, inlineStyle)
	}
	return inlineNodesFromPieces(builder.pieces, ctx)
}

func collectInlinePieces(builder *inlinePieceBuilder, node *Node, ctx *renderContext, currentStyle inlineTextStyle) {
	if builder == nil || node == nil {
		return
	}
	switch node.Type {
	case CommentNode:
		return
	case TextNode:
		builder.appendText(node.Text, currentStyle)
		return
	case DocumentNode:
		for _, child := range node.Children {
			collectInlinePieces(builder, child, ctx, currentStyle)
		}
		return
	case ElementNode:
	default:
		return
	}
	if pageNodeDisplayNone(node, ctx) {
		return
	}
	nextStyle := currentStyle
	applyPageTextPropertiesToInlineStyle(&nextStyle, node, ctx)
	applyInlineSemanticStyle(&nextStyle, node)

	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link", "source", "template":
		return
	case "br":
		builder.appendBreak(nextStyle)
		return
	case "a":
		baseURL := ""
		if ctx != nil {
			baseURL = ctx.baseURL
		}
		builder.appendLink(node, collectNodeText(node, false), resolveURL(baseURL, node.Attrs["href"]), nextStyle)
		return
	case "code":
		builder.appendCode(collectNodeTextPreserve(node, false), inlineCodeTextStyle(nextStyle))
		return
	case "img":
		label := normalizeBlockText(node.Attrs["alt"])
		if label == "" {
			label = displayURL(resolveRenderedImageURL(node, ctx))
		}
		if label == "" {
			label = displayURL(strings.TrimSpace(node.Attrs["src"]))
		}
		builder.appendImage(node, label, nextStyle)
		return
	case "button", "textarea", "select", "progress":
		builder.appendControl(node, nextStyle)
		return
	case "input":
		if htmlInputType(node) == "hidden" {
			return
		}
		builder.appendControl(node, nextStyle)
		return
	default:
		for _, child := range node.Children {
			collectInlinePieces(builder, child, ctx, nextStyle)
		}
	}
}

func (builder *inlinePieceBuilder) appendText(raw string, style inlineTextStyle) {
	if builder == nil {
		return
	}
	words := strings.Fields(raw)
	if len(words) == 0 {
		if strings.TrimSpace(raw) == "" && len(builder.pieces) > 0 {
			builder.needSpace = true
		}
		return
	}
	for i, word := range words {
		if builder.needSpace || i > 0 {
			builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " ", style: style})
			builder.needSpace = false
		}
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: word, style: style})
	}
	if len(raw) > 0 && isSpaceByte(raw[len(raw)-1]) {
		builder.needSpace = true
	}
}

func (builder *inlinePieceBuilder) appendLink(node *Node, label string, href string, style inlineTextStyle) {
	if builder == nil || strings.TrimSpace(href) == "" {
		return
	}
	label = normalizeBlockText(label)
	if label == "" {
		label = displayURL(href)
	}
	if label == "" {
		return
	}
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " ", style: style})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceLink, text: label, href: href, node: node, style: style})
}

func (builder *inlinePieceBuilder) appendCode(text string, style inlineTextStyle) {
	if builder == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " ", style: style})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceCode, text: text, style: style})
}

func (builder *inlinePieceBuilder) appendImage(node *Node, label string, style inlineTextStyle) {
	if builder == nil {
		return
	}
	label = normalizeBlockText(label)
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " ", style: style})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceImage, text: label, node: node, style: style})
}

func (builder *inlinePieceBuilder) appendBreak(style inlineTextStyle) {
	if builder == nil {
		return
	}
	builder.needSpace = false
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceBreak, style: style})
}

func (builder *inlinePieceBuilder) appendControl(node *Node, style inlineTextStyle) {
	if builder == nil || node == nil {
		return
	}
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " ", style: style})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{
		kind: inlinePieceControl,
		node: node,
	})
}

func inlineNodesFromPieces(pieces []inlinePiece, ctx *renderContext) []*ui.DocumentNode {
	if len(pieces) == 0 {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, len(pieces))
	for _, piece := range pieces {
		baseStyle := piece.style.uiStyle()
		switch piece.kind {
		case inlinePieceText:
			nodes = append(nodes, inlineTextNode(piece.text, baseStyle))
		case inlinePieceLink:
			if link := inlineLinkNode(piece.text, piece.href, piece.node, baseStyle, ctx); link != nil {
				nodes = append(nodes, link)
			}
		case inlinePieceCode:
			if code := inlineCodeNode(piece.text, baseStyle); code != nil {
				nodes = append(nodes, code)
			}
		case inlinePieceImage:
			if image := inlineImageNode(piece.node, piece.text, baseStyle, ctx); image != nil {
				nodes = append(nodes, image)
			}
		case inlinePieceBreak:
			nodes = append(nodes, inlineBreakNode(baseStyle))
		case inlinePieceControl:
			if control := inlineControlNode(piece.node, ctx); control != nil {
				nodes = append(nodes, control)
			}
		}
	}
	return nodes
}

func inlineTextStyleFromStyle(style ui.Style) inlineTextStyle {
	value := inlineTextStyle{}
	if path, ok := style.GetFontPath(); ok {
		value.fontPath = strings.TrimSpace(path)
	}
	if size, ok := style.GetFontSize(); ok && size > 0 {
		value.fontSize = size
	}
	if lineHeight, ok := style.GetLineHeight(); ok && lineHeight > 0 {
		value.lineHeight = lineHeight
	}
	if color, ok := style.GetForeground(); ok {
		value.foreground = color
		value.hasForeground = true
	}
	if decoration, ok := style.GetTextDecoration(); ok {
		value.textDecoration = decoration
		value.hasTextDecoration = true
	}
	if whiteSpace, ok := style.GetWhiteSpace(); ok {
		value.whiteSpace = whiteSpace
		value.hasWhiteSpace = true
	}
	return value
}

func (style inlineTextStyle) uiStyle() ui.Style {
	value := ui.Style{}
	value.SetDisplay(ui.DisplayInline)
	if style.fontPath != "" {
		value.SetFontPath(style.fontPath)
	}
	if style.fontSize > 0 {
		value.SetFontSize(style.fontSize)
	}
	if style.lineHeight > 0 {
		value.SetLineHeight(style.lineHeight)
	}
	if style.hasForeground {
		value.SetForeground(style.foreground)
	}
	if style.hasTextDecoration {
		value.SetTextDecoration(style.textDecoration)
	}
	if style.hasWhiteSpace {
		value.SetWhiteSpace(style.whiteSpace)
	}
	return value
}

func copyPageTextPropertiesToInlineStyle(target *inlineTextStyle, source ui.Style) {
	if target == nil {
		return
	}
	if path, ok := source.GetFontPath(); ok {
		target.fontPath = strings.TrimSpace(path)
	}
	if size, ok := source.GetFontSize(); ok && size > 0 {
		target.fontSize = size
	}
	if lineHeight, ok := source.GetLineHeight(); ok && lineHeight > 0 {
		target.lineHeight = lineHeight
	}
	if color, ok := source.GetForeground(); ok {
		target.foreground = color
		target.hasForeground = true
	}
	if decoration, ok := source.GetTextDecoration(); ok {
		target.textDecoration = decoration
		target.hasTextDecoration = true
	}
	if whiteSpace, ok := source.GetWhiteSpace(); ok {
		target.whiteSpace = whiteSpace
		target.hasWhiteSpace = true
	}
}

func applyPageTextPropertiesToInlineStyle(style *inlineTextStyle, node *Node, ctx *renderContext) {
	if style == nil || node == nil {
		return
	}
	resolved := ui.Style{}
	applyPageNodeStyles(&resolved, node, ctx)
	copyPageTextPropertiesToInlineStyle(style, resolved)
}

func styleFontPath(style ui.Style) string {
	if path, ok := style.GetFontPath(); ok {
		return strings.TrimSpace(path)
	}
	return ""
}

func inlineStyleFontSize(style inlineTextStyle) int {
	if style.fontSize > 0 {
		return style.fontSize
	}
	return defaultPageFontSize
}

func bundledFontVariantInfo(path string) (string, bool, bool, bool) {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if path == "" || !strings.HasPrefix(path, bundledFontDir+"/") {
		return "", false, false, false
	}
	key := normalizeCSSFontFamilyName(strings.TrimSuffix(pathpkg.Base(path), pathpkg.Ext(path)))
	if key == "" {
		return "", false, false, false
	}
	family := key
	bold := false
	italic := false
	switch {
	case strings.HasSuffix(key, "bolditalic"):
		family = strings.TrimSuffix(key, "bolditalic")
		bold = true
		italic = true
	case strings.HasSuffix(key, "italicbold"):
		family = strings.TrimSuffix(key, "italicbold")
		bold = true
		italic = true
	case strings.HasSuffix(key, "bold"):
		family = strings.TrimSuffix(key, "bold")
		bold = true
	case strings.HasSuffix(key, "italic"):
		family = strings.TrimSuffix(key, "italic")
		italic = true
	case strings.HasSuffix(key, "regular"):
		family = strings.TrimSuffix(key, "regular")
	}
	if family == "" {
		family = key
	}
	return family, bold, italic, true
}

func isBundledMonospaceFamilyKey(key string) bool {
	switch key {
	case "gomono", "mono", "monospace", "uimonospace", "fixed",
		"consolas", "couriernew", "dejavusansmono", "jetbrainsmono",
		"liberationmono", "menlo", "monaco", "robotomono", "sfmono":
		return true
	default:
		return false
	}
}

func lookupBundledFontVariantPath(key string, bold bool, italic bool) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	candidates := make([]string, 0, 4)
	switch {
	case bold && italic:
		candidates = append(candidates, key+"bolditalic", key+"italicbold")
	case bold:
		candidates = append(candidates, key+"bold")
	case italic:
		candidates = append(candidates, key+"italic")
	}
	candidates = append(candidates, key, key+"regular")
	for _, candidate := range candidates {
		if path := lookupBundledFontFamilyPath(candidate); path != "" {
			return path
		}
	}
	return ""
}

func defaultSemanticFontVariantPath(preferMono bool, bold bool, italic bool) string {
	if preferMono {
		if path := lookupBundledFontVariantPath("gomono", bold, italic); path != "" {
			return path
		}
		if path := lookupBundledFontFamilyPath("gomono"); path != "" {
			return path
		}
		return webMonoFontPath
	}
	if path := lookupBundledFontVariantPath("go", bold, italic); path != "" {
		return path
	}
	if path := lookupBundledFontFamilyPath("go"); path != "" {
		return path
	}
	if bold || italic {
		return webSansFontPath
	}
	return webSansFontPath
}

func resolveRegularSemanticFontPath(current string, preferMono bool) string {
	family, _, _, bundled := bundledFontVariantInfo(current)
	if bundled {
		if preferMono && !isBundledMonospaceFamilyKey(family) {
			return defaultSemanticFontVariantPath(true, false, false)
		}
		if path := lookupBundledFontVariantPath(family, false, false); path != "" {
			return path
		}
		if path := lookupBundledFontFamilyPath(family); path != "" {
			return path
		}
	}
	if current != "" && !preferMono {
		return current
	}
	return defaultSemanticFontVariantPath(preferMono, false, false)
}

func resolveSemanticFontPath(current string, preferMono bool, wantBold bool, wantItalic bool) string {
	family, currentBold, currentItalic, bundled := bundledFontVariantInfo(current)
	targetBold := currentBold || wantBold
	targetItalic := currentItalic || wantItalic
	if bundled {
		if path := lookupBundledFontVariantPath(family, targetBold, targetItalic); path != "" {
			return path
		}
		if preferMono && !isBundledMonospaceFamilyKey(family) {
			if path := lookupBundledFontVariantPath("gomono", targetBold, targetItalic); path != "" {
				return path
			}
		}
		if current != "" {
			return current
		}
	}
	if current != "" && !preferMono {
		return current
	}
	return defaultSemanticFontVariantPath(preferMono, targetBold, targetItalic)
}

func scaleInlineFontSize(style *inlineTextStyle, factor float64, minSize int) {
	if style == nil || factor <= 0 {
		return
	}
	size := roundCSSPixels(float64(inlineStyleFontSize(*style)) * factor)
	if size < minSize {
		size = minSize
	}
	if size <= 0 {
		return
	}
	style.fontSize = size
}

func applyInlineSemanticStyle(style *inlineTextStyle, node *Node) {
	if style == nil || node == nil {
		return
	}
	switch node.Tag {
	case "b", "strong":
		style.fontPath = resolveSemanticFontPath(style.fontPath, false, true, false)
	case "i", "em", "cite", "dfn", "var":
		style.fontPath = resolveSemanticFontPath(style.fontPath, false, false, true)
	case "u", "ins":
		style.textDecoration = ui.TextDecorationUnderline
		style.hasTextDecoration = true
	case "tt", "kbd", "samp":
		style.fontPath = resolveRegularSemanticFontPath(style.fontPath, true)
	case "small":
		scaleInlineFontSize(style, 0.85, 10)
	case "big":
		scaleInlineFontSize(style, 1.15, 0)
	}
}

func inlineCodeTextStyle(baseStyle inlineTextStyle) inlineTextStyle {
	style := baseStyle
	style.fontPath = resolveRegularSemanticFontPath(style.fontPath, true)
	return style
}

func inlineControlNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	switch node.Tag {
	case "button":
		return htmlButtonNode(node, ctx)
	case "input":
		return htmlInputNode(node, ctx)
	case "textarea":
		return htmlTextareaNode(node, ctx)
	case "select":
		return htmlSelectNode(node, ctx)
	case "progress":
		return htmlProgressNode(node)
	default:
		return nil
	}
}

func inlineTextNode(text string, baseStyle ui.Style) *ui.DocumentNode {
	if text == "" {
		return nil
	}
	style := baseStyle
	style.SetDisplay(ui.DisplayInline)
	return ui.NewDocumentText(text, style)
}

func inlineTextTokens(text string, baseStyle ui.Style) []*ui.DocumentNode {
	text = normalizeBlockText(text)
	if text == "" {
		return nil
	}
	parts := strings.Split(text, " ")
	nodes := make([]*ui.DocumentNode, 0, len(parts)*2)
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i > 0 {
			if space := inlineTextNode(" ", baseStyle); space != nil {
				nodes = append(nodes, space)
			}
		}
		if token := inlineTextNode(part, baseStyle); token != nil {
			nodes = append(nodes, token)
		}
	}
	return nodes
}

func inlineLinkNodeFromAnchor(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	baseURL := ""
	if ctx != nil {
		baseURL = ctx.baseURL
	}
	href := resolveURL(baseURL, node.Attrs["href"])
	if href == "" {
		return nil
	}
	label := collectNodeText(node, false)
	if label == "" {
		label = displayURL(href)
	}
	return inlineLinkNode(label, href, node, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(0x333333)
		style.SetFontSize(defaultPageFontSize)
		style.SetLineHeight(defaultPageLineHeight)
	}), ctx)
}

func inlineLinkNode(label string, href string, node *Node, baseStyle ui.Style, ctx *renderContext) *ui.DocumentNode {
	label = normalizeBlockText(label)
	if label == "" {
		label = displayURL(href)
	}
	if label == "" {
		return nil
	}
	style := baseStyle
	style.SetDisplay(ui.DisplayInlineBlock)
	style.SetForeground(ui.Blue)
	style.SetTextDecoration(ui.TextDecorationUnderline)
	if node != nil {
		applyPageNodeStyles(&style, node, ctx)
	}
	textStyle := baseStyle
	textStyle.SetDisplay(ui.DisplayInline)
	textStyle.SetForeground(ui.Blue)
	textStyle.SetTextDecoration(ui.TextDecorationUnderline)
	if node != nil {
		applyPageTextProperties(&textStyle, node, ctx)
	}
	link := ui.NewDocumentElement("inline-link", style, inlineTextTokens(label, textStyle)...)
	link.Focusable = true
	link.StyleHover = styled(func(style *ui.Style) {
		style.SetForeground(ui.Teal)
	})
	link.StyleActive = styled(func(style *ui.Style) {
		style.SetForeground(ui.Navy)
	})
	link.StyleFocus = styled(func(style *ui.Style) {
		style.SetOutline(1, ui.Blue)
		style.SetOutlineOffset(1)
	})
	applyPageInteractionStyles(link, node, ctx)
	applyPageInteractionTextStyles(link)
	if ctx != nil && ctx.openURL != nil && href != "" {
		link.OnClick = func() {
			ctx.openURL(href)
		}
	}
	bindLinkStatusHint(link, href, ctx)
	return link
}

func inlineCodeNode(text string, baseStyle ui.Style) *ui.DocumentNode {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	style := baseStyle
	style.SetDisplay(ui.DisplayInline)
	style.SetFontPath(resolveRegularSemanticFontPath(styleFontPath(baseStyle), true))
	return ui.NewDocumentText(text, style)
}

func inlineImageNode(node *Node, label string, baseStyle ui.Style, ctx *renderContext) *ui.DocumentNode {
	label = normalizeBlockText(label)
	image := resolveRenderedImage(node, ctx)
	reason := resolveRenderedImageError(node, ctx)
	style := baseStyle
	style.SetDisplay(ui.DisplayInlineBlock)
	style.SetMargin(0, 1)
	style.SetContain(ui.ContainPaint)
	if node != nil {
		applyPageNodeStyles(&style, node, ctx)
		applyPresentationalNodeAttrs(&style, node)
	}
	width, height := resolveImageBoxSize(style, node, image, 16, 0)
	if width > 0 {
		style.SetWidth(width)
	}
	if height > 0 {
		style.SetHeight(height)
	}
	imageNode := ui.NewDocumentElement("inline-image", style)
	imageNode.Image = image
	if image != nil {
		return imageNode
	}
	style.SetBorderRadius(4)
	style.SetBackground(0xEEF2F6)
	style.SetBorder(1, 0xD8DEE4)
	if reason != "" {
		style.SetBackground(0xFDECEC)
		style.SetBorder(1, 0xD93025)
	}
	imageNode.Style = style
	if width <= 24 && height <= 24 {
		return imageNode
	}
	childStyle := baseStyle
	childStyle.SetDisplay(ui.DisplayInline)
	childStyle.SetForeground(ui.Gray)
	childStyle.SetFontSize(11)
	childStyle.SetLineHeight(15)
	if label == "" {
		label = "image"
	}
	if reason != "" {
		label += " (" + reason + ")"
	}
	imageNode.Append(ui.NewDocumentText("[image] "+label, childStyle))
	return imageNode
}

func inlineBreakNode(baseStyle ui.Style) *ui.DocumentNode {
	style := baseStyle
	style.SetDisplay(ui.DisplayBlock)
	style.SetHeight(0)
	return ui.NewDocumentElement("inline-break", style)
}

func preformattedNode(text string) *ui.DocumentNode {
	return ui.NewDocumentElement("pre", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(8)
		style.SetBackground(0xF6F8FA)
		style.SetBorder(1, 0xD8DEE4)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(text, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Navy)
		style.SetFontSize(12)
		style.SetFontPath(webMonoFontPath)
		style.SetWhiteSpace(ui.WhiteSpacePreWrap)
		style.SetLineHeight(17)
	})))
}

func tableBlockNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := make([]*ui.DocumentNode, 0, len(node.Children)+1)
	for _, child := range node.Children {
		if child == nil || child.Type != ElementNode {
			continue
		}
		switch child.Tag {
		case "caption":
			if caption := figureCaptionBlockNode(child, ctx); caption != nil {
				children = append(children, caption)
			}
		case "thead", "tbody", "tfoot":
			appendTableRowNodes(&children, child, ctx)
		case "tr":
			if row := tableRowNode(child, ctx); row != nil {
				children = append(children, row)
			}
		}
	}
	if len(children) == 0 {
		children = buildFlowNodes(node, ctx)
	}
	if len(children) == 0 {
		return nil
	}
	table := ui.NewDocumentElement("table", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetContain(ui.ContainPaint)
	}), children...)
	applyPageNodeStyles(&table.Style, node, ctx)
	applyPresentationalNodeAttrs(&table.Style, node)
	return table
}

func appendTableRowNodes(out *[]*ui.DocumentNode, node *Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	for _, child := range node.Children {
		if child == nil || child.Type != ElementNode {
			continue
		}
		switch child.Tag {
		case "tr":
			if row := tableRowNode(child, ctx); row != nil {
				*out = append(*out, row)
			}
		case "thead", "tbody", "tfoot":
			appendTableRowNodes(out, child, ctx)
		}
	}
}

func tableRowNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	cells := make([]*ui.DocumentNode, 0, len(node.Children))
	for _, child := range node.Children {
		if child == nil || child.Type != ElementNode {
			continue
		}
		switch child.Tag {
		case "td", "th":
			if cell := tableCellNode(child, ctx); cell != nil {
				cells = append(cells, cell)
			}
		}
	}
	if len(cells) == 0 {
		text := collectNodeText(node, false)
		if text == "" {
			return nil
		}
		return paragraphNode(text)
	}
	row := ui.NewDocumentElement("table-row", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetContain(ui.ContainPaint)
	}), cells...)
	applyPageNodeStyles(&row.Style, node, ctx)
	applyPresentationalNodeAttrs(&row.Style, node)
	applySimpleTableRowLayout(row, node, ctx)
	return row
}

func applySimpleTableRowLayout(row *ui.DocumentNode, source *Node, ctx *renderContext) {
	if row == nil || source == nil || len(row.Children) != 3 {
		return
	}
	left := row.Children[0]
	center := row.Children[1]
	right := row.Children[2]
	if left == nil || center == nil || right == nil {
		return
	}
	leftWidth, leftOK := left.Style.GetWidth()
	rightWidth, rightOK := right.Style.GetWidth()
	if !leftOK || !rightOK || leftWidth <= 0 || rightWidth <= 0 {
		return
	}
	leftNode := nthTableCellElement(source, 0)
	centerNode := nthTableCellElement(source, 1)
	rightNode := nthTableCellElement(source, 2)
	leftHeight := estimateDocumentNodeOuterHeight(left)
	centerHeight := estimateDocumentNodeOuterHeight(center)
	rightHeight := estimateDocumentNodeOuterHeight(right)
	rowHeight := leftHeight
	if rowHeight < centerHeight {
		rowHeight = centerHeight
	}
	if rowHeight < rightHeight {
		rowHeight = rightHeight
	}
	row.Style.SetPosition(ui.PositionRelative)
	if rowHeight <= 0 {
		if leftWidth > rightWidth {
			rowHeight = leftWidth
		} else {
			rowHeight = rightWidth
		}
	}
	row.Style.SetMinHeight(rowHeight)

	left.Style.SetPosition(ui.PositionAbsolute)
	left.Style.SetLeft(0)
	left.Style.SetTop(tableCellVerticalOffset(resolvedNodeVerticalAlign(leftNode, ctx), leftHeight, rowHeight))
	left.Style.SetMargin(0)

	right.Style.SetPosition(ui.PositionAbsolute)
	right.Style.SetRight(0)
	right.Style.SetTop(tableCellVerticalOffset(resolvedNodeVerticalAlign(rightNode, ctx), rightHeight, rowHeight))
	right.Style.SetMargin(0)

	center.Style.SetDisplay(ui.DisplayBlock)
	center.Style.SetMarginLeft(leftWidth)
	center.Style.SetMarginRight(rightWidth)
	if centerHeight > 0 {
		center.Style.SetMinHeight(centerHeight)
	}
	if top := tableCellVerticalOffset(resolvedNodeVerticalAlign(centerNode, ctx), centerHeight, rowHeight); top > 0 {
		center.Style.SetMarginTop(top)
	}
}

func tableCellNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	header := node.Tag == "th"
	children := buildFlowNodes(node, ctx)
	children = unwrapSingleMediaParagraph(children)
	if len(children) == 0 {
		text := collectNodeText(node, false)
		if text == "" {
			return nil
		}
		textStyle := paragraphInlineStyle()
		if header {
			textStyle.SetForeground(ui.Navy)
		}
		applyPageTextProperties(&textStyle, node, ctx)
		children = inlineTextTokens(text, textStyle)
	}
	cell := ui.NewDocumentElement("table-cell", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetContain(ui.ContainPaint)
		if header {
			style.SetForeground(ui.Navy)
		}
	}), children...)
	applyPageNodeStyles(&cell.Style, node, ctx)
	applyPresentationalNodeAttrs(&cell.Style, node)
	inheritTextPropertiesFromStyle(cell.Children, cell.Style)
	return cell
}

func unwrapSingleMediaParagraph(children []*ui.DocumentNode) []*ui.DocumentNode {
	if len(children) != 1 || children[0] == nil || children[0].Name != "flow-paragraph" {
		return children
	}
	paragraph := children[0]
	if len(paragraph.Children) == 0 {
		return children
	}
	for _, child := range paragraph.Children {
		if child == nil || child.Kind != ui.DocumentNodeElement || child.Name != "inline-image" {
			return children
		}
		child.Style.SetDisplay(ui.DisplayBlock)
		child.Style.SetMargin(0)
	}
	return paragraph.Children
}

func nthTableCellElement(row *Node, index int) *Node {
	if row == nil || index < 0 {
		return nil
	}
	current := 0
	for _, child := range row.Children {
		if child == nil || child.Type != ElementNode {
			continue
		}
		if child.Tag != "td" && child.Tag != "th" {
			continue
		}
		if current == index {
			return child
		}
		current++
	}
	return nil
}

func tableCellVerticalOffset(align string, cellHeight int, rowHeight int) int {
	if cellHeight <= 0 || rowHeight <= cellHeight {
		return 0
	}
	switch align {
	case "bottom":
		return rowHeight - cellHeight
	case "middle":
		return (rowHeight - cellHeight) / 2
	default:
		return 0
	}
}

func estimateDocumentNodeOuterHeight(node *ui.DocumentNode) int {
	if node == nil {
		return 0
	}
	contentHeight := 0
	if height, ok := node.Style.GetHeight(); ok && height > 0 {
		contentHeight = height
	} else if node.Image != nil && node.Image.Valid() && node.Image.Height > 0 {
		contentHeight = node.Image.Height
	} else if node.Kind == ui.DocumentNodeText {
		if lineHeight, ok := node.Style.GetLineHeight(); ok && lineHeight > 0 {
			contentHeight = lineHeight
		} else if fontSize, ok := node.Style.GetFontSize(); ok && fontSize > 0 {
			contentHeight = defaultLineHeightForFontSize(fontSize)
		} else {
			contentHeight = defaultPageLineHeight
		}
	} else if len(node.Children) > 0 {
		display, _ := node.Style.GetDisplay()
		if display == ui.DisplayBlock {
			for _, child := range node.Children {
				childHeight := estimateDocumentNodeOuterHeight(child)
				if childHeight > 0 {
					childHeight += verticalMarginsForStyle(child.Style)
					contentHeight += childHeight
				}
			}
		} else {
			for _, child := range node.Children {
				childHeight := estimateDocumentNodeOuterHeight(child)
				if childHeight > 0 {
					childHeight += verticalMarginsForStyle(child.Style)
					if childHeight > contentHeight {
						contentHeight = childHeight
					}
				}
			}
		}
	}
	return contentHeight + verticalChromeForStyle(node.Style)
}

func verticalChromeForStyle(style ui.Style) int {
	total := 0
	if padding, ok := style.GetPadding(); ok {
		total += padding.Top + padding.Bottom
	}
	if border, ok := style.GetBorderTopWidth(); ok {
		total += border
	}
	if border, ok := style.GetBorderBottomWidth(); ok {
		total += border
	}
	return total
}

func verticalMarginsForStyle(style ui.Style) int {
	if margin, ok := style.GetMargin(); ok {
		return margin.Top + margin.Bottom
	}
	return 0
}

func listItemNode(text string) *ui.DocumentNode {
	return ui.NewDocumentText("- "+text, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 10)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}))
}

func separatorNode() *ui.DocumentNode {
	return ui.NewDocumentElement("separator", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(2, 0, 8, 0)
		style.SetHeight(1)
		style.SetBackground(ui.Silver)
		style.SetContain(ui.ContainPaint)
	}))
}

func applyPresentationalNodeAttrs(style *ui.Style, node *Node) {
	if style == nil || node == nil {
		return
	}
	if width := attrInt(node, "width", 0); width > 0 {
		style.SetWidth(width)
	}
	if height := attrInt(node, "height", 0); height > 0 {
		style.SetHeight(height)
	}
	if color, ok := parseHTMLColor(attrValue(node, "bgcolor")); ok {
		style.SetBackground(color)
	}
	if color, ok := parseHTMLColor(attrValue(node, "bg")); ok {
		style.SetBackground(color)
	}
	if color, ok := parseHTMLColor(attrValue(node, "color")); ok {
		style.SetForeground(color)
	}
}

func resolveFallbackBoxSize(style ui.Style, node *Node, fallback int) (int, int) {
	width, widthOK := style.GetWidth()
	height, heightOK := style.GetHeight()
	if !widthOK && node != nil {
		if value := attrInt(node, "width", 0); value > 0 {
			width = value
			widthOK = true
		}
	}
	if !heightOK && node != nil {
		if value := attrInt(node, "height", 0); value > 0 {
			height = value
			heightOK = true
		}
	}
	if !heightOK {
		if lineHeight, ok := style.GetLineHeight(); ok && lineHeight > 0 {
			height = lineHeight
			heightOK = true
		}
	}
	if !widthOK && heightOK {
		width = height
		widthOK = true
	}
	if !heightOK && widthOK {
		height = width
		heightOK = true
	}
	if !widthOK {
		width = fallback
	}
	if !heightOK {
		height = fallback
	}
	return width, height
}

type imageSourceCandidate struct {
	url        string
	width      int
	density    float64
	hasWidth   bool
	hasDensity bool
}

func resolveRenderedImageURL(node *Node, ctx *renderContext) string {
	if node == nil {
		return ""
	}
	if node.Tag == "img" {
		if picture := pictureParentNode(node); picture != nil {
			if sourceURL := resolvePictureSourceURL(picture, node, ctx); sourceURL != "" {
				return sourceURL
			}
		}
	}
	if sourceURL := selectImageSourceURL(node, attrValue(node, "srcset"), ctx); sourceURL != "" {
		return sourceURL
	}
	baseURL := ""
	if ctx != nil {
		baseURL = ctx.baseURL
	}
	return imageResourceURL(baseURL, attrValue(node, "src"))
}

func pictureParentNode(node *Node) *Node {
	if node == nil || node.Parent == nil {
		return nil
	}
	if node.Parent.Type == ElementNode && node.Parent.Tag == "picture" {
		return node.Parent
	}
	return nil
}

func resolvePictureSourceURL(picture *Node, fallback *Node, ctx *renderContext) string {
	if picture == nil {
		return ""
	}
	for _, child := range picture.Children {
		if child == nil || child.Type != ElementNode {
			continue
		}
		switch child.Tag {
		case "source":
			if !pictureSourceMatches(child, ctx) {
				continue
			}
			if sourceURL := selectImageSourceURL(child, attrValue(child, "srcset"), ctx); sourceURL != "" {
				return sourceURL
			}
			baseURL := ""
			if ctx != nil {
				baseURL = ctx.baseURL
			}
			if sourceURL := imageResourceURL(baseURL, attrValue(child, "src")); sourceURL != "" {
				return sourceURL
			}
		case "img":
			if child == fallback {
				return ""
			}
		}
	}
	return ""
}

func pictureSourceMatches(node *Node, ctx *renderContext) bool {
	if node == nil {
		return false
	}
	typeAttr := strings.ToLower(strings.TrimSpace(attrValue(node, "type")))
	if typeAttr != "" && !imageTypeSupported(typeAttr) {
		return false
	}
	media := strings.ToLower(strings.TrimSpace(attrValue(node, "media")))
	if media == "" || media == "all" || media == "screen" || media == "only screen" {
		return true
	}
	condition, ok := parseCSSMediaCondition(media)
	if !ok {
		return false
	}
	layout := cssLayoutContext{fontSize: defaultPageFontSize}
	if ctx != nil {
		layout = ctx.cssLayoutContext()
	}
	return condition.matches(layout)
}

func imageTypeSupported(value string) bool {
	if value == "" {
		return true
	}
	if semicolon := strings.IndexByte(value, ';'); semicolon >= 0 {
		value = value[:semicolon]
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image/*",
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
		"image/webp":
		return true
	default:
		return false
	}
}

func selectImageSourceURL(node *Node, srcset string, ctx *renderContext) string {
	candidates := parseImageSourceCandidates(srcset, ctx)
	if len(candidates) == 0 {
		return ""
	}
	best := candidates[0]
	idealWidth := preferredImageCandidateWidth(node, ctx)
	bestWidthDelta := 0
	bestDensityDelta := 0.0
	for index, candidate := range candidates {
		if index == 0 {
			if candidate.hasWidth {
				bestWidthDelta = absInt(candidate.width - idealWidth)
			}
			if candidate.hasDensity {
				bestDensityDelta = absFloat(candidate.density - 1.0)
			}
			continue
		}
		switch {
		case candidate.hasWidth && best.hasWidth:
			delta := absInt(candidate.width - idealWidth)
			if delta < bestWidthDelta || (delta == bestWidthDelta && candidate.width > best.width) {
				best = candidate
				bestWidthDelta = delta
			}
		case candidate.hasWidth && !best.hasWidth:
			best = candidate
			bestWidthDelta = absInt(candidate.width - idealWidth)
		case candidate.hasDensity && best.hasDensity:
			delta := absFloat(candidate.density - 1.0)
			if delta < bestDensityDelta || (delta == bestDensityDelta && candidate.density > best.density) {
				best = candidate
				bestDensityDelta = delta
			}
		case candidate.hasDensity && !best.hasWidth && !best.hasDensity:
			best = candidate
			bestDensityDelta = absFloat(candidate.density - 1.0)
		}
	}
	return best.url
}

func parseImageSourceCandidates(srcset string, ctx *renderContext) []imageSourceCandidate {
	srcset = strings.TrimSpace(srcset)
	if srcset == "" {
		return nil
	}
	baseURL := ""
	if ctx != nil {
		baseURL = ctx.baseURL
	}
	candidates := make([]imageSourceCandidate, 0, 4)
	for _, chunk := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(chunk))
		if len(fields) == 0 {
			continue
		}
		url := imageResourceURL(baseURL, fields[0])
		if url == "" {
			continue
		}
		candidate := imageSourceCandidate{url: url}
		if len(fields) >= 2 {
			descriptor := strings.ToLower(strings.TrimSpace(fields[1]))
			switch {
			case strings.HasSuffix(descriptor, "w"):
				if value, err := strconv.Atoi(strings.TrimSuffix(descriptor, "w")); err == nil && value > 0 {
					candidate.width = value
					candidate.hasWidth = true
				}
			case strings.HasSuffix(descriptor, "x"):
				if value, err := strconv.ParseFloat(strings.TrimSuffix(descriptor, "x"), 64); err == nil && value > 0 {
					candidate.density = value
					candidate.hasDensity = true
				}
			}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func preferredImageCandidateWidth(node *Node, ctx *renderContext) int {
	style := ui.Style{}
	if node != nil {
		applyPageNodeStyles(&style, node, ctx)
		applyPresentationalNodeAttrs(&style, node)
	}
	if width, ok := style.GetWidth(); ok && width > 0 {
		return width
	}
	if width, ok := style.GetMaxWidth(); ok && width > 0 {
		return width
	}
	if node != nil {
		if width := attrInt(node, "width", 0); width > 0 {
			return width
		}
	}
	if ctx != nil && ctx.viewportWidth > 0 {
		return ctx.viewportWidth
	}
	return 0
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func resolveRenderedImage(node *Node, ctx *renderContext) *ui.DocumentImage {
	if node == nil || ctx == nil || ctx.loadImage == nil {
		return nil
	}
	src := resolveRenderedImageURL(node, ctx)
	if src == "" {
		return nil
	}
	return ctx.loadImage(src)
}

func resolveRenderedImageError(node *Node, ctx *renderContext) string {
	if node == nil || ctx == nil || ctx.imageError == nil {
		return ""
	}
	src := resolveRenderedImageURL(node, ctx)
	if src == "" {
		return ""
	}
	return strings.TrimSpace(ctx.imageError(src))
}

func resolveImageBoxSize(style ui.Style, node *Node, image *ui.DocumentImage, fallback int, maxWidth int) (int, int) {
	width, widthOK := style.GetWidth()
	height, heightOK := style.GetHeight()
	if !widthOK && node != nil {
		if value := attrInt(node, "width", 0); value > 0 {
			width = value
			widthOK = true
		}
	}
	if !heightOK && node != nil {
		if value := attrInt(node, "height", 0); value > 0 {
			height = value
			heightOK = true
		}
	}
	naturalWidth := 0
	naturalHeight := 0
	if image != nil && image.Valid() {
		naturalWidth = image.Width
		naturalHeight = image.Height
	}
	if widthOK && !heightOK && naturalWidth > 0 && naturalHeight > 0 {
		height = (width * naturalHeight) / naturalWidth
		if height < 1 {
			height = 1
		}
		heightOK = true
	}
	if heightOK && !widthOK && naturalWidth > 0 && naturalHeight > 0 {
		width = (height * naturalWidth) / naturalHeight
		if width < 1 {
			width = 1
		}
		widthOK = true
	}
	if !widthOK && naturalWidth > 0 {
		width = naturalWidth
		widthOK = true
	}
	if !heightOK && naturalHeight > 0 {
		height = naturalHeight
		heightOK = true
	}
	if maxWidth > 0 && widthOK && width > maxWidth {
		if heightOK && width > 0 {
			height = (height * maxWidth) / width
		} else if naturalWidth > 0 && naturalHeight > 0 {
			height = (naturalHeight * maxWidth) / naturalWidth
			heightOK = true
		}
		if height < 1 {
			height = 1
		}
		width = maxWidth
	}
	if !widthOK {
		width = fallback
	}
	if !heightOK {
		if naturalHeight > 0 {
			height = naturalHeight
		} else {
			height = fallback
		}
	}
	if width < 1 {
		width = fallback
	}
	if height < 1 {
		height = fallback
	}
	return width, height
}

func imageFallbackNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	image := resolveRenderedImage(node, ctx)
	label := normalizeBlockText(node.Attrs["alt"])
	resolvedURL := resolveRenderedImageURL(node, ctx)
	if label == "" {
		label = displayURL(resolvedURL)
	}
	if label == "" {
		label = displayURL(strings.TrimSpace(node.Attrs["src"]))
	}
	if reason := resolveRenderedImageError(node, ctx); reason != "" {
		if label == "" {
			label = reason
		} else {
			label += " (" + reason + ")"
		}
	}
	if label == "" && image == nil {
		return nil
	}
	style := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetContain(ui.ContainPaint)
	})
	applyPageNodeStyles(&style, node, ctx)
	applyPresentationalNodeAttrs(&style, node)
	width, height := resolveImageBoxSize(style, node, image, 96, ctx.viewportWidth)
	style.SetWidth(width)
	style.SetHeight(height)
	imageNode := ui.NewDocumentElement("image", style)
	imageNode.Image = image
	if image != nil {
		return imageNode
	}
	style.SetPadding(4)
	style.SetBorderRadius(4)
	style.SetBackground(0xF6F8FA)
	style.SetBorder(1, 0xD8DEE4)
	imageNode.Style = style
	if width <= 48 || height <= 48 {
		return imageNode
	}
	imageNode.Append(ui.NewDocumentText("[image] "+label, styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
		style.SetLineHeight(16)
	})))
	return imageNode
}

func iframeFallbackNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	label := normalizeBlockText(attrValue(node, "title"))
	src := strings.TrimSpace(attrValue(node, "src"))
	if label == "" {
		label = "Embedded frame"
	}
	detail := displayURL(src)
	if detail == "" {
		detail = "iframe host"
	}
	frame := ui.NewDocumentElement("iframe", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(10)
		style.SetBorder(1, 0xD7DEE7)
		style.SetBackground(0xF8FAFC)
		style.SetContain(ui.ContainPaint)
	}))
	applyPageNodeStyles(&frame.Style, node, ctx)
	applyPresentationalNodeAttrs(&frame.Style, node)
	width, widthOK := frame.Style.GetWidth()
	if !widthOK && ctx != nil && ctx.viewportWidth > 0 {
		width = ctx.viewportWidth
	}
	if height, ok := frame.Style.GetHeight(); !ok || height <= 0 {
		if width <= 0 {
			width = 320
		}
		frame.Style.SetHeight((width * 9) / 16)
	}
	frame.Append(
		ui.NewDocumentText("[iframe] "+label, styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(0x202124)
			style.SetFontSize(13)
			style.SetLineHeight(18)
		})),
		ui.NewDocumentText(detail, styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(0x5F6368)
			style.SetFontSize(11)
			style.SetLineHeight(15)
		})),
		ui.NewDocumentText("Embedded content placeholder", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(18, 0, 0, 0)
			style.SetForeground(0x7A8694)
			style.SetFontSize(12)
			style.SetLineHeight(16)
			style.SetTextAlign(ui.TextAlignCenter)
		})),
	)
	return frame
}

func hasAttr(node *Node, name string) bool {
	if node == nil || node.Attrs == nil {
		return false
	}
	_, ok := node.Attrs[name]
	return ok
}

func attrValue(node *Node, name string) string {
	if node == nil || node.Attrs == nil {
		return ""
	}
	return strings.TrimSpace(node.Attrs[name])
}

func attrInt(node *Node, name string, fallback int) int {
	value := attrValue(node, name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func htmlInputType(node *Node) string {
	value := strings.ToLower(attrValue(node, "type"))
	if value == "" {
		return "text"
	}
	return value
}

func htmlButtonType(node *Node) string {
	if node == nil {
		return "button"
	}
	if node.Tag == "button" {
		value := strings.ToLower(attrValue(node, "type"))
		if value == "" {
			if nearestAncestorTag(node, "form") != nil {
				return "submit"
			}
			return "button"
		}
		return value
	}
	switch htmlInputType(node) {
	case "submit":
		return "submit"
	case "reset":
		return "reset"
	default:
		return "button"
	}
}

func htmlControlLabel(node *Node, fallback string) string {
	label := collectNodeText(node, false)
	if label == "" {
		label = attrValue(node, "value")
	}
	if label == "" {
		label = attrValue(node, "aria-label")
	}
	if label == "" {
		label = attrValue(node, "name")
	}
	if label == "" {
		label = fallback
	}
	return normalizeBlockText(label)
}

func htmlOptionValue(node *Node) string {
	value := attrValue(node, "value")
	if value != "" {
		return value
	}
	return htmlControlLabel(node, "Option")
}

func requestRenderedPagePaint(ctx *renderContext) {
	if ctx == nil {
		return
	}
	if ctx.requestPaint != nil {
		ctx.requestPaint()
		return
	}
	if ctx.requestLayout != nil {
		ctx.requestLayout()
	}
}

func requestRenderedPageLayout(ctx *renderContext) {
	if ctx == nil || ctx.requestLayout == nil {
		requestRenderedPagePaint(ctx)
		return
	}
	ctx.requestLayout()
}

func htmlControlStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 6, 0, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(8)
		style.SetBorder(1, 0xC9CFD5)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
	})
}

func neutralHTMLControlStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetContain(ui.ContainPaint)
		style.SetForeground(ui.Black)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetTextAlign(ui.TextAlignCenter)
	})
}

func htmlControlTextStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Black)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	})
}

func htmlControlHintStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(4, 0, 0, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})
}

func htmlControlIndicatorStyle() ui.Style {
	return styled(func(style *ui.Style) {
		applyTagixIconTextStyle(style, 16, 18, ui.Navy)
	})
}

func applyInteractiveControlStyles(node *ui.DocumentNode) {
	if node == nil {
		return
	}
	node.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(0x9AA8B8)
		style.SetBackground(0xF7F9FB)
	})
	node.StyleActive = styled(func(style *ui.Style) {
		style.SetBorderColor(0x7D8FA6)
		style.SetBackground(0xE9EEF4)
	})
	node.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(0x1A73E8)
		style.SetOutline(2, 0x1A73E8)
		style.SetOutlineOffset(1)
	})
}

func applyDisabledControlState(node *ui.DocumentNode) {
	if node == nil {
		return
	}
	node.Focusable = false
	node.Editable = false
	node.Style.SetForeground(ui.Gray)
	node.Style.SetBorderColor(ui.Silver)
	node.Style.SetBackground(ui.White)
	node.Style.SetOpacity(190)
}

func htmlButtonNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	buttonType := htmlButtonType(node)
	fallback := "Button"
	switch buttonType {
	case "submit":
		fallback = "Submit"
	case "reset":
		fallback = "Reset"
	}
	label := htmlControlLabel(node, fallback)
	if label == "" {
		label = fallback
	}
	textStyle := htmlControlTextStyle()
	applyPageTextProperties(&textStyle, node, ctx)
	children := buildInlineNodes(node, ctx, textStyle)
	if len(children) == 0 {
		children = []*ui.DocumentNode{
			ui.NewDocumentText(label, textStyle),
		}
	}
	baseStyle := htmlControlStyle()
	if htmlButtonUseNeutralBase(node, children) {
		baseStyle = neutralHTMLControlStyle()
	}
	button := ui.NewDocumentElement("html-button", baseStyle, children...)
	applyPageNodeStyles(&button.Style, node, ctx)
	applyPresentationalNodeAttrs(&button.Style, node)
	button.Focusable = true
	applyInteractiveControlStyles(button)
	applyPageInteractionStyles(button, node, ctx)
	applyPageInteractionTextStyles(button)
	disabled := hasAttr(node, "disabled")
	if disabled {
		applyDisabledControlState(button)
		return button
	}
	form := ctx.formForControl(node)
	switch buttonType {
	case "submit":
		if form != nil {
			button.OnClick = func() {
				form.submit(node)
			}
		}
	case "reset":
		if form != nil {
			button.OnClick = func() {
				form.reset()
			}
		}
	}
	if form != nil && buttonType == "submit" {
		name := attrValue(node, "name")
		value := attrValue(node, "value")
		if value == "" {
			value = label
		}
		form.addControl(&formControlState{
			node: node,
			fields: func(submitter *Node) []formField {
				if submitter != node || name == "" {
					return nil
				}
				return []formField{{name: name, value: value}}
			},
		})
	}
	return button
}

func htmlButtonUseNeutralBase(node *Node, children []*ui.DocumentNode) bool {
	if node == nil {
		return false
	}
	if strings.TrimSpace(attrValue(node, "id")) != "" ||
		strings.TrimSpace(attrValue(node, "class")) != "" ||
		strings.TrimSpace(attrValue(node, "style")) != "" {
		return true
	}
	if len(children) == 1 && children[0] != nil && children[0].Image != nil {
		return true
	}
	return false
}

func htmlInputNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	switch htmlInputType(node) {
	case "hidden":
		if form := ctx.formForControl(node); form != nil {
			name := attrValue(node, "name")
			value := attrValue(node, "value")
			form.addControl(&formControlState{
				node: node,
				fields: func(_ *Node) []formField {
					if hasAttr(node, "disabled") || name == "" {
						return nil
					}
					return []formField{{name: name, value: value}}
				},
			})
		}
		return nil
	case "submit", "button", "reset":
		return htmlButtonNode(node, ctx)
	case "checkbox":
		return htmlCheckboxNode(node, ctx)
	case "radio":
		return htmlRadioNode(node, ctx)
	case "range":
		return htmlRangeNode(node, ctx)
	default:
		return htmlTextInputNode(node, ctx)
	}
}

func htmlTextInputNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	input := ui.NewDocumentElement("html-input", htmlControlStyle(), nil)
	input.Editable = true
	input.Focusable = true
	input.Value = attrValue(node, "value")
	input.Placeholder = attrValue(node, "placeholder")
	input.Style.SetOverflow(ui.OverflowHidden)
	input.Style.SetWhiteSpace(ui.WhiteSpaceNoWrap)
	if size := attrInt(node, "size", 0); size > 0 {
		width := size * 12
		if width < 120 {
			width = 120
		}
		if width > 520 {
			width = 520
		}
		input.Style.SetWidth(width)
	} else {
		input.Style.SetWidth(220)
	}
	applyPageNodeStyles(&input.Style, node, ctx)
	applyPresentationalNodeAttrs(&input.Style, node)
	input.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Teal)
	})
	input.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Blue)
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})
	applyPageInteractionStyles(input, node, ctx)
	disabled := hasAttr(node, "disabled")
	if disabled {
		applyDisabledControlState(input)
		return input
	}
	form := ctx.formForControl(node)
	if form != nil {
		name := attrValue(node, "name")
		initialValue := attrValue(node, "value")
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *Node) []formField {
				if name == "" {
					return nil
				}
				return []formField{{name: name, value: input.Value}}
			},
			reset: func() bool {
				if input.Value == initialValue {
					return false
				}
				input.Value = initialValue
				return true
			},
		})
		input.OnChange = func(*ui.DocumentNode) {
			form.submit(nil)
		}
	}
	return input
}

func htmlCheckboxNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	checked := hasAttr(node, "checked")
	initialChecked := checked
	indicator := ui.NewDocumentText(htmlCheckboxGlyph(checked), htmlControlIndicatorStyle())
	label := htmlControlLabel(node, "Checkbox")
	control := ui.NewDocumentElement("html-checkbox", htmlControlStyle(),
		indicator,
		ui.NewDocumentText(" "+label, htmlControlTextStyle()),
	)
	applyPageNodeStyles(&control.Style, node, ctx)
	applyPresentationalNodeAttrs(&control.Style, node)
	control.Focusable = true
	applyInteractiveControlStyles(control)
	applyPageInteractionStyles(control, node, ctx)
	applyPageInteractionTextStyles(control)
	disabled := hasAttr(node, "disabled")
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		value := attrValue(node, "value")
		if value == "" {
			value = "on"
		}
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *Node) []formField {
				if disabled || name == "" || !checked {
					return nil
				}
				return []formField{{name: name, value: value}}
			},
			reset: func() bool {
				if checked == initialChecked {
					return false
				}
				checked = initialChecked
				indicator.Text = htmlCheckboxGlyph(checked)
				return true
			},
		})
	}
	if disabled {
		applyDisabledControlState(control)
		return control
	}
	toggle := func() {
		checked = !checked
		indicator.Text = htmlCheckboxGlyph(checked)
		requestRenderedPageLayout(ctx)
	}
	control.OnClick = func() {
		toggle()
	}
	control.OnKeyDown = func(_ *ui.DocumentNode, event *ui.DocumentEvent) {
		if event == nil {
			return
		}
		if event.Key.Code == 13 || event.Key.Code == 32 {
			toggle()
			event.PreventDefault()
			event.StopPropagation()
		}
	}
	return control
}

func htmlRadioNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	checked := hasAttr(node, "checked")
	initialChecked := checked
	indicator := ui.NewDocumentText(htmlRadioGlyph(checked), htmlControlIndicatorStyle())
	label := htmlControlLabel(node, "Radio")
	control := ui.NewDocumentElement("html-radio", htmlControlStyle(),
		indicator,
		ui.NewDocumentText(" "+label, htmlControlTextStyle()),
	)
	applyPageNodeStyles(&control.Style, node, ctx)
	applyPresentationalNodeAttrs(&control.Style, node)
	control.Focusable = true
	applyInteractiveControlStyles(control)
	applyPageInteractionStyles(control, node, ctx)
	applyPageInteractionTextStyles(control)
	if hasAttr(node, "disabled") {
		applyDisabledControlState(control)
		return control
	}
	group := attrValue(node, "name")
	if group == "" {
		group = "radio:" + strconv.Itoa(node.ID)
	}
	state := &radioControlState{
		node:      control,
		indicator: indicator,
		checked:   checked,
	}
	if ctx != nil {
		ctx.radioGroups[group] = append(ctx.radioGroups[group], state)
	}
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		value := attrValue(node, "value")
		if value == "" {
			value = "on"
		}
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *Node) []formField {
				if name == "" || !state.checked {
					return nil
				}
				return []formField{{name: name, value: value}}
			},
			reset: func() bool {
				if state.checked == initialChecked {
					return false
				}
				state.checked = initialChecked
				state.indicator.Text = htmlRadioGlyph(state.checked)
				return true
			},
		})
	}
	selectRadio := func() {
		changed := false
		if ctx != nil {
			for _, candidate := range ctx.radioGroups[group] {
				if candidate == nil {
					continue
				}
				if candidate == state {
					if !candidate.checked {
						candidate.checked = true
						candidate.indicator.Text = htmlRadioGlyph(true)
						changed = true
					}
					continue
				}
				if candidate.checked {
					candidate.checked = false
					candidate.indicator.Text = htmlRadioGlyph(false)
					changed = true
				}
			}
		} else if !state.checked {
			state.checked = true
			state.indicator.Text = htmlRadioGlyph(true)
			changed = true
		}
		if changed {
			requestRenderedPageLayout(ctx)
		}
	}
	control.OnClick = func() {
		selectRadio()
	}
	control.OnKeyDown = func(_ *ui.DocumentNode, event *ui.DocumentEvent) {
		if event == nil {
			return
		}
		if event.Key.Code == 13 || event.Key.Code == 32 {
			selectRadio()
			event.PreventDefault()
			event.StopPropagation()
		}
	}
	return control
}

func htmlRangeNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	minValue := attrInt(node, "min", 0)
	maxValue := attrInt(node, "max", 100)
	if maxValue <= minValue {
		maxValue = minValue + 100
	}
	stepValue := attrInt(node, "step", 1)
	if stepValue <= 0 {
		stepValue = 1
	}
	value := attrInt(node, "value", minValue)
	if value < minValue {
		value = minValue
	}
	if value > maxValue {
		value = maxValue
	}
	initialValue := value
	label := htmlControlLabel(node, "Range")
	valueText := ui.NewDocumentText("", htmlControlTextStyle())
	hintText := ui.NewDocumentText("", htmlControlHintStyle())
	update := func() {
		valueText.Text = label
		hintText.Text = "Value " + strconv.Itoa(value) + " of " + strconv.Itoa(maxValue)
	}
	update()
	control := ui.NewDocumentElement("html-range", htmlControlStyle(), valueText, hintText)
	control.Style.SetWidth(220)
	applyPageNodeStyles(&control.Style, node, ctx)
	applyPresentationalNodeAttrs(&control.Style, node)
	control.Focusable = true
	applyInteractiveControlStyles(control)
	applyPageInteractionStyles(control, node, ctx)
	applyPageInteractionTextStyles(control)
	disabled := hasAttr(node, "disabled")
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *Node) []formField {
				if disabled || name == "" {
					return nil
				}
				return []formField{{name: name, value: strconv.Itoa(value)}}
			},
			reset: func() bool {
				if value == initialValue {
					return false
				}
				value = initialValue
				update()
				return true
			},
		})
	}
	if disabled {
		applyDisabledControlState(control)
		return control
	}
	setValue := func(next int) {
		if next < minValue {
			next = minValue
		}
		if next > maxValue {
			next = maxValue
		}
		if next == value {
			return
		}
		value = next
		update()
		requestRenderedPageLayout(ctx)
	}
	control.OnClick = func() {
		next := value + stepValue
		if next > maxValue {
			next = minValue
		}
		setValue(next)
	}
	control.OnKeyDown = func(_ *ui.DocumentNode, event *ui.DocumentEvent) {
		if event == nil {
			return
		}
		switch {
		case event.Key.ScanCode == 0x4B:
			setValue(value - stepValue)
		case event.Key.ScanCode == 0x4D:
			setValue(value + stepValue)
		case event.Key.ScanCode == 0x47:
			setValue(minValue)
		case event.Key.ScanCode == 0x4F:
			setValue(maxValue)
		default:
			return
		}
		event.PreventDefault()
		event.StopPropagation()
	}
	return control
}

func htmlTextareaNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	text := collectNodeTextPreserve(node, false)
	initialText := text
	if text == "" {
		text = attrValue(node, "placeholder")
	}
	if text == "" {
		text = "Textarea"
	}
	area := ui.NewDocumentElement("html-textarea", htmlControlStyle(),
		ui.NewDocumentText(text, styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(ui.Black)
			style.SetFontPath(webMonoFontPath)
			style.SetFontSize(12)
			style.SetLineHeight(17)
			style.SetWhiteSpace(ui.WhiteSpacePreWrap)
		})),
	)
	applyPageNodeStyles(&area.Style, node, ctx)
	applyPresentationalNodeAttrs(&area.Style, node)
	applyPageInteractionStyles(area, node, ctx)
	applyPageInteractionTextStyles(area)
	if cols := attrInt(node, "cols", 0); cols > 0 {
		width := cols * 10
		if width < 180 {
			width = 180
		}
		if width > 520 {
			width = 520
		}
		area.Style.SetWidth(width)
	} else {
		area.Style.SetWidth(320)
	}
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *Node) []formField {
				if hasAttr(node, "disabled") || name == "" {
					return nil
				}
				return []formField{{name: name, value: initialText}}
			},
		})
	}
	if rows := attrInt(node, "rows", 0); rows > 0 {
		height := rows*18 + 18
		if height < 56 {
			height = 56
		}
		area.Style.SetHeight(height)
	}
	return area
}

func htmlSelectNode(node *Node, ctx *renderContext) *ui.DocumentNode {
	type optionState struct {
		label string
		value string
	}
	options := make([]optionState, 0, len(node.Children))
	selected := 0
	for _, child := range node.Children {
		if child == nil || child.Type != ElementNode || child.Tag != "option" {
			continue
		}
		label := htmlControlLabel(child, attrValue(child, "value"))
		if label == "" {
			label = "Option"
		}
		options = append(options, optionState{
			label: label,
			value: htmlOptionValue(child),
		})
		if hasAttr(child, "selected") {
			selected = len(options) - 1
		}
	}
	if len(options) == 0 {
		options = append(options, optionState{label: "Select option", value: "Select option"})
	}
	if selected < 0 || selected >= len(options) {
		selected = 0
	}
	initialSelected := selected
	valueText := ui.NewDocumentText(options[selected].label, htmlControlTextStyle())
	control := ui.NewDocumentElement("html-select", htmlControlStyle(), valueText)
	control.Style.SetMinWidth(140)
	applyPageNodeStyles(&control.Style, node, ctx)
	applyPresentationalNodeAttrs(&control.Style, node)
	control.Focusable = true
	applyInteractiveControlStyles(control)
	applyPageInteractionStyles(control, node, ctx)
	applyPageInteractionTextStyles(control)
	disabled := hasAttr(node, "disabled")
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *Node) []formField {
				if disabled || name == "" || selected < 0 || selected >= len(options) {
					return nil
				}
				return []formField{{name: name, value: options[selected].value}}
			},
			reset: func() bool {
				if selected == initialSelected {
					return false
				}
				selected = initialSelected
				valueText.Text = options[selected].label
				return true
			},
		})
	}
	if disabled {
		applyDisabledControlState(control)
		return control
	}
	cycle := func(step int) {
		if len(options) == 0 {
			return
		}
		selected += step
		for selected < 0 {
			selected += len(options)
		}
		for selected >= len(options) {
			selected -= len(options)
		}
		valueText.Text = options[selected].label
		requestRenderedPageLayout(ctx)
	}
	control.OnClick = func() {
		cycle(1)
	}
	control.OnKeyDown = func(_ *ui.DocumentNode, event *ui.DocumentEvent) {
		if event == nil {
			return
		}
		switch {
		case event.Key.ScanCode == 0x48 || event.Key.ScanCode == 0x4B:
			cycle(-1)
		case event.Key.ScanCode == 0x50 || event.Key.ScanCode == 0x4D || event.Key.Code == 32 || event.Key.Code == 13:
			cycle(1)
		default:
			return
		}
		event.PreventDefault()
		event.StopPropagation()
	}
	return control
}

func htmlProgressNode(node *Node) *ui.DocumentNode {
	maxValue := attrInt(node, "max", 1)
	if maxValue <= 0 {
		maxValue = 1
	}
	value := attrInt(node, "value", 0)
	if value < 0 {
		value = 0
	}
	if value > maxValue {
		value = maxValue
	}
	percent := (value * 100) / maxValue
	control := ui.NewDocumentElement("html-progress", htmlControlStyle(),
		ui.NewDocumentText("Progress", htmlControlTextStyle()),
		ui.NewDocumentText(strconv.Itoa(percent)+"% complete", htmlControlHintStyle()),
	)
	control.Style.SetWidth(180)
	return control
}

func buildMessageDocument(title string, detail string) *ui.DocumentNode {
	return ui.NewDocumentElement("message-page", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(18, 16)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
	}), ui.NewDocumentText(title, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(0x202124)
		style.SetFontSize(20)
		style.SetLineHeight(24)
		style.SetMargin(0, 0, 8, 0)
	})), ui.NewDocumentText(detail, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(0x3C4043)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	})))
}

func messageCard(title string, detail string) *ui.DocumentNode {
	return ui.NewDocumentElement("message", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12)
		style.SetBorderRadius(10)
		style.SetBorder(1, ui.Silver)
		style.SetBackground(ui.White)
	}), ui.NewDocumentText(title, styled(func(style *ui.Style) {
		style.SetForeground(ui.Navy)
		style.SetFontSize(16)
		style.SetMargin(0, 0, 4, 0)
	})), ui.NewDocumentText(detail, styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
		style.SetLineHeight(17)
	})))
}

func documentLinkNodeFromAnchor(node *Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	baseURL := ""
	if ctx != nil {
		baseURL = ctx.baseURL
	}
	href := resolveURL(baseURL, node.Attrs["href"])
	if href == "" {
		return nil
	}
	label := collectNodeText(node, false)
	if label == "" {
		label = displayURL(href)
	}
	return documentLinkCard(label, href, ctx)
}

func documentLinkCard(label string, href string, ctx *renderContext) *ui.DocumentNode {
	title := label
	if title == "" {
		title = displayURL(href)
	}
	card := ui.NewDocumentElement("link", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetPadding(0)
		style.SetBorder(0, ui.White)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(title, styled(func(style *ui.Style) {
		style.SetForeground(ui.Blue)
		style.SetFontSize(13)
		style.SetTextDecoration(ui.TextDecorationUnderline)
		style.SetMargin(0, 0, 2, 0)
	})), ui.NewDocumentText(href, styled(func(style *ui.Style) {
		style.SetForeground(0x5F6368)
		style.SetFontSize(11)
	})))
	card.Focusable = true
	card.StyleHover = styled(func(style *ui.Style) {
		style.SetForeground(ui.Teal)
	})
	card.StyleActive = styled(func(style *ui.Style) {
		style.SetForeground(ui.Navy)
	})
	card.StyleFocus = styled(func(style *ui.Style) {
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})
	if ctx != nil && ctx.openURL != nil && href != "" {
		card.OnClick = func() {
			ctx.openURL(href)
		}
	}
	bindLinkStatusHint(card, href, ctx)
	return card
}

func bindLinkStatusHint(target *ui.DocumentNode, href string, ctx *renderContext) {
	if target == nil || ctx == nil || ctx.setStatusHint == nil {
		return
	}
	hint := displayURL(href)
	if hint == "" {
		return
	}
	target.OnMouseEnter = func() {
		ctx.setStatusHint(hint)
	}
	target.OnMouseLeave = func() {
		ctx.setStatusHint("")
	}
}

func normalizeBlockText(value string) string {
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func displayURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimSuffix(value, "/")
	return value
}
