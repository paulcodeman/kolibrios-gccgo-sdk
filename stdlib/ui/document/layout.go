package ui

type documentBoxPlan struct {
	position  PositionMode
	margin    Spacing
	left      int
	top       int
	right     int
	bottom    int
	leftSet   bool
	topSet    bool
	rightSet  bool
	bottomSet bool
	width     int
	widthSet  bool
	x         int
	y         int
}

func documentHitGridBoundsFor(viewport Rect, content Rect) Rect {
	if !content.Empty() {
		return content
	}
	return viewport
}

func (document *Document) Layout(ctx LayoutContext) {
	if document == nil {
		return
	}
	oldHitBounds := documentHitGridBoundsFor(document.viewport, document.content)
	oldDisplay := document.resetLayoutState()
	document.viewport = ctx.Viewport
	if document.Root == nil {
		document.bumpDisplayVersion()
		newHitBounds := documentHitGridBoundsFor(document.viewport, document.content)
		if oldHitBounds != newHitBounds || !sameFragmentDisplayHitGeometry(oldDisplay, document.displayList) {
			document.bumpGeometryVersion()
			document.invalidateHitGrid()
		}
		return
	}
	document.fragmentByNode = make(map[*DocumentNode]*Fragment, 16)
	root, _ := document.layoutNode(ctx, Style{}, document.Root, ctx.Viewport, ctx.Viewport.Y)
	document.rootFragment = root
	document.displayList = buildFragmentDisplayList(root, ctx.Viewport)
	document.content = fragmentUnionBounds(root)
	document.bumpDisplayVersion()
	newHitBounds := documentHitGridBoundsFor(document.viewport, document.content)
	if oldHitBounds != newHitBounds || !sameFragmentDisplayHitGeometry(oldDisplay, document.displayList) {
		document.bumpGeometryVersion()
		document.invalidateHitGrid()
	}
}

func (document *Document) Paint(canvas *Canvas) {
	if document == nil {
		return
	}
	document.displayList.Paint(canvas, true, Rect{})
}

func (document *Document) PaintDirty(canvas *Canvas, dirty Rect) {
	if document == nil {
		return
	}
	document.displayList.Paint(canvas, false, dirty)
}

func (document *Document) PaintOffset(canvas *Canvas, offsetX int, offsetY int) {
	if document == nil {
		return
	}
	document.displayList.PaintOffset(canvas, true, Rect{}, offsetX, offsetY)
}

func (document *Document) PaintDirtyOffset(canvas *Canvas, dirty Rect, offsetX int, offsetY int) {
	if document == nil {
		return
	}
	document.displayList.PaintOffset(canvas, false, dirty, offsetX, offsetY)
}

func (document *Document) HitTest(x int, y int) *DocumentNode {
	if document == nil {
		return nil
	}
	if document.ensureHitGrid() {
		if node, ok := document.hitGrid.find(x, y, document.displayList); ok {
			return node
		}
	}
	return document.displayList.Find(x, y)
}

func (document *Document) clearLayout() {
	if document == nil {
		return
	}
	oldHitBounds := documentHitGridBoundsFor(document.viewport, document.content)
	oldDisplay := document.resetLayoutState()
	document.bumpDisplayVersion()
	newHitBounds := documentHitGridBoundsFor(document.viewport, document.content)
	if oldHitBounds != newHitBounds || !sameFragmentDisplayHitGeometry(oldDisplay, document.displayList) {
		document.bumpGeometryVersion()
		document.invalidateHitGrid()
	}
}

func (document *Document) resetLayoutState() FragmentDisplayList {
	if document == nil {
		return FragmentDisplayList{}
	}
	oldDisplay := document.displayList
	releaseFragmentTree(document.rootFragment)
	document.rootFragment = nil
	document.displayList = FragmentDisplayList{}
	document.content = Rect{}
	document.fragmentByNode = nil
	return oldDisplay
}

