package ui

func (list FragmentDisplayList) Paint(canvas *Canvas, full bool, dirty Rect) {
	list.PaintOffset(canvas, full, dirty, 0, 0)
}

func (list FragmentDisplayList) itemPaintState(item FragmentDisplayItem, full bool, dirty Rect, offsetX int, offsetY int) (Rect, Rect, bool, bool) {
	if item.Fragment == nil {
		return Rect{}, Rect{}, false, false
	}
	paint := item.Paint
	if offsetX != 0 || offsetY != 0 {
		paint.X += offsetX
		paint.Y += offsetY
	}
	if paint.Empty() {
		return Rect{}, Rect{}, false, false
	}
	actual := paint
	clipSet := false
	clipRect := Rect{}
	if !full {
		if dirty.Empty() || IntersectRect(actual, dirty).Empty() {
			return Rect{}, Rect{}, false, false
		}
		if !fragmentNeedsFullDirtyPaint(item.Fragment) {
			actual = IntersectRect(actual, dirty)
			clipRect = dirty
			clipSet = true
		}
	}
	if item.ClipSet {
		clip := item.Clip
		if offsetX != 0 || offsetY != 0 {
			clip.X += offsetX
			clip.Y += offsetY
		}
		if clipSet {
			clipRect = IntersectRect(clipRect, clip)
		} else {
			clipRect = clip
			clipSet = true
		}
		actual = IntersectRect(actual, clip)
	}
	if actual.Empty() {
		return Rect{}, Rect{}, false, false
	}
	return actual, clipRect, clipSet, true
}

func (list FragmentDisplayList) itemOpaqueCoverRect(item FragmentDisplayItem, full bool, dirty Rect, offsetX int, offsetY int) (Rect, bool) {
	paint, _, _, ok := list.itemPaintState(item, full, dirty, offsetX, offsetY)
	if !ok {
		return Rect{}, false
	}
	cover, ok := fragmentOpaqueCoverRect(item.Fragment)
	if !ok {
		return Rect{}, false
	}
	if offsetX != 0 || offsetY != 0 {
		cover.X += offsetX
		cover.Y += offsetY
	}
	cover = IntersectRect(cover, paint)
	if cover.Empty() {
		return Rect{}, false
	}
	return cover, true
}

func (list FragmentDisplayList) PaintOffset(canvas *Canvas, full bool, dirty Rect, offsetX int, offsetY int) {
	if canvas == nil {
		return
	}
	var skip []bool
	if DisplayListOcclusionCulling && len(list.items) > 1 {
		skip = make([]bool, len(list.items))
		covers := make([]Rect, 0, 8)
		for i := len(list.items) - 1; i >= 0; i-- {
			item := list.items[i]
			paint, _, _, ok := list.itemPaintState(item, full, dirty, offsetX, offsetY)
			if !ok {
				continue
			}
			if rectCoveredByAny(paint, covers) {
				skip[i] = true
				continue
			}
			if cover, ok := list.itemOpaqueCoverRect(item, full, dirty, offsetX, offsetY); ok {
				covers = append(covers, cover)
			}
		}
	}
	for i, item := range list.items {
		if item.Fragment == nil {
			continue
		}
		paint, clipRect, clipSet, ok := list.itemPaintState(item, full, dirty, offsetX, offsetY)
		if !ok {
			continue
		}
		if skip != nil && skip[i] {
			continue
		}
		if clipSet {
			canvas.PushClip(clipRect)
		}
		if !full && !fragmentNeedsFullDirtyPaint(item.Fragment) {
			canvas.PushClip(paint)
		}
		item.Fragment.paintOffset(canvas, offsetX, offsetY)
		if !full && !fragmentNeedsFullDirtyPaint(item.Fragment) {
			canvas.PopClip()
		}
		if clipSet {
			canvas.PopClip()
		}
	}
}

func (list FragmentDisplayList) Find(x int, y int) *DocumentNode {
	for i := len(list.items) - 1; i >= 0; i-- {
		item := list.items[i]
		if item.Fragment == nil {
			continue
		}
		hit := item.Bounds
		if item.ClipSet {
			hit = IntersectRect(hit, item.Clip)
		}
		if hit.Contains(x, y) {
			return item.Fragment.Node
		}
	}
	return nil
}

func buildFragmentDisplayList(root *Fragment, viewport Rect) FragmentDisplayList {
	if root == nil {
		return FragmentDisplayList{}
	}
	items := make([]FragmentDisplayItem, 0, 16)
	appendFragmentDisplayItems(&items, root, clipState{}, viewport)
	return FragmentDisplayList{items: items}
}

func fragmentNeedsFullDirtyPaint(fragment *Fragment) bool {
	if fragment == nil {
		return false
	}
	style := fragment.effectiveStyle()
	if fragment.Node != nil && documentNodeShowsDefaultFocusRing(fragment.Node) {
		return true
	}
	if resolveBorderRadius(style).Active() {
		return true
	}
	if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
		return true
	}
	if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
		return true
	}
	return false
}

