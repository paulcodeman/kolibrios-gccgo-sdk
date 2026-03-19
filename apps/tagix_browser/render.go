package main

import (
	"dom"
	"os"
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

type shellActions struct {
	Back        func()
	Forward     func()
	Reload      func()
	Home        func()
	EditAddress func()
}

func styled(update func(*ui.Style)) ui.Style {
	value := ui.Style{}
	if update != nil {
		update(&value)
	}
	return value
}

func buildShellDocument(title string, status string, currentURL string, canBack bool, canForward bool, canEditAddress bool, actions shellActions) *ui.DocumentNode {
	doc := dom.Parse(shellTemplateHTML(title, status, currentURL, canBack, canForward, canEditAddress))
	if doc != nil && doc.Root != nil {
		if root := buildShellTemplateRoot(doc.Root, actions); root != nil {
			return root
		}
	}
	return ui.NewDocumentElement("browser-shell", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(2)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
		style.SetContain(ui.ContainPaint)
	}), messageCard("Tagix Browser shell", "Failed to build the shell document template."))
}

func shellTemplateHTML(title string, status string, currentURL string, canBack bool, canForward bool, canEditAddress bool) string {
	if strings.TrimSpace(title) == "" {
		title = "Tagix Browser"
	}
	if strings.TrimSpace(status) == "" {
		status = "Ready"
	}
	if strings.TrimSpace(currentURL) == "" {
		currentURL = defaultURL
	}
	replacer := strings.NewReplacer(
		"<<title>>", escapeHTMLText(title),
		"<<status>>", escapeHTMLText(status),
		"<<current_url>>", escapeHTMLText(currentURL),
		"<<back_enabled>>", boolAttr(canBack),
		"<<forward_enabled>>", boolAttr(canForward),
		"<<address_editable>>", boolAttr(canEditAddress),
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

func buildShellTemplateRoot(node *dom.Node, actions shellActions) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	if built := buildShellTemplateNode(node, actions); built != nil {
		return built
	}
	for _, child := range node.Children {
		if built := buildShellTemplateRoot(child, actions); built != nil {
			return built
		}
	}
	return nil
}

func buildShellTemplateNode(node *dom.Node, actions shellActions) *ui.DocumentNode {
	if node == nil || node.Type != dom.ElementNode {
		return nil
	}
	role := strings.TrimSpace(node.Attrs["data-role"])
	switch role {
	case "shell-root":
		return ui.NewDocumentElement("browser-shell", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetPadding(2)
			style.SetFontPath(webSansFontPath)
			style.SetFontSize(13)
			style.SetLineHeight(18)
			style.SetForeground(ui.Black)
			style.SetContain(ui.ContainPaint)
		}), buildShellChildren(node, actions)...)
	case "hero":
		return ui.NewDocumentElement("browser-shell-hero", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 10, 0)
			style.SetPadding(10, 12)
			style.SetBorderRadius(12)
			style.SetGradient(ui.Gradient{
				From:      ui.Navy,
				To:        ui.Blue,
				Direction: ui.GradientHorizontal,
			})
			style.SetContain(ui.ContainPaint)
		}), buildShellChildren(node, actions)...)
	case "title":
		return ui.NewDocumentText(collectNodeText(node, false), styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 4, 0)
			style.SetForeground(ui.White)
			style.SetFontSize(18)
			style.SetLineHeight(22)
		}))
	case "status":
		return ui.NewDocumentText(collectNodeText(node, false), styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(ui.Silver)
			style.SetFontSize(11)
			style.SetLineHeight(15)
		}))
	case "toolbar":
		return ui.NewDocumentElement("browser-shell-toolbar", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetPadding(10)
			style.SetBorderRadius(12)
			style.SetBackground(ui.White)
			style.SetBorder(1, ui.Silver)
			style.SetContain(ui.ContainPaint)
		}), buildShellChildren(node, actions)...)
	case "actions":
		return ui.NewDocumentElement("browser-shell-actions", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 8, 0)
		}), buildShellChildren(node, actions)...)
	case "button":
		return shellButtonNode(collectNodeText(node, false), strings.TrimSpace(node.Attrs["data-action"]), attrIsTrue(node.Attrs["data-enabled"]), actions)
	case "address":
		return shellAddressNode(collectNodeText(node, false), attrIsTrue(node.Attrs["data-editable"]), actions)
	case "hint":
		return ui.NewDocumentText(collectNodeText(node, false), styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(8, 0, 0, 0)
			style.SetForeground(ui.Gray)
			style.SetFontSize(11)
			style.SetLineHeight(16)
		}))
	default:
		children := buildShellChildren(node, actions)
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

