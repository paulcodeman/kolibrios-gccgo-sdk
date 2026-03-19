package main

import (
	"dom"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"ui"
)

const (
	webSansFontPath = "assets/OpenSans-Regular.ttf"
	webMonoFontPath = "assets/RobotoMono-Regular.ttf"
	webShellHTML    = "assets/shell.html"
)

var (
	cachedShellTemplate     string
	cachedShellTemplateRead bool
)

type renderContext struct {
	baseURL       string
	openURL       func(string)
	submitForm    func(string, string, neturl.Values)
	requestPaint  func()
	requestLayout func()
	radioGroups   map[string][]*radioControlState
	forms         map[*dom.Node]*formState
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
	node   *dom.Node
	fields func(*dom.Node) []formField
	reset  func() bool
}

type formState struct {
	node     *dom.Node
	ctx      *renderContext
	controls []*formControlState
}

func nearestAncestorTag(node *dom.Node, tag string) *dom.Node {
	for current := node; current != nil; current = current.Parent {
		if current.Type == dom.ElementNode && current.Tag == tag {
			return current
		}
	}
	return nil
}

func (ctx *renderContext) formForControl(node *dom.Node) *formState {
	if ctx == nil || node == nil {
		return nil
	}
	formNode := nearestAncestorTag(node, "form")
	if formNode == nil {
		return nil
	}
	if ctx.forms == nil {
		ctx.forms = map[*dom.Node]*formState{}
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

func (form *formState) submit(submitter *dom.Node) {
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

func resolveFormAction(baseURL string, formNode *dom.Node) string {
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
	root := ui.NewDocumentElement("browser-shell", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(0)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
		style.SetContain(ui.ContainPaint)
	}))

	actions := ui.NewDocumentElement("browser-shell-actions", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 8, 0, 0)
	}),
		shellButtonNode(app, "Back", "back", canBack),
		shellButtonNode(app, "Forward", "forward", canForward),
		shellButtonNode(app, "Reload", "reload", true),
		shellButtonNode(app, "Home", "home", true),
	)

	address := shellAddressNode(app, currentURL)
	address.Style.SetMargin(0, 0, 0, 0)
	root.Append(actions)
	root.Append(address)
	return root
}

func renderShellRoot(app *App) *ui.DocumentNode {
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
	doc := dom.Parse(shellTemplateHTML(currentURL, canBack, canForward))
	if doc != nil && doc.Root != nil {
		if root := buildShellTemplateRoot(app, doc.Root); root != nil {
			return root
		}
	}
	return buildShellDocument(app)
}

func shellTemplateHTML(currentURL string, canBack bool, canForward bool) string {
	if strings.TrimSpace(currentURL) == "" {
		currentURL = defaultURL
	}
	replacer := strings.NewReplacer(
		"<<current_url>>", escapeHTMLText(currentURL),
		"<<back_enabled>>", boolAttr(canBack),
		"<<forward_enabled>>", boolAttr(canForward),
	)
	return replacer.Replace(loadShellTemplateSource())
}

func loadShellTemplateSource() string {
	if cachedShellTemplateRead {
		return cachedShellTemplate
	}
	cachedShellTemplateRead = true
	data, err := os.ReadFile(webShellHTML)
	if err != nil || len(data) == 0 {
		cachedShellTemplate = defaultShellTemplateHTML
		return cachedShellTemplate
	}
	cachedShellTemplate = string(data)
	return cachedShellTemplate
}

func buildShellTemplateRoot(app *App, node *dom.Node) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	if built := buildShellTemplateNode(app, node); built != nil {
		return built
	}
	for _, child := range node.Children {
		if built := buildShellTemplateRoot(app, child); built != nil {
			return built
		}
	}
	return nil
}

