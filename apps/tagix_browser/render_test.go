package main

import (
	"testing"
	"ui"
)

func TestInlinePieceBuilderPreservesLeadingWhitespaceBetweenInlineNodes(t *testing.T) {
	builder := inlinePieceBuilder{}
	style := inlineTextStyle{}

	builder.appendText("Tagix Browser", style)
	builder.appendText(" is a web browser", style)

	got := inlinePieceTexts(builder.pieces)
	want := []string{"Tagix", " ", "Browser", " ", "is", " ", "a", " ", "web", " ", "browser"}
	if len(got) != len(want) {
		t.Fatalf("piece count mismatch: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("piece %d mismatch: got %q want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestInlinePieceBuilderPreservesAnchorEdgeWhitespace(t *testing.T) {
	builder := inlinePieceBuilder{}
	style := inlineTextStyle{}

	builder.appendLink(nil, " Menu ", "https://example.com/menu", style)
	builder.appendLink(nil, " Next ", "https://example.com/next", style)

	got := inlinePieceTexts(builder.pieces)
	want := []string{"Menu", " ", "Next"}
	if len(got) != len(want) {
		t.Fatalf("piece count mismatch: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("piece %d mismatch: got %q want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestNodeParticipatesInInlineFlowHonorsCSSDisplay(t *testing.T) {
	node := &Node{
		Type: ElementNode,
		Tag:  "a",
		Attrs: map[string]string{
			"style": "display:block",
		},
	}
	if nodeParticipatesInInlineFlow(node, nil) {
		t.Fatalf("display:block anchor should not stay in inline flow")
	}
	node.Attrs["style"] = "display:inline"
	if !nodeParticipatesInInlineFlow(node, nil) {
		t.Fatalf("display:inline anchor should participate in inline flow")
	}
}

func TestInlineTextNodePreservesStandaloneSpace(t *testing.T) {
	node := inlineTextNode(" ", ui.Style{})
	if node == nil {
		t.Fatalf("expected space text node")
	}
	if value, ok := node.Style.GetWhiteSpace(); !ok || value != ui.WhiteSpacePre {
		t.Fatalf("space node white-space mismatch: got %v set=%v", value, ok)
	}
}

func TestInlineLinkNodePreservesInlineImageChildren(t *testing.T) {
	anchor := &Node{
		Type: ElementNode,
		Tag:  "a",
		Attrs: map[string]string{
			"href": "https://example.com/social",
		},
	}
	image := &Node{
		Type: ElementNode,
		Tag:  "img",
		Attrs: map[string]string{
			"src": "https://example.com/icon.png",
			"alt": "Telegram",
		},
	}
	text := &Node{Type: TextNode, Text: "Telegram "}
	anchor.Children = []*Node{image, text}
	image.Parent = anchor
	text.Parent = anchor

	link := inlineLinkNode("Telegram", "https://example.com/social", anchor, ui.Style{}, nil)
	if link == nil {
		t.Fatalf("expected inline link")
	}
	if len(link.Children) < 2 {
		t.Fatalf("expected rich inline link children, got %d", len(link.Children))
	}
	if got := link.Children[0].Name; got != "inline-image" {
		t.Fatalf("first child mismatch: got %q want %q", got, "inline-image")
	}
}

func TestStandaloneLinkNodeHonorsBlockAnchorWrappingStyles(t *testing.T) {
	anchor := &Node{
		Type: ElementNode,
		Tag:  "a",
		Attrs: map[string]string{
			"href":  "https://example.com/banner",
			"style": "display:block; margin-top:16px; overflow-wrap:anywhere; text-decoration:none; color:#f9ab00",
		},
		Children: []*Node{
			{Type: TextNode, Text: "https://example.com/very/long/banner/link"},
		},
	}

	link := standaloneLinkNode(anchor, nil)
	if link == nil {
		t.Fatalf("expected standalone link")
	}
	if display, ok := link.Style.GetDisplay(); !ok || display != ui.DisplayBlock {
		t.Fatalf("standalone link display mismatch: got %v set=%v", display, ok)
	}
	margin, ok := link.Style.GetMargin()
	if !ok || margin.Top != 16 {
		t.Fatalf("standalone link margin-top mismatch: got %+v set=%v", margin, ok)
	}
	if len(link.Children) != 1 || link.Children[0] == nil || link.Children[0].Kind != ui.DocumentNodeText {
		t.Fatalf("expected single block text child, got %#v", link.Children)
	}
	child := link.Children[0]
	if display, ok := child.Style.GetDisplay(); !ok || display != ui.DisplayBlock {
		t.Fatalf("child display mismatch: got %v set=%v", display, ok)
	}
	if overflowWrap, ok := child.Style.GetOverflowWrap(); !ok || overflowWrap != ui.OverflowWrapBreakWord {
		t.Fatalf("child overflow-wrap mismatch: got %v set=%v", overflowWrap, ok)
	}
	if decoration, ok := child.Style.GetTextDecoration(); !ok || decoration != ui.TextDecorationNone {
		t.Fatalf("child text-decoration mismatch: got %v set=%v", decoration, ok)
	}
}

func TestHtmlRadioNodeDoesNotRenderValueAsVisibleLabel(t *testing.T) {
	node := &Node{
		Type: ElementNode,
		Tag:  "input",
		Attrs: map[string]string{
			"type":  "radio",
			"value": "off",
		},
	}

	control := htmlRadioNode(node, nil)
	if control == nil {
		t.Fatalf("expected radio control")
	}
	if len(control.Children) != 1 {
		t.Fatalf("radio child count mismatch: got %d want 1", len(control.Children))
	}
	if control.Children[0] == nil || control.Children[0].Text != htmlRadioGlyph(false) {
		t.Fatalf("radio indicator mismatch: got %#v", control.Children[0])
	}
	if width, ok := control.Style.GetBorderWidth(); !ok || width != 1 {
		t.Fatalf("radio border mismatch: got %d set=%v", width, ok)
	}
	if background, ok := control.Style.GetBackground(); !ok || background != ui.White {
		t.Fatalf("radio background mismatch: got %v set=%v", background, ok)
	}
}

func TestHtmlCheckboxNodeDoesNotRenderValueAsVisibleLabel(t *testing.T) {
	node := &Node{
		Type: ElementNode,
		Tag:  "input",
		Attrs: map[string]string{
			"type":  "checkbox",
			"value": "on",
		},
	}

	control := htmlCheckboxNode(node, nil)
	if control == nil {
		t.Fatalf("expected checkbox control")
	}
	if len(control.Children) != 1 {
		t.Fatalf("checkbox child count mismatch: got %d want 1", len(control.Children))
	}
	if control.Children[0] == nil || control.Children[0].Text != htmlCheckboxGlyph(false) {
		t.Fatalf("checkbox indicator mismatch: got %#v", control.Children[0])
	}
	if width, ok := control.Style.GetBorderWidth(); !ok || width != 1 {
		t.Fatalf("checkbox border mismatch: got %d set=%v", width, ok)
	}
	if background, ok := control.Style.GetBackground(); !ok || background != ui.White {
		t.Fatalf("checkbox background mismatch: got %v set=%v", background, ok)
	}
}

func TestDefinitionTermBlockNodeDoesNotForceSemanticForeground(t *testing.T) {
	node := &Node{Type: ElementNode, Tag: "dt", Children: []*Node{{Type: TextNode, Text: "toggle"}}}

	term := definitionTermBlockNode(node, nil)
	if term == nil {
		t.Fatalf("expected definition term")
	}
	if _, ok := term.Style.GetForeground(); ok {
		t.Fatalf("definition term should not force its own foreground color")
	}
}

func TestBlockquoteBlockNodeHonorsPageStyles(t *testing.T) {
	doc := Parse(`<!doctype html><html><head><style>
blockquote {
  margin: 1em 1em 1em 2em;
  border-width: 1em 1.5em 2em .5em;
  border-style: solid;
  border-color: black;
  padding: 1em 0;
  width: 5em;
  height: 9em;
  float: left;
  background-color: #FC0;
  color: black;
}
</style></head><body><blockquote>bar maids,</blockquote></body></html>`)
	ctx := &renderContext{
		stylesheet:     parseDocumentStylesheet(doc),
		viewportWidth:  800,
		viewportHeight: 600,
	}
	node := findFirstNodeByTag(doc, "blockquote")
	if node == nil {
		t.Fatalf("expected blockquote node")
	}

	quote := blockquoteBlockNode(node, ctx)
	if quote == nil {
		t.Fatalf("expected blockquote")
	}
	if value, ok := quote.Style.GetFloat(); !ok || value != ui.FloatLeft {
		t.Fatalf("blockquote float mismatch: got %v set=%v", value, ok)
	}
	if value, ok := quote.Style.GetBackground(); !ok || value != 0xFFCC00 {
		t.Fatalf("blockquote background mismatch: got %v set=%v", value, ok)
	}
	if value, ok := quote.Style.GetWidth(); !ok || value != 80 {
		t.Fatalf("blockquote width mismatch: got %d set=%v", value, ok)
	}
}

func TestBuildRenderedDocumentPrefersHTMLCanvasBackground(t *testing.T) {
	doc := Parse(`<!doctype html><html><head><style>
html { background-color: blue; color: white; }
body { background-color: white; width: 48em; border: .5em solid black; }
</style></head><body><p>hello</p></body></html>`)

	page := buildRenderedDocument("", "https://example.com", doc, 800, 600, nil, nil, nil, nil, nil, nil, nil)
	if page == nil {
		t.Fatalf("expected rendered page")
	}
	if background, ok := page.Style.GetBackground(); !ok || background != ui.Blue {
		t.Fatalf("page background mismatch: got %v set=%v", background, ok)
	}
}

func TestNestedAnchorDisplayNodeIsNotFocusable(t *testing.T) {
	anchor := &Node{
		Type: ElementNode,
		Tag:  "a",
		Attrs: map[string]string{
			"href":  "https://example.com/banner",
			"style": "display:block; margin-top:16px; overflow-wrap:anywhere",
		},
		Children: []*Node{
			{Type: TextNode, Text: "https://example.com/very/long/banner/link"},
		},
	}

	link := nestedAnchorDisplayNode(anchor, &renderContext{anchorDepth: 1})
	if link == nil {
		t.Fatalf("expected nested anchor display node")
	}
	if link.Focusable {
		t.Fatalf("nested anchor should not be focusable")
	}
	if !link.StyleHover.IsZero() || !link.StyleActive.IsZero() || !link.StyleFocus.IsZero() {
		t.Fatalf("nested anchor should not have interaction styles")
	}
}

func TestCollectInlinePiecesSuppressesNestedInteractiveAnchors(t *testing.T) {
	builder := inlinePieceBuilder{}
	style := inlineTextStyle{}
	outerCtx := &renderContext{anchorDepth: 1}
	anchor := &Node{
		Type: ElementNode,
		Tag:  "a",
		Attrs: map[string]string{
			"href": "https://example.com/nested",
		},
		Children: []*Node{
			{Type: TextNode, Text: "Nested"},
		},
	}

	collectInlinePieces(&builder, anchor, outerCtx, style)

	if len(builder.pieces) != 1 {
		t.Fatalf("piece count mismatch: got %d want 1", len(builder.pieces))
	}
	if builder.pieces[0].kind != inlinePieceText {
		t.Fatalf("expected nested anchor to collapse to text, got kind %v", builder.pieces[0].kind)
	}
	if builder.pieces[0].text != "Nested" {
		t.Fatalf("nested anchor text mismatch: got %q", builder.pieces[0].text)
	}
}

func TestStructuredBannerHoverUsesOutlineOnly(t *testing.T) {
	doc := Parse(`<!doctype html><html><head><style>
#banner{box-sizing:border-box;border-radius:4px;box-shadow:inset 0 0 0 1px rgba(0,0,0,.2509803922);transition:box-shadow .3s ease;display:block;margin:0 0 1em;padding:1.25em;text-decoration:none}
#banner:hover{box-shadow:inset 0 0 0 4px rgba(249,171,0,.6666666667)}
#banner table{table-layout:fixed}
#banner td{text-align:center;vertical-align:middle}
#banner td:first-child,#banner td:last-child{width:128px}
#banner img{height:7em}
#banner h1{color:#f9ab00;font-size:2.5em;margin:0 0 16px}
#banner p{margin:0em}
#banner a{display:block;margin-top:16px;overflow-wrap:anywhere;word-break:break-word;font-weight:bold;color:#f9ab00;border:none}
table a{padding-bottom:1px;border-bottom:1px solid;text-decoration:none;color:#0472d8}
</style></head><body>
<a id=banner href=https://example.com target=_blank>
  <table><tr>
    <td valign=top width=128><img src=/static/img/logo.png alt=КолибриОС></td>
    <td valign=top>
      <h1>KolibriOS принята в GSoC 2026!</h1>
      <p>Ознакомьтесь с деталями программы и нашими идеями проектов</p>
      <a href=https://example.com/inner>https://example.com/inner</a>
    </td>
    <td valign=top width=128><img src=/static/img/banners/gsoc.png alt=GSoC></td>
  </tr></table>
</a>
</body></html>`)
	anchor := doc.GetElementByID("banner")
	if anchor == nil {
		t.Fatalf("expected banner anchor")
	}
	ctx := &renderContext{
		baseURL:        "http://kolibrios.org/ru",
		stylesheet:     parseDocumentStylesheet(doc),
		viewportWidth:  910,
		viewportHeight: 600,
	}
	link := structuredLinkContainerNode(anchor, ctx)
	if link == nil {
		t.Fatalf("expected structured banner link")
	}
	if _, ok := link.Style.GetBorderWidth(); ok {
		t.Fatalf("banner base style should not use layout border")
	}
	baseOutline, ok := link.Style.GetOutlineWidth()
	if !ok || baseOutline != 1 {
		t.Fatalf("banner base outline mismatch: got %d set=%v", baseOutline, ok)
	}
	baseOffset, ok := link.Style.GetOutlineOffset()
	if !ok || baseOffset != -1 {
		t.Fatalf("banner base outline offset mismatch: got %d set=%v", baseOffset, ok)
	}
	if _, ok := link.StyleHover.GetBorderWidth(); ok {
		t.Fatalf("banner hover style should not use layout border")
	}
	hoverOutline, ok := link.StyleHover.GetOutlineWidth()
	if !ok || hoverOutline != 4 {
		t.Fatalf("banner hover outline mismatch: got %d set=%v", hoverOutline, ok)
	}
	hoverOffset, ok := link.StyleHover.GetOutlineOffset()
	if !ok || hoverOffset != -4 {
		t.Fatalf("banner hover outline offset mismatch: got %d set=%v", hoverOffset, ok)
	}
	heading := findDocumentNodeByName(link, "heading-h1")
	if heading == nil {
		t.Fatalf("expected heading child")
	}
	if !heading.StyleHover.IsZero() {
		t.Fatalf("heading should not receive standalone hover box style")
	}
	paragraph := findDocumentNodeByName(link, "paragraph")
	if paragraph == nil {
		t.Fatalf("expected paragraph child")
	}
	if !paragraph.StyleHover.IsZero() || !paragraph.StyleActive.IsZero() || !paragraph.StyleFocus.IsZero() {
		t.Fatalf("paragraph inside banner should not receive descendant interaction styles")
	}
	inner := findDocumentNodeByName(link, "standalone-link")
	if inner == nil {
		t.Fatalf("expected nested banner link")
	}
	if !inner.StyleHover.IsZero() || !inner.StyleActive.IsZero() || !inner.StyleFocus.IsZero() {
		t.Fatalf("nested banner link should not be interactive inside outer anchor")
	}
	if _, ok := heading.Style.GetOutlineWidth(); ok {
		t.Fatalf("heading should not receive base outline style")
	}
	if _, ok := paragraph.Style.GetOutlineWidth(); ok {
		t.Fatalf("paragraph should not receive base outline style")
	}
	if _, ok := inner.Style.GetOutlineWidth(); ok {
		t.Fatalf("nested banner link should not receive base outline style")
	}
}

func TestApplySimpleTableRowLayoutUsesFlexForThreeCellRows(t *testing.T) {
	row := ui.NewDocumentElement("table-row", ui.Style{},
		ui.NewDocumentElement("table-cell", styled(func(style *ui.Style) {
			style.SetWidth(128)
		})),
		ui.NewDocumentElement("table-cell", ui.Style{}),
		ui.NewDocumentElement("table-cell", styled(func(style *ui.Style) {
			style.SetWidth(128)
		})),
	)
	source := &Node{
		Type: ElementNode,
		Tag:  "tr",
		Children: []*Node{
			{Type: ElementNode, Tag: "td"},
			{Type: ElementNode, Tag: "td"},
			{Type: ElementNode, Tag: "td"},
		},
	}

	applySimpleTableRowLayout(row, source, nil)

	if display, ok := row.Style.GetDisplay(); !ok || display != ui.DisplayFlex {
		t.Fatalf("row display mismatch: got %v set=%v", display, ok)
	}
	if alignItems, ok := row.Style.GetAlignItems(); !ok || alignItems != ui.AlignItemsCenter {
		t.Fatalf("row align-items mismatch: got %v set=%v", alignItems, ok)
	}
	if position, ok := row.Style.GetPosition(); !ok || position != ui.PositionStatic {
		t.Fatalf("row position mismatch: got %v set=%v", position, ok)
	}
	if _, ok := row.Style.GetMinHeight(); ok {
		t.Fatalf("row should not rely on synthetic min-height")
	}
	center := row.Children[1]
	if grow, ok := center.Style.GetFlexGrow(); !ok || grow != 1000 {
		t.Fatalf("center flex-grow mismatch: got %v set=%v", grow, ok)
	}
	if margin, ok := center.Style.GetMargin(); !ok || margin.Top != 0 || margin.Right != 0 || margin.Bottom != 0 || margin.Left != 0 {
		t.Fatalf("center margin mismatch: got %+v set=%v", margin, ok)
	}
}

func TestApplyPageNodeStylesParsesPercentWidthsFromStylesheet(t *testing.T) {
	doc := Parse(`<!doctype html><html><head><style>
dt { width: 10.638%; float: left; }
#bar { width: 41.17%; }
</style></head><body><dl><dt>toggle</dt><dd><ul><li id="bar">the world ends</li></ul></dd></dl></body></html>`)
	ctx := &renderContext{
		baseURL:        "https://www.w3.org/Style/CSS/Test/CSS1/current/test5526c.htm",
		stylesheet:     parseDocumentStylesheet(doc),
		viewportWidth:  1000,
		viewportHeight: 800,
	}

	term := findFirstNodeByTag(doc, "dt")
	if term == nil {
		t.Fatalf("expected dt node")
	}
	termStyle := ui.Style{}
	applyPageNodeStyles(&termStyle, term, ctx)
	if value, ok := termStyle.GetWidthPercent(); !ok || value != 10638 {
		t.Fatalf("dt width percent mismatch: got %d set=%v", value, ok)
	}
	if value, ok := termStyle.GetFloat(); !ok || value != ui.FloatLeft {
		t.Fatalf("dt float mismatch: got %v set=%v", value, ok)
	}

	bar := doc.GetElementByID("bar")
	if bar == nil {
		t.Fatalf("expected #bar node")
	}
	barStyle := ui.Style{}
	applyPageNodeStyles(&barStyle, bar, ctx)
	if value, ok := barStyle.GetWidthPercent(); !ok || value != 41170 {
		t.Fatalf("#bar width percent mismatch: got %d set=%v", value, ok)
	}
}

func TestDocumentLayoutUsesPercentWidthsForFloats(t *testing.T) {
	leftStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetFloat(ui.FloatLeft)
		style.SetBoxSizing(ui.BoxSizingContentBox)
		style.SetWidthPercent(10638)
		style.SetHeight(280)
		style.SetPadding(10)
		style.SetBorder(5, 0)
	})
	left := ui.NewDocumentElement("left", leftStyle, ui.NewDocumentText("toggle", ui.Style{}))

	rightStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetFloat(ui.FloatRight)
		style.SetBoxSizing(ui.BoxSizingContentBox)
		style.SetWidth(340)
		style.SetHeight(270)
		style.SetPadding(10)
		style.SetBorder(10, 0)
		style.SetMargin(0, 0, 0, 10)
	})
	right := ui.NewDocumentElement("right", rightStyle, ui.NewDocumentText("content", ui.Style{}))

	clearStyle := styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetClear(ui.ClearBoth)
	})
	clearNode := ui.NewDocumentElement("clear", clearStyle, ui.NewDocumentText("after", ui.Style{}))

	root := ui.NewDocumentElement("root", styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetWidth(470)
	}), left, right, clearNode)

	document := ui.NewDocument(root)
	document.Layout(ui.DefaultLayoutContext(ui.Rect{X: 0, Y: 0, Width: 470, Height: 800}))

	leftFragment := document.FragmentForNode(left)
	if leftFragment == nil {
		t.Fatalf("expected left float fragment")
	}
	if leftFragment.Bounds.Width != 80 {
		t.Fatalf("left float width mismatch: got %d want 80", leftFragment.Bounds.Width)
	}

	rightFragment := document.FragmentForNode(right)
	if rightFragment == nil {
		t.Fatalf("expected right float fragment")
	}
	if rightFragment.Bounds.X != 90 {
		t.Fatalf("right float x mismatch: got %d want 90", rightFragment.Bounds.X)
	}
	if rightFragment.Bounds.Width != 380 {
		t.Fatalf("right float width mismatch: got %d want 380", rightFragment.Bounds.Width)
	}

	clearFragment := document.FragmentForNode(clearNode)
	if clearFragment == nil {
		t.Fatalf("expected clear fragment")
	}
	rightBottom := rightFragment.Bounds.Y + rightFragment.Bounds.Height
	if clearFragment.Bounds.Y < rightBottom {
		t.Fatalf("clear fragment should start below floats: got y=%d rightBottom=%d", clearFragment.Bounds.Y, rightBottom)
	}
}

func inlinePieceTexts(pieces []inlinePiece) []string {
	texts := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		texts = append(texts, piece.text)
	}
	return texts
}

func findDocumentNodeByName(node *ui.DocumentNode, name string) *ui.DocumentNode {
	if node == nil {
		return nil
	}
	if node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if found := findDocumentNodeByName(child, name); found != nil {
			return found
		}
	}
	return nil
}

func findFirstNodeByTag(node *Node, tag string) *Node {
	if node == nil {
		return nil
	}
	if node.Type == ElementNode && node.Tag == tag {
		return node
	}
	for _, child := range node.Children {
		if found := findFirstNodeByTag(child, tag); found != nil {
			return found
		}
	}
	return nil
}