func (document *Document) registerFragment(fragment *Fragment) {
	if document == nil || fragment == nil || fragment.Node == nil {
		return
	}
	if document.fragmentByNode == nil {
		document.fragmentByNode = make(map[*DocumentNode]*Fragment, 16)
	}
	document.fragmentByNode[fragment.Node] = fragment
}

func (document *Document) layoutNode(ctx LayoutContext, parentStyle Style, node *DocumentNode, container Rect, flowY int) (*Fragment, int) {
	if node == nil {
		return nil, flowY
	}
	style := documentComputedStyle(parentStyle, node)
	display := documentDisplay(style, node.Kind)
	if display == DisplayNone {
		return nil, flowY
	}
	fragment := document.layoutStyledNode(ctx, node, style, display, container, 0, flowY)
	if fragment == nil {
		return nil, flowY
	}
	if effectivePosition(style) == PositionAbsolute {
		return fragment, flowY
	}
	margin, _ := resolveSpacingNormalized(style.margin)
	return fragment, flowY + fragment.Bounds.Height + margin.Top + margin.Bottom
}

func (document *Document) layoutStyledNode(ctx LayoutContext, node *DocumentNode, style Style, display DisplayMode, container Rect, flowX int, flowY int) *Fragment {
	switch node.Kind {
	case DocumentNodeText:
		return document.layoutTextNode(ctx, node, style, display, container, flowX, flowY)
	default:
		return document.layoutElementNode(ctx, node, style, display, container, flowX, flowY)
	}
}

func (document *Document) layoutChildren(ctx LayoutContext, parentStyle Style, nodes []*DocumentNode, container Rect) ([]*Fragment, int) {
	if len(nodes) == 0 {
		return nil, container.Y
	}
	children := make([]*Fragment, 0, len(nodes))
	cursorX := 0
	cursorY := container.Y
	lineHeight := 0
	maxWidth := container.Width
	for _, child := range nodes {
		fragment, nextX, nextY, nextLineHeight := document.layoutChildFlow(ctx, parentStyle, child, container, cursorX, cursorY, lineHeight, maxWidth)
		if fragment == nil {
			continue
		}
		children = append(children, fragment)
		cursorX = nextX
		cursorY = nextY
		lineHeight = nextLineHeight
	}
	flowBottom := cursorY
	if lineHeight > 0 && cursorY+lineHeight > flowBottom {
		flowBottom = cursorY + lineHeight
	}
	return children, flowBottom
}

func (document *Document) layoutChildFlow(ctx LayoutContext, parentStyle Style, node *DocumentNode, container Rect, cursorX int, cursorY int, lineHeight int, maxWidth int) (*Fragment, int, int, int) {
	if node == nil {
		return nil, cursorX, cursorY, lineHeight
	}
	style := documentComputedStyle(parentStyle, node)
	display := documentDisplay(style, node.Kind)
	if display == DisplayNone {
		return nil, cursorX, cursorY, lineHeight
	}
	if effectivePosition(style) == PositionAbsolute {
		fragment := document.layoutStyledNode(ctx, node, style, display, container, 0, cursorY)
		return fragment, cursorX, cursorY, lineHeight
	}
	margin, _ := resolveSpacingNormalized(style.margin)
	if display == DisplayBlock {
		if cursorX > 0 {
			cursorX = 0
			cursorY += lineHeight
			lineHeight = 0
		}
		fragment := document.layoutStyledNode(ctx, node, style, display, container, 0, cursorY)
		if fragment == nil {
			return nil, cursorX, cursorY, lineHeight
		}
		nextY := cursorY + fragment.Bounds.Height + margin.Top + margin.Bottom
		return fragment, 0, nextY, 0
	}

	fragment := document.layoutStyledNode(ctx, node, style, display, container, cursorX, cursorY)
	if fragment == nil {
		return nil, cursorX, cursorY, lineHeight
	}
	outerW := fragment.Bounds.Width + margin.Left + margin.Right
	outerH := fragment.Bounds.Height + margin.Top + margin.Bottom
	if cursorX > 0 && outerW > 0 && cursorX+outerW > maxWidth {
		cursorX = 0
		cursorY += lineHeight
		lineHeight = 0
		fragment = document.layoutStyledNode(ctx, node, style, display, container, 0, cursorY)
		if fragment == nil {
			return nil, cursorX, cursorY, lineHeight
		}
		outerW = fragment.Bounds.Width + margin.Left + margin.Right
		outerH = fragment.Bounds.Height + margin.Top + margin.Bottom
	}
	cursorX += outerW
	if outerH > lineHeight {
		lineHeight = outerH
	}
	return fragment, cursorX, cursorY, lineHeight
}