func buildShellTemplateNode(app *App, node *dom.Node) *ui.DocumentNode {
	if node == nil || node.Type != dom.ElementNode {
		return nil
	}
	role := strings.TrimSpace(node.Attrs["data-role"])
	switch role {
	case "shell-root":
		return ui.NewDocumentElement("browser-shell", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetPadding(0)
			style.SetFontPath(webSansFontPath)
			style.SetFontSize(13)
			style.SetLineHeight(18)
			style.SetForeground(ui.Black)
			style.SetContain(ui.ContainPaint)
		}), buildShellChildren(app, node)...)
	case "meta", "hero", "title", "status":
		return nil
	case "nav-row", "toolbar", "actions":
		return ui.NewDocumentElement("browser-shell-actions", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayInlineBlock)
			style.SetMargin(0, 8, 0, 0)
		}), buildShellChildren(app, node)...)
	case "button":
		return shellButtonNode(app, collectNodeText(node, false), strings.TrimSpace(node.Attrs["data-action"]), attrIsTrue(node.Attrs["data-enabled"]))
	case "address":
		return shellAddressNode(app, strings.TrimSpace(node.Attrs["value"]))
	case "hint":
		return nil
	default:
		children := buildShellChildren(app, node)
		if len(children) == 1 {
			return children[0]
		}
		if len(children) == 0 {
			return nil
		}
		return ui.NewDocumentElement(role, styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
		}), children...)
	}
}

func buildShellChildren(app *App, node *dom.Node) []*ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := make([]*ui.DocumentNode, 0, len(node.Children))
	for _, child := range node.Children {
		built := buildShellTemplateNode(app, child)
		if built != nil {
			children = append(children, built)
		}
	}
	return children
}

func shellButtonNode(app *App, label string, action string, enabled bool) *ui.DocumentNode {
	label = normalizeBlockText(label)
	if label == "" {
		label = "Action"
	}
	button := ui.NewDocumentElement("shell-button-"+action, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 6, 0, 0)
		style.SetPadding(5, 10)
		style.SetBorderRadius(8)
		style.SetBorder(1, 0xC3CAD2)
		style.SetBackground(0xE5E9EE)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(label, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(0x202124)
		style.SetFontSize(12)
		style.SetLineHeight(16)
	})))
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
	switch action {
	case "back":
		if app != nil {
			app.shellBackNode = button
		}
		button.OnClick = func() {
			if app != nil {
				app.goBack()
			}
		}
	case "forward":
		if app != nil {
			app.shellForwardNode = button
		}
		button.OnClick = func() {
			if app != nil {
				app.goForward()
			}
		}
	case "reload":
		if app != nil {
			app.shellReloadNode = button
		}
		button.OnClick = func() {
			if app != nil {
				app.reloadCurrent()
			}
		}
	case "home":
		if app != nil {
			app.shellHomeNode = button
		}
		button.OnClick = func() {
			if app != nil {
				app.goHome()
			}
		}
	}
	applyShellButtonState(button, enabled)
	return button
}

func applyShellButtonState(node *ui.DocumentNode, enabled bool) {
	if node == nil {
		return
	}
	node.Focusable = enabled
	if enabled {
		node.Style.SetBackground(0xE5E9EE)
		node.Style.SetBorderColor(0xC3CAD2)
		node.Style.SetForeground(0x202124)
		node.Style.SetOpacity(255)
	} else {
		node.Style.SetBackground(0xEEF1F4)
		node.Style.SetBorderColor(0xD8DDE3)
		node.Style.SetForeground(0x8B9299)
		node.Style.SetOpacity(220)
	}
	if len(node.Children) > 0 && node.Children[0] != nil {
		if enabled {
			node.Children[0].Style.SetForeground(0x202124)
			node.Children[0].Style.SetOpacity(255)
		} else {
			node.Children[0].Style.SetForeground(0x8B9299)
			node.Children[0].Style.SetOpacity(220)
		}
	}
}

func shellAddressNode(app *App, value string) *ui.DocumentNode {
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
	if app != nil {
		app.shellAddressNode = input
	}
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
	return input
}

