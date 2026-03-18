package ui

func styleLayoutOnly(style Style) Style {
	return Style{
		display:  style.display,
		position: style.position,
		left:     style.left,
		top:      style.top,
		right:    style.right,
		bottom:   style.bottom,
		width:    style.width,
		height:   style.height,
		margin:   style.margin,
	}
}

func styleVisualOnly(style Style) Style {
	return Style{
		background:           style.background,
		foreground:           style.foreground,
		borderColor:          style.borderColor,
		borderWidth:          style.borderWidth,
		borderRadius:         style.borderRadius,
		gradient:             style.gradient,
		backgroundAttachment: style.backgroundAttachment,
		shadow:               style.shadow,
		textAlign:            style.textAlign,
		textShadow:           style.textShadow,
		fontPath:             style.fontPath,
		fontSize:             style.fontSize,
		padding:              style.padding,
		opacity:              style.opacity,
		overflow:             style.overflow,
		overflowX:            style.overflowX,
		overflowY:            style.overflowY,
		scrollbarWidth:       style.scrollbarWidth,
		scrollbarTrack:       style.scrollbarTrack,
		scrollbarThumb:       style.scrollbarThumb,
		scrollbarRadius:      style.scrollbarRadius,
		scrollbarPadding:     style.scrollbarPadding,
	}
}

func documentNodeInteractionStyle(node *DocumentNode) Style {
	if node == nil {
		return Style{}
	}
	style := Style{}
	if node.focused && !node.StyleFocus.IsZero() {
		style = mergeStyle(style, node.StyleFocus)
	}
	if node.hovered && !node.StyleHover.IsZero() {
		style = mergeStyle(style, node.StyleHover)
	}
	if node.active && !node.StyleActive.IsZero() {
		style = mergeStyle(style, node.StyleActive)
	}
	return style
}

func documentNodeLayoutStyle(node *DocumentNode) Style {
	return styleLayoutOnly(documentNodeInteractionStyle(node))
}

func documentNodePaintStyle(node *DocumentNode) Style {
	return styleVisualOnly(documentNodeInteractionStyle(node))
}

func documentNodeCanFocus(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	if node.Focusable {
		return true
	}
	if !node.StyleFocus.IsZero() {
		return true
	}
	return node.OnFocus != nil || node.OnBlur != nil || node.OnKeyDown != nil
}

func documentNodeInteractionNeedsLayout(style Style) bool {
	return styleLayoutOnly(style).HasLayout()
}

func (node *DocumentNode) setHover(hover bool) (bool, bool) {
	if node == nil || node.hovered == hover {
		return false, false
	}
	node.hovered = hover
	return true, documentNodeInteractionNeedsLayout(node.StyleHover)
}

func (node *DocumentNode) setActive(active bool) (bool, bool) {
	if node == nil || node.active == active {
		return false, false
	}
	node.active = active
	return true, documentNodeInteractionNeedsLayout(node.StyleActive)
}

func (node *DocumentNode) setFocus(focus bool) (bool, bool) {
	if node == nil || node.focused == focus {
		return false, false
	}
	node.focused = focus
	return true, documentNodeInteractionNeedsLayout(node.StyleFocus) || (documentNodeCanFocus(node) && node.StyleFocus.IsZero())
}

func (fragment *Fragment) effectiveStyle() Style {
	if fragment == nil {
		return Style{}
	}
	style := fragment.Style
	if fragment.Node != nil {
		style = mergeStyle(style, documentNodePaintStyle(fragment.Node))
	}
	return style
}

func dispatchDocumentNodeHandler(handler interface{}, node *DocumentNode, event DocumentEvent) bool {
	if handler == nil || node == nil {
		return false
	}
	switch current := handler.(type) {
	case func():
		current()
		return true
	case func(*DocumentNode):
		current(node)
		return true
	case func(DocumentEvent):
		current(event)
		return true
	default:
		return false
	}
}
