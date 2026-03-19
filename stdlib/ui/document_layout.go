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

func (document *Document) Layout(ctx LayoutContext) {
	if document == nil {
		return
	}
	document.clearLayout()
	document.viewport = ctx.Viewport
	if document.Root == nil {
		return
	}
	document.fragmentByNode = make(map[*DocumentNode]*Fragment, 16)
	root, _ := document.layoutNode(ctx, Style{}, document.Root, ctx.Viewport, ctx.Viewport.Y)
	document.rootFragment = root
	document.displayList = buildFragmentDisplayList(root, ctx.Viewport)
	document.content = fragmentUnionBounds(root)
	document.invalidateHitGrid()
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
	releaseFragmentTree(document.rootFragment)
	document.rootFragment = nil
	document.displayList = FragmentDisplayList{}
	document.content = Rect{}
	document.fragmentByNode = nil
	document.invalidateHitGrid()
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
	switch node.Kind {
	case DocumentNodeText:
		return document.layoutTextNode(ctx, node, style, container, flowY)
	default:
		return document.layoutElementNode(ctx, node, style, container, flowY)
	}
}

func (document *Document) layoutChildren(ctx LayoutContext, parentStyle Style, nodes []*DocumentNode, container Rect) ([]*Fragment, int) {
	if len(nodes) == 0 {
		return nil, container.Y
	}
	children := make([]*Fragment, 0, len(nodes))
	cursorY := container.Y
	for _, child := range nodes {
		fragment, nextY := document.layoutNode(ctx, parentStyle, child, container, cursorY)
		if fragment == nil {
			continue
		}
		children = append(children, fragment)
		if nextY > cursorY {
			cursorY = nextY
		}
	}
	return children, cursorY
}

func (document *Document) layoutElementNode(ctx LayoutContext, node *DocumentNode, style Style, container Rect, flowY int) (*Fragment, int) {
	plan := planDocumentBox(style, container, flowY)
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
	return fragment, nextFlowY(plan, height, flowY)
}

func (document *Document) layoutTextNode(ctx LayoutContext, node *DocumentNode, style Style, container Rect, flowY int) (*Fragment, int) {
	if node == nil || node.Text == "" {
		return nil, flowY
	}
	plan := planDocumentBox(style, container, flowY)
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
	return fragment, nextFlowY(plan, height, flowY)
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

func planDocumentBox(style Style, container Rect, flowY int) documentBoxPlan {
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
		plan.width = container.Width - plan.margin.Left - plan.margin.Right
		if plan.position == PositionAbsolute && plan.leftSet && plan.rightSet {
			plan.width = container.Width - plan.left - plan.right - plan.margin.Left - plan.margin.Right
		}
	}
	plan.width = clampWidthForStyle(style, plan.width)
	plan.x = container.X + plan.margin.Left
	plan.y = flowY + plan.margin.Top
	if plan.position == PositionAbsolute {
		plan.y = container.Y + plan.margin.Top
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