func syncShellDocument(app *App, title string, status string) {
	if app == nil || app.shellDocument == nil {
		return
	}
	layoutDirty := false
	if app.shellTitleNode != nil && app.shellTitleNode.Text != title {
		app.shellTitleNode.Text = title
		layoutDirty = true
	}
	if app.shellStatusNode != nil && app.shellStatusNode.Text != status {
		app.shellStatusNode.Text = status
		layoutDirty = true
	}
	applyShellButtonState(app.shellBackNode, app.historyIndex > 0)
	applyShellButtonState(app.shellForwardNode, app.historyIndex+1 < len(app.history))
	applyShellButtonState(app.shellReloadNode, true)
	applyShellButtonState(app.shellHomeNode, true)
	address := strings.TrimSpace(app.addressText)
	if address == "" {
		address = defaultURL
	}
	if app.shellAddressNode != nil && app.shellAddressNode.Value != address {
		app.shellAddressNode.Value = address
		app.shellAddressNode.Placeholder = "Type URL here"
		app.shellDocument.MarkDirty()
		layoutDirty = false
	}
	if layoutDirty {
		app.shellDocument.MarkLayoutDirty()
	} else {
		app.shellDocument.MarkDirty()
	}
}

func attrIsTrue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func boolAttr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

const defaultShellTemplateHTML = `<body>
<section data-role="shell-root">
<div data-role="actions">
<button data-role="button" data-action="back" data-enabled="<<back_enabled>>">Back</button>
<button data-role="button" data-action="forward" data-enabled="<<forward_enabled>>">Forward</button>
<button data-role="button" data-action="reload" data-enabled="true">Reload</button>
<button data-role="button" data-action="home" data-enabled="true">Home</button>
</div>
<input data-role="address" value="<<current_url>>">
</section>
</body>`

func escapeHTMLText(value string) string {
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return replacer.Replace(value)
}

func buildRenderedDocument(title string, currentURL string, doc *dom.Document, openURL func(string), submitForm func(string, string, neturl.Values), requestLayout func(), requestPaint func()) *ui.DocumentNode {
	ctx := &renderContext{
		baseURL:       currentURL,
		openURL:       openURL,
		submitForm:    submitForm,
		requestLayout: requestLayout,
		requestPaint:  requestPaint,
		radioGroups:   map[string][]*radioControlState{},
		forms:         map[*dom.Node]*formState{},
	}
	children := make([]*ui.DocumentNode, 0, 24)

	contentNodes := buildDocumentNodes(doc, ctx)
	if len(contentNodes) == 0 {
		children = append(children, messageCard("No renderable content", "The HTML5 parser returned a tree, but the current browser-host adapter did not find readable nodes yet."))
	} else {
		content := ui.NewDocumentElement("content", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0)
			style.SetContain(ui.ContainPaint)
		}), contentNodes...)
		children = append(children, content)
	}

	return ui.NewDocumentElement("page", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(16)
		style.SetContain(ui.ContainPaint)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
	}), children...)
}

func buildDocumentNodes(doc *dom.Document, ctx *renderContext) []*ui.DocumentNode {
	if doc == nil || doc.Root == nil {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, 16)
	appendDocumentNodes(&nodes, doc.Root, ctx)
	return nodes
}

func buildFlowNodes(node *dom.Node, ctx *renderContext) []*ui.DocumentNode {
	if node == nil {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, len(node.Children))
	appendFlowContentNodes(&nodes, node, ctx)
	return nodes
}