func buildShellChildren(node *dom.Node, actions shellActions) []*ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := make([]*ui.DocumentNode, 0, len(node.Children))
	for _, child := range node.Children {
		built := buildShellTemplateNode(child, actions)
		if built != nil {
			children = append(children, built)
		}
	}
	return children
}

func shellButtonNode(label string, action string, enabled bool, actions shellActions) *ui.DocumentNode {
	label = normalizeBlockText(label)
	if label == "" {
		if action != "" {
			label = strings.ToUpper(action[:1]) + action[1:]
		} else {
			label = "Action"
		}
	}
	button := ui.NewDocumentElement("shell-button", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 8, 8, 0)
		style.SetPadding(6, 10)
		style.SetBorderRadius(8)
		style.SetBorder(1, ui.Silver)
		style.SetBackground(ui.Silver)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(label, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
	})))
	if !enabled {
		button.Style.SetBackground(ui.White)
		button.Style.SetOpacity(180)
		button.Style.SetForeground(ui.Gray)
		button.Children[0].Style.SetForeground(ui.Gray)
		return button
	}
	button.Focusable = true
	button.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Teal)
		style.SetBackground(ui.Aqua)
	})
	button.StyleActive = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Navy)
		style.SetBackground(ui.Silver)
	})
	button.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Blue)
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})
	if handler := shellActionHandler(action, actions); handler != nil {
		button.OnClick = handler
	}
	return button
}

func shellAddressNode(currentURL string, editable bool, actions shellActions) *ui.DocumentNode {
	currentURL = strings.TrimSpace(currentURL)
	if currentURL == "" {
		currentURL = defaultURL
	}
	card := ui.NewDocumentElement("shell-address", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(8, 10)
		style.SetBorderRadius(10)
		style.SetBorder(1, ui.Silver)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText("Address", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 2, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(10)
	})), ui.NewDocumentText(currentURL, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	})))
	if !editable {
		card.Children[1].Style.SetForeground(ui.Gray)
		card.Children[1].Style.SetOpacity(200)
		return card
	}
	card.Focusable = true
	card.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Teal)
		style.SetBackground(ui.Aqua)
	})
	card.StyleActive = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Navy)
		style.SetBackground(ui.Silver)
	})
	card.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Blue)
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})
	card.OnClick = actions.EditAddress
	return card
}

func shellActionHandler(action string, actions shellActions) func() {
	switch action {
	case "back":
		return actions.Back
	case "forward":
		return actions.Forward
	case "reload":
		return actions.Reload
	case "home":
		return actions.Home
	case "address":
		return actions.EditAddress
	default:
		return nil
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
<header data-role="hero">
<h1 data-role="title"><<title>></h1>
<p data-role="status"><<status>></p>
</header>
<div data-role="toolbar">
<div data-role="actions">
<a data-role="button" data-action="back" data-enabled="<<back_enabled>>">Back</a>
<a data-role="button" data-action="forward" data-enabled="<<forward_enabled>>">Forward</a>
<a data-role="button" data-action="reload" data-enabled="true">Reload</a>
<a data-role="button" data-action="home" data-enabled="true">Home</a>
</div>
<a data-role="address" data-action="address" data-editable="<<address_editable>>"><<current_url>></a>
</div>
<p data-role="hint">Browser shell now renders from an HTML template through the same document pipeline. The page below lives in its own embedded frame host.</p>
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

func buildRenderedDocument(title string, currentURL string, doc *dom.Document, openURL func(string)) *ui.DocumentNode {
	children := make([]*ui.DocumentNode, 0, 24)

	titleText := title
	if titleText == "" {
		titleText = displayURL(currentURL)
		if titleText == "" {
			titleText = "Rendered page"
		}
	}
	children = append(children, ui.NewDocumentElement("hero", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(10, 12)
		style.SetBorderRadius(10)
		style.SetGradient(ui.Gradient{
			From:      ui.Aqua,
			To:        ui.White,
			Direction: ui.GradientHorizontal,
		})
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(titleText, styled(func(style *ui.Style) {
		style.SetForeground(ui.Navy)
		style.SetFontSize(18)
		style.SetMargin(0, 0, 4, 0)
	})), ui.NewDocumentText(currentURL, styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
	}))))

	contentNodes := buildDocumentNodes(doc, currentURL, openURL)
	if len(contentNodes) == 0 {
		children = append(children, messageCard("No renderable content", "The HTML5 parser returned a tree, but the current browser-host adapter did not find readable nodes yet."))
	} else {
		content := ui.NewDocumentElement("content", styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 10, 0)
			style.SetContain(ui.ContainPaint)
		}), contentNodes...)
		children = append(children, content)
	}

	children = append(children, ui.NewDocumentElement("note", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(6, 0, 0, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(10)
		style.SetBackground(ui.Silver)
	}), ui.NewDocumentText("This page content is rendered in a separate document host below the browser shell. The next stage can improve inline and CSS-like semantics on top of the same pipeline.", styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
	}))))

	return ui.NewDocumentElement("page", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(2)
		style.SetContain(ui.ContainPaint)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
	}), children...)
}