func (document *Document) layoutElementNode(ctx LayoutContext, node *DocumentNode, style Style, display DisplayMode, container Rect, flowX int, flowY int) *Fragment {
	if documentNodeIsTextInput(node) {
		return document.layoutTextInputNode(ctx, node, style, display, container, flowX, flowY)
	}
	plan := planDocumentBox(style, display, container, flowX, flowY)
	insets := boxInsets(style)
	contentX := plan.x + insets.Left
	contentY := plan.y + insets.Top
	contentWidth := plan.width - insets.Left - insets.Right
	if contentWidth < 0 {
		contentWidth = 0
	}
	childContainer := Rect{
		X:      contentX,
		Y:      contentY,
		Width:  contentWidth,
		Height: container.Height,
	}
	children, flowBottom := document.layoutChildren(ctx, style, node.Children, childContainer)
	height, heightSet := explicitOuterHeight(style)
	if !heightSet {
		contentHeight := 0
		if flowBottom > contentY {
			contentHeight = flowBottom - contentY
		}
		height = insets.Top + contentHeight + insets.Bottom
	}
	height = clampHeightForStyle(style, height)
	if !plan.widthSet && display != DisplayBlock {
		autoWidth := insets.Left + insets.Right
		maxRight := contentX
		for _, child := range children {
			if child == nil {
				continue
			}
			right := child.Bounds.X + child.Bounds.Width
			if !child.PaintBounds.Empty() {
				right = child.PaintBounds.X + child.PaintBounds.Width
			}
			if right > maxRight {
				maxRight = right
			}
		}
		autoWidth += maxRight - contentX
		plan.width = clampWidthForStyle(style, autoWidth)
	}
	if plan.position == PositionAbsolute && !plan.topSet && plan.bottomSet {
		finalY := container.Y + container.Height - plan.bottom - plan.margin.Bottom - height
		if finalY != plan.y {
			shiftFragments(children, 0, finalY-plan.y)
			plan.y = finalY
		}
	}
	bounds := Rect{
		X:      plan.x,
		Y:      plan.y,
		Width:  plan.width,
		Height: height,
	}
	paintStyle := style
	if node != nil {
		paintStyle = mergeStyle(style, documentNodePaintStyle(node))
	}
	fragment := &Fragment{
		Kind:       FragmentKindBlock,
		Node:       node,
		Style:      style,
		PaintStyle: paintStyle,
		Bounds:     bounds,
		Content:    contentRectFor(bounds, style),
		Children:   children,
	}
	fragment.PaintBounds = fragmentPaintBounds(fragment)
	document.registerFragment(fragment)
	return fragment
}