func appendFlowContentNodes(out *[]*ui.DocumentNode, node *dom.Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	builder := inlinePieceBuilder{}
	flushParagraph := func() {
		if paragraph := flowParagraphNode(builder.pieces, ctx); paragraph != nil {
			*out = append(*out, paragraph)
		}
		builder = inlinePieceBuilder{}
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		switch child.Type {
		case dom.CommentNode:
			continue
		case dom.TextNode:
			builder.appendText(child.Text)
		case dom.DocumentNode:
			flushParagraph()
			appendFlowContentNodes(out, child, ctx)
		case dom.ElementNode:
			if isSkippableElementTag(child.Tag) {
				continue
			}
			if nodeParticipatesInInlineFlow(child) {
				collectInlinePieces(&builder, child, ctx)
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
	case "script", "style", "head", "title", "meta", "link", "option":
		return true
	default:
		return false
	}
}

func nodeParticipatesInInlineFlow(node *dom.Node) bool {
	if node == nil {
		return false
	}
	switch node.Type {
	case dom.TextNode:
		return normalizeBlockText(node.Text) != ""
	case dom.CommentNode, dom.DocumentNode:
		return false
	case dom.ElementNode:
	default:
		return false
	}
	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link", "option":
		return false
	case "br", "a", "code", "img", "span", "strong", "em", "b", "i", "u", "small", "big", "abbr", "cite", "q", "s", "sub", "sup", "mark", "time", "kbd", "samp", "var", "wbr", "label", "button", "input", "textarea", "select", "progress":
		return true
	case "html", "body", "main", "section", "article", "aside", "nav", "header", "footer", "div", "form", "fieldset", "legend", "figure", "figcaption", "details", "summary", "dl", "dt", "dd", "table", "caption", "tbody", "thead", "tfoot", "tr", "td", "th", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "p", "blockquote", "pre", "ul", "ol", "li":
		return false
	default:
		return true
	}
}

func appendDocumentNodes(out *[]*ui.DocumentNode, node *dom.Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}

	switch node.Type {
	case dom.CommentNode:
		return
	case dom.TextNode:
		text := normalizeBlockText(node.Text)
		if text != "" {
			*out = append(*out, paragraphNode(text))
		}
		return
	case dom.DocumentNode:
		appendFlowContentNodes(out, node, ctx)
		return
	case dom.ElementNode:
	default:
		return
	}

	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link", "option":
		return
	case "html", "body", "main", "section", "article", "aside", "nav", "header", "footer", "div", "form", "label", "dl", "tbody", "thead", "tfoot", "tr", "td", "th":
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
	case "pre", "code":
		text := collectNodeTextPreserve(node, true)
		if text != "" {
			*out = append(*out, preformattedNode(text))
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
	case "option":
		return
	case "img":
		if image := imageFallbackNode(node); image != nil {
			*out = append(*out, image)
		}
		return
	default:
		appendFlowContentNodes(out, node, ctx)
	}
}

func appendListNodes(out *[]*ui.DocumentNode, node *dom.Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	ordered := node.Tag == "ol"
	index := attrInt(node, "start", 1)
	if index <= 0 {
		index = 1
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		if child.Type == dom.ElementNode && child.Tag == "li" {
			marker := "-"
			if ordered {
				marker = strconv.Itoa(index) + "."
				index++
			}
			if item := listItemBlockNode(child, ctx, marker); item != nil {
				*out = append(*out, item)
			}
			continue
		}
		appendDocumentNodes(out, child, ctx)
	}
}

func appendNestedDocumentLinks(out *[]*ui.DocumentNode, node *dom.Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	for _, child := range node.Children {
		appendDirectAnchorNodes(out, child, ctx)
	}
}

func appendDirectAnchorNodes(out *[]*ui.DocumentNode, node *dom.Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	if node.Type == dom.ElementNode && node.Tag == "a" {
		if link := documentLinkNodeFromAnchor(node, ctx); link != nil {
			*out = append(*out, link)
		}
		return
	}
	for _, child := range node.Children {
		appendDirectAnchorNodes(out, child, ctx)
	}
}

func collectNodeText(node *dom.Node, skipLinks bool) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	collectNodeTextInto(&builder, node, skipLinks, false)
	return normalizeBlockText(builder.String())
}

func collectNodeTextPreserve(node *dom.Node, skipLinks bool) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	collectNodeTextInto(&builder, node, skipLinks, true)
	return strings.TrimSpace(builder.String())
}

