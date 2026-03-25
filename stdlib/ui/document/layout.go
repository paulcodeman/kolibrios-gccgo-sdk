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

func displayUsesBlockFlow(display DisplayMode) bool {
	return display == DisplayBlock || display == DisplayFlex
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

type documentFlexItem struct {
	node      *DocumentNode
	style     Style
	display   DisplayMode
	margin    Spacing
	grow      int
	baseWidth int
	fragment  *Fragment
}

func flexAlignItems(style Style) AlignItemsMode {
	if value, ok := resolveAlignItems(style.alignItems); ok {
		return value
	}
	return AlignItemsFlexStart
}

func flexBaseOuterWidth(style Style) int {
	if value, ok := explicitOuterWidth(style); ok {
		return clampWidthForStyle(style, value)
	}
	return clampWidthForStyle(style, 0)
}

func setOuterWidthForStyle(style *Style, outerWidth int) {
	if style == nil {
		return
	}
	if outerWidth < 0 {
		outerWidth = 0
	}
	width := outerWidth
	if effectiveBoxSizing(*style) == BoxSizingContentBox {
		insets := boxInsets(*style)
		width -= insets.Left + insets.Right
		if width < 0 {
			width = 0
		}
	}
	style.SetWidth(width)
}

func (document *Document) layoutFlexChildren(ctx LayoutContext, parentStyle Style, nodes []*DocumentNode, container Rect) ([]*Fragment, int) {
	if len(nodes) == 0 {
		return nil, container.Y
	}
	items := make([]documentFlexItem, 0, len(nodes))
	children := make([]*Fragment, 0, len(nodes))
	usedWidth := 0
	totalGrow := 0
	for _, child := range nodes {
		if child == nil {
			continue
		}
		style := documentComputedStyle(parentStyle, child)
		display := documentDisplay(style, child.Kind)
		if display == DisplayNone {
			continue
		}
		if effectivePosition(style) == PositionAbsolute {
			if fragment := document.layoutStyledNode(ctx, child, style, display, container, 0, container.Y); fragment != nil {
				children = append(children, fragment)
			}
			continue
		}
		margin, _ := resolveSpacingNormalized(style.margin)
		grow, _ := resolveFlexGrow(style.flexGrow)
		baseWidth := flexBaseOuterWidth(style)
		items = append(items, documentFlexItem{
			node:      child,
			style:     style,
			display:   display,
			margin:    margin,
			grow:      grow,
			baseWidth: baseWidth,
		})
		usedWidth += margin.Left + baseWidth + margin.Right
		totalGrow += grow
	}
	remainingWidth := container.Width - usedWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	cursorX := 0
	crossSize := 0
	growRemaining := remainingWidth
	growUnitsRemaining := totalGrow
	for index := range items {
		item := &items[index]
		outerWidth := item.baseWidth
		if item.grow > 0 && growUnitsRemaining > 0 && growRemaining > 0 {
			share := growRemaining * item.grow / growUnitsRemaining
			outerWidth += share
			growRemaining -= share
			growUnitsRemaining -= item.grow
		}
		childStyle := item.style
		setOuterWidthForStyle(&childStyle, outerWidth)
		childContainer := Rect{
			X:      container.X + cursorX,
			Y:      container.Y,
			Width:  outerWidth + item.margin.Left + item.margin.Right,
			Height: container.Height,
		}
		fragment := document.layoutStyledNode(ctx, item.node, childStyle, item.display, childContainer, 0, container.Y)
		if fragment == nil {
			continue
		}
		item.fragment = fragment
		children = append(children, fragment)
		outerHeight := fragment.Bounds.Height + item.margin.Top + item.margin.Bottom
		if outerHeight > crossSize {
			crossSize = outerHeight
		}
		cursorX += item.margin.Left + fragment.Bounds.Width + item.margin.Right
	}
	alignItems := flexAlignItems(parentStyle)
	if alignItems == AlignItemsCenter || alignItems == AlignItemsFlexEnd {
		for index := range items {
			item := &items[index]
			if item.fragment == nil {
				continue
			}
			outerHeight := item.fragment.Bounds.Height + item.margin.Top + item.margin.Bottom
			shiftY := 0
			switch alignItems {
			case AlignItemsCenter:
				shiftY = (crossSize - outerHeight) / 2
			case AlignItemsFlexEnd:
				shiftY = crossSize - outerHeight
			}
			if shiftY > 0 {
				shiftFragmentTree(item.fragment, 0, shiftY)
			}
		}
	}
	return children, container.Y + crossSize
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
	if displayUsesBlockFlow(display) {
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
	children := []*Fragment(nil)
	flowBottom := contentY
	if display == DisplayFlex {
		children, flowBottom = document.layoutFlexChildren(ctx, style, node.Children, childContainer)
	} else {
		children, flowBottom = document.layoutChildren(ctx, style, node.Children, childContainer)
	}
	height, heightSet := explicitOuterHeight(style)
	if !heightSet {
		contentHeight := 0
		if flowBottom > contentY {
			contentHeight = flowBottom - contentY
		}
		height = insets.Top + contentHeight + insets.Bottom
	}
	height = clampHeightForStyle(style, height)
	if !plan.widthSet && !displayUsesBlockFlow(display) {
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
	if display != DisplayFlex && (displayUsesBlockFlow(display) || plan.widthSet) {
		finalContentWidth := plan.width - insets.Left - insets.Right
		if finalContentWidth < 0 {
			finalContentWidth = 0
		}
		alignInlineChildFragments(style, Rect{
			X:      contentX,
			Y:      contentY,
			Width:  finalContentWidth,
			Height: container.Height,
		}, children)
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

func alignInlineChildFragments(style Style, content Rect, children []*Fragment) {
	if len(children) == 0 || content.Width <= 0 {
		return
	}
	align, ok := resolveTextAlign(style.textAlign)
	if !ok || align == TextAlignLeft {
		return
	}
	lineStart := -1
	lineRight := content.X
	lineY := 0
	flush := func(end int) {
		if lineStart < 0 || end <= lineStart {
			lineStart = -1
			lineRight = content.X
			return
		}
		lineWidth := lineRight - content.X
		if lineWidth <= 0 || lineWidth >= content.Width {
			lineStart = -1
			lineRight = content.X
			return
		}
		shift := content.Width - lineWidth
		if shift <= 0 {
			lineStart = -1
			lineRight = content.X
			return
		}
		if align == TextAlignCenter {
			shift /= 2
		}
		if shift > 0 {
			shiftFragments(children[lineStart:end], shift, 0)
		}
		lineStart = -1
		lineRight = content.X
	}
	for index, child := range children {
		if child == nil {
			continue
		}
		if effectivePosition(child.Style) == PositionAbsolute {
			flush(index)
			continue
		}
		childKind := DocumentNodeElement
		if child.Node != nil {
			childKind = child.Node.Kind
		} else if child.Kind == FragmentKindText {
			childKind = DocumentNodeText
		}
		childDisplay := documentDisplay(child.Style, childKind)
		if displayUsesBlockFlow(childDisplay) {
			flush(index)
			continue
		}
		if lineStart >= 0 && child.Bounds.Y != lineY {
			flush(index)
		}
		if lineStart < 0 {
			lineStart = index
			lineY = child.Bounds.Y
		}
		right := child.Bounds.X + child.Bounds.Width
		if margin, ok := resolveSpacingNormalized(child.Style.margin); ok {
			right += margin.Right
		}
		if right > lineRight {
			lineRight = right
		}
	}
	flush(len(children))
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
		if !displayUsesBlockFlow(display) {
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