func (document *Document) layoutTextInputNode(ctx LayoutContext, node *DocumentNode, style Style, display DisplayMode, container Rect, flowX int, flowY int) *Fragment {
	plan := planDocumentBox(style, display, container, flowX, flowY)
	height, heightSet := explicitOuterHeight(style)
	if !heightSet {
		height = documentNodeInputHeight(style)
	}
	height = clampHeightForStyle(style, height)
	if plan.position == PositionAbsolute && !plan.topSet && plan.bottomSet {
		plan.y = container.Y + container.Height - plan.bottom - plan.margin.Bottom - height
	}
	bounds := Rect{
		X:      plan.x,
		Y:      plan.y,
		Width:  plan.width,
		Height: height,
	}
	documentNodeInputEnsureCaretVisible(node, bounds, style)
	paintStyle := style
	if node != nil {
		paintStyle = mergeStyle(style, documentNodePaintStyle(node))
	}
	fragment := &Fragment{
		Kind:       FragmentKindBlock,
		Node:       node,
		Style:      style,
		PaintStyle: paintStyle,
		Bounds:     bounds,
		Content:    contentRectFor(bounds, style),
	}
	fragment.PaintBounds = fragmentPaintBounds(fragment)
	document.registerFragment(fragment)
	return fragment
}

func (document *Document) layoutTextNode(ctx LayoutContext, node *DocumentNode, style Style, display DisplayMode, container Rect, flowX int, flowY int) *Fragment {
	if node == nil || node.Text == "" {
		return nil
	}
	plan := planDocumentBox(style, display, container, flowX, flowY)
	insets := boxInsets(style)
	contentWidth := plan.width - insets.Left - insets.Right
	if contentWidth < 0 {
		contentWidth = 0
	}
	font, metrics := ctx.FontForStyle(style)
	charWidth := metrics.width
	lineHeight := lineHeightForStyle(style, metrics.height)
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	lines := node.wrapTextLinesCachedStyle(node.Text, contentWidth, font, charWidth, style)
	if display != DisplayBlock {
		inlineStyle := style
		inlineStyle.SetWhiteSpace(WhiteSpaceNoWrap)
		inlineWidth := textWidthWithFont(node.Text, font, charWidth)
		if inlineWidth < 1 {
			inlineWidth = 1
		}
		lines = node.wrapTextLinesCachedStyle(node.Text, inlineWidth, font, charWidth, inlineStyle)
		contentWidth = maxLineWidth(lines, font, charWidth)
		plan.width = insets.Left + contentWidth + insets.Right
	}
	contentHeight := len(lines) * lineHeight
	height, heightSet := explicitOuterHeight(style)
	if !heightSet {
		height = insets.Top + contentHeight + insets.Bottom
	}
	height = clampHeightForStyle(style, height)
	if plan.position == PositionAbsolute && !plan.topSet && plan.bottomSet {
		plan.y = container.Y + container.Height - plan.bottom - plan.margin.Bottom - height
	}
	bounds := Rect{
		X:      plan.x,
		Y:      plan.y,
		Width:  plan.width,
		Height: height,
	}
	paintStyle := style
	if node != nil {
		paintStyle = mergeStyle(style, documentNodePaintStyle(node))
	}
	fragment := &Fragment{
		Kind:       FragmentKindText,
		Node:       node,
		Style:      style,
		PaintStyle: paintStyle,
		Bounds:     bounds,
		Content:    contentRectFor(bounds, style),
		Text:       node.Text,
		font:       font,
		metrics:    metrics,
		lineHeight: lineHeight,
		lines:      lines,
		linesOwned: FastNoTextCache,
	}
	fragment.PaintBounds = fragmentPaintBounds(fragment)
	document.registerFragment(fragment)
	return fragment
}

func releaseFragmentTree(fragment *Fragment) {
	if fragment == nil {
		return
	}
	for _, child := range fragment.Children {
		releaseFragmentTree(child)
	}
	if fragment.linesOwned && fragment.lines != nil {
		releaseTextLines(fragment.lines)
	}
	fragment.lines = nil
	fragment.Children = nil
}

func fragmentUnionBounds(fragment *Fragment) Rect {
	if fragment == nil {
		return Rect{}
	}
	bounds := fragment.PaintBounds
	if bounds.Empty() {
		bounds = fragment.Bounds
	}
	for _, child := range fragment.Children {
		bounds = UnionRect(bounds, fragmentUnionBounds(child))
	}
	return bounds
}