func collectNodeTextInto(builder *strings.Builder, node *dom.Node, skipLinks bool, preserve bool) {
	if builder == nil || node == nil {
		return
	}
	switch node.Type {
	case dom.CommentNode:
		return
	case dom.TextNode:
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
	case dom.ElementNode:
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

func headingNode(tag string, text string) *ui.DocumentNode {
	size := 14
	marginBottom := 6
	switch tag {
	case "h1":
		size = 22
		marginBottom = 8
	case "h2":
		size = 18
		marginBottom = 8
	case "h3":
		size = 16
	}
	return ui.NewDocumentText(text, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 4, marginBottom, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(size)
		style.SetLineHeight(size + 4)
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
	kind inlinePieceKind
	text string
	href string
	node *dom.Node
}

type inlinePieceBuilder struct {
	pieces    []inlinePiece
	needSpace bool
}

func paragraphInlineStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	})
}

func paragraphBlockStyle() ui.Style {
	return styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	})
}

func paragraphNode(text string) *ui.DocumentNode {
	return ui.NewDocumentText(text, paragraphBlockStyle())
}

func flowParagraphNode(pieces []inlinePiece, ctx *renderContext) *ui.DocumentNode {
	children := inlineNodesFromPieces(pieces, paragraphInlineStyle(), ctx)
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("flow-paragraph", paragraphBlockStyle(), children...)
}

func headingBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	size := 14
	marginBottom := 6
	switch node.Tag {
	case "h1":
		size = 22
		marginBottom = 8
	case "h2":
		size = 18
		marginBottom = 8
	case "h3":
		size = 16
	}
	children := buildInlineNodes(node, ctx, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Navy)
		style.SetFontSize(size)
		style.SetLineHeight(size + 4)
	}))
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("heading-"+node.Tag, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 4, marginBottom, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(size)
		style.SetLineHeight(size + 4)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func paragraphBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, paragraphInlineStyle())
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("paragraph", paragraphBlockStyle(), children...)
}

func blockquoteBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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

func legendBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, semanticLabelInlineStyle())
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("legend", semanticLabelBlockStyle(6), children...)
}

func figureCaptionBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
	return ui.NewDocumentElement("figcaption", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(4, 0, 8, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
		style.SetLineHeight(17)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func summaryBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, semanticLabelInlineStyle())
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("summary", semanticLabelBlockStyle(6), children...)
}

func definitionTermBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, ctx, semanticLabelInlineStyle())
	if len(children) == 0 {
		return nil
	}
	return ui.NewDocumentElement("definition-term", semanticLabelBlockStyle(2), children...)
}

func definitionDetailBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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

func fieldsetBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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

func figureBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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

func detailsBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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

func listItemBlockNode(node *dom.Node, ctx *renderContext, marker string) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildFlowNodes(node, ctx)
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

func standaloneLinkNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	link := inlineLinkNodeFromAnchor(node, ctx)
	if link == nil {
		return nil
	}
	return ui.NewDocumentElement("standalone-link", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	}), link)
}

func buildInlineNodes(node *dom.Node, ctx *renderContext, baseStyle ui.Style) []*ui.DocumentNode {
	builder := inlinePieceBuilder{}
	for _, child := range node.Children {
		collectInlinePieces(&builder, child, ctx)
	}
	return inlineNodesFromPieces(builder.pieces, baseStyle, ctx)
}

func collectInlinePieces(builder *inlinePieceBuilder, node *dom.Node, ctx *renderContext) {
	if builder == nil || node == nil {
		return
	}
	switch node.Type {
	case dom.CommentNode:
		return
	case dom.TextNode:
		builder.appendText(node.Text)
		return
	case dom.DocumentNode:
		for _, child := range node.Children {
			collectInlinePieces(builder, child, ctx)
		}
		return
	case dom.ElementNode:
	default:
		return
	}

	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link":
		return
	case "br":
		builder.appendBreak()
		return
	case "a":
		baseURL := ""
		if ctx != nil {
			baseURL = ctx.baseURL
		}
		builder.appendLink(collectNodeText(node, false), resolveURL(baseURL, node.Attrs["href"]))
		return
	case "code":
		builder.appendCode(collectNodeTextPreserve(node, false))
		return
	case "img":
		label := normalizeBlockText(node.Attrs["alt"])
		if label == "" {
			label = displayURL(strings.TrimSpace(node.Attrs["src"]))
		}
		builder.appendImage(label)
		return
	case "button", "textarea", "select", "progress":
		builder.appendControl(node)
		return
	case "input":
		if htmlInputType(node) == "hidden" {
			return
		}
		builder.appendControl(node)
		return
	default:
		for _, child := range node.Children {
			collectInlinePieces(builder, child, ctx)
		}
	}
}