func fragmentPaintBounds(fragment *Fragment) Rect {
	if fragment == nil {
		return Rect{}
	}
	style := fragment.effectiveStyle()
	includeTextShadow := fragment.Kind == FragmentKindText
	bounds := visualBoundsForStyle(fragment.Bounds, style, includeTextShadow)
	if fragment.Node != nil && documentNodeShowsDefaultFocusRing(fragment.Node) {
		bounds = UnionRect(bounds, focusRingBounds(fragment.Bounds))
	}
	return bounds
}

func appendFragmentDisplayItems(items *[]FragmentDisplayItem, fragment *Fragment, clip clipState, viewport Rect) {
	if fragment == nil {
		return
	}
	if styleVisible(fragment.effectiveStyle()) {
		paint := fragmentPaintBounds(fragment)
		if paint.Empty() {
			paint = fragment.Bounds
		}
		if clip.set {
			paint = IntersectRect(paint, clip.rect)
		}
		if !paint.Empty() {
			*items = append(*items, FragmentDisplayItem{
				Fragment: fragment,
				Bounds:   fragment.Bounds,
				Paint:    paint,
				Clip:     clip.rect,
				ClipSet:  clip.set,
			})
		}
	}
	childClip := clip
	if fragment.Kind == FragmentKindBlock {
		clipX, clipY := paintClipAxes(fragment.Style)
		if clipX || clipY {
			childClip = mergeFragmentClip(viewport, clip, fragment.Content, clipX, clipY)
		}
	}
	for _, child := range fragment.Children {
		appendFragmentDisplayItems(items, child, childClip, viewport)
	}
}

func mergeFragmentClip(viewport Rect, parent clipState, rect Rect, clipX bool, clipY bool) clipState {
	if !clipX && !clipY {
		return parent
	}
	base := viewport
	if base.Empty() {
		base = rect
	}
	if parent.set {
		base = parent.rect
	}
	if clipX {
		base.X = rect.X
		base.Width = rect.Width
	}
	if clipY {
		base.Y = rect.Y
		base.Height = rect.Height
	}
	if !viewport.Empty() {
		base = IntersectRect(base, viewport)
	}
	if parent.set {
		base = IntersectRect(base, parent.rect)
	}
	return clipState{rect: base, set: true}
}

func (fragment *Fragment) paintOffset(canvas *Canvas, offsetX int, offsetY int) {
	if fragment == nil || canvas == nil {
		return
	}
	style := fragment.effectiveStyle()
	if !styleVisible(style) {
		return
	}
	switch fragment.Kind {
	case FragmentKindText:
		fragment.paintTextOffset(canvas, offsetX, offsetY, style)
	default:
		bounds := fragment.Bounds
		if offsetX != 0 || offsetY != 0 {
			bounds.X += offsetX
			bounds.Y += offsetY
		}
		drawStyledBox(canvas, bounds, style, bounds, nil)
	}
	if fragment.Node != nil && documentNodeShowsDefaultFocusRing(fragment.Node) {
		bounds := fragment.Bounds
		if offsetX != 0 || offsetY != 0 {
			bounds.X += offsetX
			bounds.Y += offsetY
		}
		drawDefaultFocusRing(canvas, bounds, style)
	}
}

func (fragment *Fragment) paintTextOffset(canvas *Canvas, offsetX int, offsetY int, style Style) {
	if fragment == nil || canvas == nil || fragment.Bounds.Empty() || fragment.Text == "" {
		return
	}
	if FastNoText {
		return
	}
	lines := fragment.lines
	if len(lines) == 0 {
		return
	}
	foreground, ok := resolveColor(style.foreground)
	if !ok {
		foreground = Black
	}
	font := fragment.font
	charWidth := fragment.metrics.width
	lineHeight := fragment.lineHeight
	if lineHeight <= 0 {
		lineHeight = lineHeightForStyle(style, fragment.metrics.height)
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	leftPad, topPad, rightPad, availableW := textPaddingAndWidth(fragment.Bounds, style)
	shadow, shadowOK := resolveTextShadow(style.textShadow)
	if FastNoTextShadow || FastNoShadows {
		shadowOK = false
	}
	for i, line := range lines {
		if line.text == "" {
			continue
		}
		bounds := fragment.Bounds
		if offsetX != 0 || offsetY != 0 {
			bounds.X += offsetX
			bounds.Y += offsetY
		}
		x := textLineXForWidth(bounds, style, leftPad, rightPad, availableW, line.width)
		y := bounds.Y + topPad + i*lineHeight
		if shadowOK {
			if font != nil {
				canvas.DrawTextFont(x+shadow.OffsetX, y+shadow.OffsetY, shadow.Color, line.text, font)
			} else {
				canvas.DrawText(x+shadow.OffsetX, y+shadow.OffsetY, shadow.Color, line.text)
			}
		}
		if font != nil {
			canvas.DrawTextFont(x, y, foreground, line.text, font)
		} else {
			canvas.DrawText(x, y, foreground, line.text)
		}
		drawTextDecorations(canvas, x, y, line.text, style, font, charWidth, foreground)
	}
}