func buildDocumentNodes(doc *dom.Document, baseURL string, openURL func(string)) []*ui.DocumentNode {
	if doc == nil || doc.Root == nil {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, 16)
	appendDocumentNodes(&nodes, doc.Root, baseURL, openURL)
	return nodes
}

func appendDocumentNodes(out *[]*ui.DocumentNode, node *dom.Node, baseURL string, openURL func(string)) {
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
		for _, child := range node.Children {
			appendDocumentNodes(out, child, baseURL, openURL)
		}
		return
	case dom.ElementNode:
	default:
		return
	}

	switch node.Tag {
	case "script", "style", "head", "title", "meta", "link":
		return
	case "html", "body", "main", "section", "article", "aside", "nav", "header", "footer", "div", "form", "table", "tbody", "thead", "tfoot", "tr", "td", "th":
		for _, child := range node.Children {
			appendDocumentNodes(out, child, baseURL, openURL)
		}
		return
	case "hr":
		*out = append(*out, separatorNode())
		return
	case "br":
		return
	case "h1", "h2", "h3", "h4", "h5", "h6":
		if heading := headingBlockNode(node, baseURL, openURL); heading != nil {
			*out = append(*out, heading)
		}
		return
	case "p", "blockquote":
		if paragraph := paragraphBlockNode(node, baseURL, openURL); paragraph != nil {
			*out = append(*out, paragraph)
		}
		return
	case "pre", "code":
		text := collectNodeTextPreserve(node, true)
		if text != "" {
			*out = append(*out, preformattedNode(text))
		}
		return
	case "ul", "ol":
		appendListNodes(out, node, baseURL, openURL)
		return
	case "li":
		if item := listItemBlockNode(node, baseURL, openURL); item != nil {
			*out = append(*out, item)
		}
		return
	case "a":
		if link := standaloneLinkNode(node, baseURL, openURL); link != nil {
			*out = append(*out, link)
		}
		return
	case "img":
		if image := imageFallbackNode(node); image != nil {
			*out = append(*out, image)
		}
		return
	default:
		text := collectNodeText(node, true)
		if text != "" {
			*out = append(*out, paragraphNode(text))
		} else {
			for _, child := range node.Children {
				appendDocumentNodes(out, child, baseURL, openURL)
			}
		}
		appendNestedDocumentLinks(out, node, baseURL, openURL)
	}
}

func appendListNodes(out *[]*ui.DocumentNode, node *dom.Node, baseURL string, openURL func(string)) {
	if out == nil || node == nil {
		return
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		if child.Type == dom.ElementNode && child.Tag == "li" {
			appendDocumentNodes(out, child, baseURL, openURL)
			continue
		}
		appendDocumentNodes(out, child, baseURL, openURL)
	}
}

func appendNestedDocumentLinks(out *[]*ui.DocumentNode, node *dom.Node, baseURL string, openURL func(string)) {
	if out == nil || node == nil {
		return
	}
	for _, child := range node.Children {
		appendDirectAnchorNodes(out, child, baseURL, openURL)
	}
}

func appendDirectAnchorNodes(out *[]*ui.DocumentNode, node *dom.Node, baseURL string, openURL func(string)) {
	if out == nil || node == nil {
		return
	}
	if node.Type == dom.ElementNode && node.Tag == "a" {
		if link := documentLinkNodeFromAnchor(node, baseURL, openURL); link != nil {
			*out = append(*out, link)
		}
		return
	}
	for _, child := range node.Children {
		appendDirectAnchorNodes(out, child, baseURL, openURL)
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
)

type inlinePiece struct {
	kind inlinePieceKind
	text string
	href string
}

type inlinePieceBuilder struct {
	pieces    []inlinePiece
	needSpace bool
}

func paragraphNode(text string) *ui.DocumentNode {
	return ui.NewDocumentText(text, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}))
}

func headingBlockNode(node *dom.Node, baseURL string, openURL func(string)) *ui.DocumentNode {
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
	children := buildInlineNodes(node, baseURL, openURL, styled(func(style *ui.Style) {
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

func paragraphBlockNode(node *dom.Node, baseURL string, openURL func(string)) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, baseURL, openURL, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}))
	if len(children) == 0 {
		return nil
	}
	name := "paragraph"
	blockStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	})
	if node.Tag == "blockquote" {
		name = "blockquote"
		blockStyle = styled(func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetMargin(0, 0, 10, 0)
			style.SetPadding(6, 10)
			style.SetBorder(1, ui.Silver)
			style.SetBorderRadius(8)
			style.SetForeground(ui.Black)
			style.SetFontSize(13)
			style.SetLineHeight(18)
			style.SetContain(ui.ContainPaint)
		})
	}
	return ui.NewDocumentElement(name, blockStyle, children...)
}