func (builder *inlinePieceBuilder) appendText(raw string) {
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
			builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " "})
			builder.needSpace = false
		}
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: word})
	}
	if len(raw) > 0 && isSpaceByte(raw[len(raw)-1]) {
		builder.needSpace = true
	}
}

func (builder *inlinePieceBuilder) appendLink(label string, href string) {
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
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " "})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceLink, text: label, href: href})
}

func (builder *inlinePieceBuilder) appendCode(text string) {
	if builder == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " "})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceCode, text: text})
}

func (builder *inlinePieceBuilder) appendImage(label string) {
	if builder == nil {
		return
	}
	label = normalizeBlockText(label)
	if label == "" {
		return
	}
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " "})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceImage, text: label})
}

func (builder *inlinePieceBuilder) appendBreak() {
	if builder == nil {
		return
	}
	builder.needSpace = false
	builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceBreak})
}

func (builder *inlinePieceBuilder) appendControl(node *dom.Node) {
	if builder == nil || node == nil {
		return
	}
	if builder.needSpace && len(builder.pieces) > 0 {
		builder.pieces = append(builder.pieces, inlinePiece{kind: inlinePieceText, text: " "})
		builder.needSpace = false
	}
	builder.pieces = append(builder.pieces, inlinePiece{
		kind: inlinePieceControl,
		node: node,
	})
}