func shiftFragments(fragments []*Fragment, dx int, dy int) {
	if dx == 0 && dy == 0 {
		return
	}
	for _, fragment := range fragments {
		shiftFragmentTree(fragment, dx, dy)
	}
}

func shiftFragmentTree(fragment *Fragment, dx int, dy int) {
	if fragment == nil {
		return
	}
	fragment.Bounds.X += dx
	fragment.Bounds.Y += dy
	fragment.PaintBounds.X += dx
	fragment.PaintBounds.Y += dy
	fragment.Content.X += dx
	fragment.Content.Y += dy
	for _, child := range fragment.Children {
		shiftFragmentTree(child, dx, dy)
	}
}

func planDocumentBox(style Style, display DisplayMode, container Rect, flowX int, flowY int) documentBoxPlan {
	plan := documentBoxPlan{}
	plan.margin, _ = resolveSpacingNormalized(style.margin)
	if position, ok := resolvePosition(style.position); ok {
		plan.position = position
	}
	plan.left, plan.leftSet = resolveLength(style.left)
	plan.top, plan.topSet = resolveLength(style.top)
	plan.right, plan.rightSet = resolveLength(style.right)
	plan.bottom, plan.bottomSet = resolveLength(style.bottom)
	plan.width, plan.widthSet = explicitOuterWidth(style)
	if !plan.widthSet {
		availableWidth := container.Width - plan.margin.Left - plan.margin.Right
		if display != DisplayBlock {
			availableWidth -= flowX
		}
		plan.width = availableWidth
		if plan.position == PositionAbsolute && plan.leftSet && plan.rightSet {
			plan.width = container.Width - plan.left - plan.right - plan.margin.Left - plan.margin.Right
		}
	}
	plan.width = clampWidthForStyle(style, plan.width)
	plan.x = container.X + flowX + plan.margin.Left
	plan.y = flowY + plan.margin.Top
	if plan.position == PositionAbsolute {
		plan.y = container.Y + plan.margin.Top
		plan.x = container.X + plan.margin.Left
	}
	switch plan.position {
	case PositionAbsolute:
		if plan.leftSet {
			plan.x = container.X + plan.left + plan.margin.Left
		} else if plan.rightSet {
			plan.x = container.X + container.Width - plan.right - plan.margin.Right - plan.width
		}
		if plan.topSet {
			plan.y = container.Y + plan.top + plan.margin.Top
		}
	case PositionRelative:
		if plan.leftSet {
			plan.x += plan.left
		} else if plan.rightSet {
			plan.x -= plan.right
		}
		if plan.topSet {
			plan.y += plan.top
		} else if plan.bottomSet {
			plan.y -= plan.bottom
		}
	}
	return plan
}

func nextFlowY(plan documentBoxPlan, height int, flowY int) int {
	if plan.position == PositionAbsolute {
		return flowY
	}
	return plan.y + height + plan.margin.Bottom
}

func documentDisplay(style Style, kind DocumentNodeKind) DisplayMode {
	if display, ok := resolveDisplay(style.display); ok {
		return display
	}
	if kind == DocumentNodeText {
		return DisplayBlock
	}
	return DisplayBlock
}

func documentComputedStyle(parent Style, node *DocumentNode) Style {
	inherited := Style{
		foreground:     parent.foreground,
		textAlign:      parent.textAlign,
		visibility:     parent.visibility,
		textDecoration: parent.textDecoration,
		whiteSpace:     parent.whiteSpace,
		overflowWrap:   parent.overflowWrap,
		wordBreak:      parent.wordBreak,
		textShadow:     parent.textShadow,
		fontPath:       parent.fontPath,
		fontSize:       parent.fontSize,
		lineHeight:     parent.lineHeight,
	}
	if node == nil {
		return inherited
	}
	style := mergeStyle(inherited, node.Style)
	return mergeStyle(style, documentNodeLayoutStyle(node))
}