func listItemBlockNode(node *dom.Node, baseURL string, openURL func(string)) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	children := buildInlineNodes(node, baseURL, openURL, styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	}))
	if len(children) == 0 {
		return nil
	}
	itemChildren := make([]*ui.DocumentNode, 0, len(children)+2)
	itemChildren = append(itemChildren, ui.NewDocumentText("- ", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInline)
		style.SetForeground(ui.Navy)
		style.SetFontSize(13)
		style.SetLineHeight(18)
	})))
	itemChildren = append(itemChildren, children...)
	return ui.NewDocumentElement("list-item", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 10)
		style.SetForeground(ui.Black)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetContain(ui.ContainPaint)
	}), itemChildren...)
}

func standaloneLinkNode(node *dom.Node, baseURL string, openURL func(string)) *ui.DocumentNode {
	link := inlineLinkNodeFromAnchor(node, baseURL, openURL)
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

func buildInlineNodes(node *dom.Node, baseURL string, openURL func(string), baseStyle ui.Style) []*ui.DocumentNode {
	builder := inlinePieceBuilder{}
	for _, child := range node.Children {
		collectInlinePieces(&builder, child, baseURL)
	}
	return inlineNodesFromPieces(builder.pieces, baseStyle, openURL)
}

func collectInlinePieces(builder *inlinePieceBuilder, node *dom.Node, baseURL string) {
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
			collectInlinePieces(builder, child, baseURL)
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
	default:
		for _, child := range node.Children {
			collectInlinePieces(builder, child, baseURL)
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

func inlineNodesFromPieces(pieces []inlinePiece, baseStyle ui.Style, openURL func(string)) []*ui.DocumentNode {
	if len(pieces) == 0 {
		return nil
	}
	nodes := make([]*ui.DocumentNode, 0, len(pieces))
	for _, piece := range pieces {
		switch piece.kind {
		case inlinePieceText:
			nodes = append(nodes, inlineTextNode(piece.text, baseStyle))
		case inlinePieceLink:
			if link := inlineLinkNode(piece.text, piece.href, baseStyle, openURL); link != nil {
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
		}
	}
	return nodes
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

func inlineLinkNodeFromAnchor(node *dom.Node, baseURL string, openURL func(string)) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
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
	}), openURL)
}

func inlineLinkNode(label string, href string, baseStyle ui.Style, openURL func(string)) *ui.DocumentNode {
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
	if openURL != nil && href != "" {
		link.OnClick = func() {
			openURL(href)
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
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
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
		style.SetBackground(ui.Silver)
		style.SetBorder(1, ui.Silver)
	}), ui.NewDocumentText("[image] "+label, styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})))
}

func buildMessageDocument(title string, detail string) *ui.DocumentNode {
	return ui.NewDocumentElement("message-page", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(2)
		style.SetFontPath(webSansFontPath)
		style.SetFontSize(13)
		style.SetLineHeight(18)
		style.SetForeground(ui.Black)
	}), messageCard(title, detail))
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

func documentLinkNodeFromAnchor(node *dom.Node, baseURL string, openURL func(string)) *ui.DocumentNode {
	if node == nil || node.Attrs == nil {
		return nil
	}
	href := resolveURL(baseURL, node.Attrs["href"])
	if href == "" {
		return nil
	}
	label := collectNodeText(node, false)
	if label == "" {
		label = displayURL(href)
	}
	return documentLinkCard(label, href, openURL)
}

func documentLinkCard(label string, href string, openURL func(string)) *ui.DocumentNode {
	title := label
	if title == "" {
		title = displayURL(href)
	}
	card := ui.NewDocumentElement("link", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetPadding(8, 10)
		style.SetBorderRadius(10)
		style.SetBorder(1, ui.Silver)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
	}), ui.NewDocumentText(title, styled(func(style *ui.Style) {
		style.SetForeground(ui.Blue)
		style.SetFontSize(13)
		style.SetMargin(0, 0, 3, 0)
	})), ui.NewDocumentText(href, styled(func(style *ui.Style) {
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
	})))
	card.Focusable = true
	card.StyleHover = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Teal)
		style.SetBackground(ui.Aqua)
	})
	card.StyleActive = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Navy)
		style.SetBackground(ui.Silver)
	})
	card.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Blue)
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})
	if openURL != nil && href != "" {
		card.OnClick = func() {
			openURL(href)
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