func inlineNodesFromPieces(pieces []inlinePiece, baseStyle ui.Style, ctx *renderContext) []*ui.DocumentNode {
	if len(pieces) == 0 {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, len(pieces))
	for _, piece := range pieces {
		switch piece.kind {
		case inlinePieceText:
			nodes = append(nodes, inlineTextNode(piece.text, baseStyle))
		case inlinePieceLink:
			if link := inlineLinkNode(piece.text, piece.href, baseStyle, ctx); link != nil {
				nodes = append(nodes, link)
			}
		case inlinePieceCode:
			if code := inlineCodeNode(piece.text, baseStyle); code != nil {
				nodes = append(nodes, code)
			}
		case inlinePieceImage:
			if image := inlineImageNode(piece.text, baseStyle); image != nil {
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

func inlineControlNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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

func inlineLinkNodeFromAnchor(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
	return inlineLinkNode(label, href, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}), ctx)
}

func inlineLinkNode(label string, href string, baseStyle ui.Style, ctx *renderContext) *ui.DocumentNode {
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
	link := ui.NewDocumentElement("inline-link", style, inlineTextTokens(label, styled(func(textStyle *ui.Style) {
		*textStyle = baseStyle
		textStyle.SetDisplay(ui.DisplayInline)
	}))...)
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
	if ctx != nil && ctx.openURL != nil && href != "" {
		link.OnClick = func() {
			ctx.openURL(href)
		}
	}
	return link
}

func inlineCodeNode(text string, baseStyle ui.Style) *ui.DocumentNode {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	style := baseStyle
	style.SetDisplay(ui.DisplayInlineBlock)
	style.SetPadding(1, 4)
	style.SetMargin(0, 1)
	style.SetBorderRadius(6)
	style.SetBackground(ui.Silver)
	style.SetForeground(ui.Navy)
	style.SetFontPath(webMonoFontPath)
	style.SetFontSize(12)
	style.SetLineHeight(16)
	childStyle := style
	childStyle.SetDisplay(ui.DisplayInline)
	return ui.NewDocumentElement("inline-code", style, ui.NewDocumentText(text, childStyle))
}

func inlineImageNode(label string, baseStyle ui.Style) *ui.DocumentNode {
	label = normalizeBlockText(label)
	if label == "" {
		return nil
	}
	style := baseStyle
	style.SetDisplay(ui.DisplayInlineBlock)
	style.SetPadding(1, 4)
	style.SetMargin(0, 1)
	style.SetBorderRadius(6)
	style.SetBackground(ui.Silver)
	style.SetForeground(ui.Gray)
	childStyle := style
	childStyle.SetDisplay(ui.DisplayInline)
	return ui.NewDocumentElement("inline-image", style, ui.NewDocumentText("[image] "+label, childStyle))
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

func tableBlockNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := make([]*ui.DocumentNode, 0, len(node.Children)+1)
	for _, child := range node.Children {
		if child == nil || child.Type != dom.ElementNode {
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
	return ui.NewDocumentElement("table", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(8)
		style.SetBorder(1, 0xD8DEE4)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
	}), children...)
}

func appendTableRowNodes(out *[]*ui.DocumentNode, node *dom.Node, ctx *renderContext) {
	if out == nil || node == nil {
		return
	}
	for _, child := range node.Children {
		if child == nil || child.Type != dom.ElementNode {
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

func tableRowNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	cells := make([]*ui.DocumentNode, 0, len(node.Children))
	for _, child := range node.Children {
		if child == nil || child.Type != dom.ElementNode {
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
	return ui.NewDocumentElement("table-row", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 4, 0)
		style.SetContain(ui.ContainPaint)
	}), cells...)
}

func tableCellNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	header := node.Tag == "th"
	textStyle := paragraphInlineStyle()
	if header {
		textStyle.SetForeground(ui.Navy)
	}
	builder := inlinePieceBuilder{}
	for _, child := range node.Children {
		collectInlinePieces(&builder, child, ctx)
	}
	children := inlineNodesFromPieces(builder.pieces, textStyle, ctx)
	if len(children) == 0 {
		text := collectNodeText(node, false)
		if text == "" {
			return nil
		}
		children = inlineTextTokens(text, textStyle)
	}
	return ui.NewDocumentElement("table-cell", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 6, 0, 0)
		style.SetPadding(2, 6)
		style.SetBorderRadius(6)
		style.SetContain(ui.ContainPaint)
		if header {
			style.SetBorder(1, 0xB5C7DD)
			style.SetBackground(0xEEF4FB)
			style.SetForeground(ui.Navy)
			return
		}
		style.SetBorder(1, 0xD8DEE4)
		style.SetBackground(0xF8FAFC)
	}), children...)
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

func imageFallbackNode(node *dom.Node) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	label := normalizeBlockText(node.Attrs["alt"])
	if label == "" {
		label = displayURL(strings.TrimSpace(node.Attrs["src"]))
	}
	if label == "" {
		return nil
	}
	return ui.NewDocumentElement("image", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(8)
		style.SetBackground(0xF6F8FA)
		style.SetBorder(1, 0xD8DEE4)
	}), ui.NewDocumentText("[image] "+label, styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})))
}

func hasAttr(node *dom.Node, name string) bool {
	if node == nil || node.Attrs == nil {
		return false
	}
	_, ok := node.Attrs[name]
	return ok
}

func attrValue(node *dom.Node, name string) string {
	if node == nil || node.Attrs == nil {
		return ""
	}
	return strings.TrimSpace(node.Attrs[name])
}

func attrInt(node *dom.Node, name string, fallback int) int {
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

func htmlInputType(node *dom.Node) string {
	value := strings.ToLower(attrValue(node, "type"))
	if value == "" {
		return "text"
	}
	return value
}

func htmlButtonType(node *dom.Node) string {
	if node == nil {
		return "button"
	}
	if node.Tag == "button" {
		value := strings.ToLower(attrValue(node, "type"))
		if value == "" {
			return "submit"
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

func htmlControlLabel(node *dom.Node, fallback string) string {
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

func htmlOptionValue(node *dom.Node) string {
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
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Navy)
		style.SetFontPath(webMonoFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
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

func htmlButtonNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
	button := ui.NewDocumentElement("html-button", htmlControlStyle(),
		ui.NewDocumentText(label, htmlControlTextStyle()),
	)
	button.Focusable = true
	applyInteractiveControlStyles(button)
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
			fields: func(submitter *dom.Node) []formField {
				if submitter != node || name == "" {
					return nil
				}
				return []formField{{name: name, value: value}}
			},
		})
	}
	return button
}

func htmlInputNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	switch htmlInputType(node) {
	case "hidden":
		if form := ctx.formForControl(node); form != nil {
			name := attrValue(node, "name")
			value := attrValue(node, "value")
			form.addControl(&formControlState{
				node: node,
				fields: func(_ *dom.Node) []formField {
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

func htmlTextInputNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
	input.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Teal)
	})
	input.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Blue)
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})
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
			fields: func(_ *dom.Node) []formField {
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

func htmlCheckboxNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	checked := hasAttr(node, "checked")
	initialChecked := checked
	indicator := ui.NewDocumentText("[ ]", htmlControlIndicatorStyle())
	if checked {
		indicator.Text = "[x]"
	}
	label := htmlControlLabel(node, "Checkbox")
	control := ui.NewDocumentElement("html-checkbox", htmlControlStyle(),
		indicator,
		ui.NewDocumentText(" "+label, htmlControlTextStyle()),
	)
	control.Focusable = true
	applyInteractiveControlStyles(control)
	disabled := hasAttr(node, "disabled")
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		value := attrValue(node, "value")
		if value == "" {
			value = "on"
		}
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *dom.Node) []formField {
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
				if checked {
					indicator.Text = "[x]"
				} else {
					indicator.Text = "[ ]"
				}
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
		if checked {
			indicator.Text = "[x]"
		} else {
			indicator.Text = "[ ]"
		}
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

func htmlRadioNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	checked := hasAttr(node, "checked")
	initialChecked := checked
	indicator := ui.NewDocumentText("( )", htmlControlIndicatorStyle())
	if checked {
		indicator.Text = "(o)"
	}
	label := htmlControlLabel(node, "Radio")
	control := ui.NewDocumentElement("html-radio", htmlControlStyle(),
		indicator,
		ui.NewDocumentText(" "+label, htmlControlTextStyle()),
	)
	control.Focusable = true
	applyInteractiveControlStyles(control)
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
			fields: func(_ *dom.Node) []formField {
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
				if state.checked {
					state.indicator.Text = "(o)"
				} else {
					state.indicator.Text = "( )"
				}
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
						candidate.indicator.Text = "(o)"
						changed = true
					}
					continue
				}
				if candidate.checked {
					candidate.checked = false
					candidate.indicator.Text = "( )"
					changed = true
				}
			}
		} else if !state.checked {
			state.checked = true
			state.indicator.Text = "(o)"
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

func htmlRangeNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
	control.Focusable = true
	applyInteractiveControlStyles(control)
	disabled := hasAttr(node, "disabled")
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *dom.Node) []formField {
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

func htmlTextareaNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
			fields: func(_ *dom.Node) []formField {
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

func htmlSelectNode(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
	type optionState struct {
		label string
		value string
	}
	options := make([]optionState, 0, len(node.Children))
	selected := 0
	for _, child := range node.Children {
		if child == nil || child.Type != dom.ElementNode || child.Tag != "option" {
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
	control.Focusable = true
	applyInteractiveControlStyles(control)
	disabled := hasAttr(node, "disabled")
	if form := ctx.formForControl(node); form != nil {
		name := attrValue(node, "name")
		form.addControl(&formControlState{
			node: node,
			fields: func(_ *dom.Node) []formField {
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

func htmlProgressNode(node *dom.Node) *ui.DocumentNode {
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

func documentLinkNodeFromAnchor(node *dom.Node, ctx *renderContext) *ui.DocumentNode {
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
	return card
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
