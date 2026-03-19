package ui

func styleLayoutOnly(style Style) Style {
	return Style{
		display:           style.display,
		position:          style.position,
		left:              style.left,
		top:               style.top,
		right:             style.right,
		bottom:            style.bottom,
		width:             style.width,
		height:            style.height,
		minWidth:          style.minWidth,
		maxWidth:          style.maxWidth,
		minHeight:         style.minHeight,
		maxHeight:         style.maxHeight,
		margin:            style.margin,
		padding:           style.padding,
		borderWidth:       style.borderWidth,
		borderTopWidth:    style.borderTopWidth,
		borderRightWidth:  style.borderRightWidth,
		borderBottomWidth: style.borderBottomWidth,
		borderLeftWidth:   style.borderLeftWidth,
		fontPath:          style.fontPath,
		fontSize:          style.fontSize,
		lineHeight:        style.lineHeight,
		whiteSpace:        style.whiteSpace,
		overflowWrap:      style.overflowWrap,
		wordBreak:         style.wordBreak,
		boxSizing:         style.boxSizing,
	}
}

func styleVisualOnly(style Style) Style {
	return Style{
		background:           style.background,
		foreground:           style.foreground,
		borderColor:          style.borderColor,
		borderWidth:          style.borderWidth,
		borderTopColor:       style.borderTopColor,
		borderRightColor:     style.borderRightColor,
		borderBottomColor:    style.borderBottomColor,
		borderLeftColor:      style.borderLeftColor,
		borderTopWidth:       style.borderTopWidth,
		borderRightWidth:     style.borderRightWidth,
		borderBottomWidth:    style.borderBottomWidth,
		borderLeftWidth:      style.borderLeftWidth,
		borderRadius:         style.borderRadius,
		gradient:             style.gradient,
		backgroundAttachment: style.backgroundAttachment,
		shadow:               style.shadow,
		visibility:           style.visibility,
		textAlign:            style.textAlign,
		textDecoration:       style.textDecoration,
		whiteSpace:           style.whiteSpace,
		overflowWrap:         style.overflowWrap,
		wordBreak:            style.wordBreak,
		textShadow:           style.textShadow,
		fontPath:             style.fontPath,
		fontSize:             style.fontSize,
		lineHeight:           style.lineHeight,
		padding:              style.padding,
		opacity:              style.opacity,
		outlineColor:         style.outlineColor,
		outlineWidth:         style.outlineWidth,
		outlineOffset:        style.outlineOffset,
		outlineRadius:        style.outlineRadius,
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
	if !fragment.PaintStyle.IsZero() {
		return fragment.PaintStyle
	}
	return fragment.Style
}

func dispatchDocumentNodeHandler(handler interface{}, node *DocumentNode, event *DocumentEvent) bool {
	if handler == nil || node == nil {
		return false
	}
	if event != nil {
		if event.Node == nil {
			event.Node = node
		}
		if event.CurrentTarget == nil {
			event.CurrentTarget = node
		}
		if event.Phase == EventPhaseNone {
			event.Phase = EventPhaseTarget
		}
	}
	switch current := handler.(type) {
	case func():
		current()
		return true
	case func(*DocumentNode):
		current(node)
		return true
	case func(DocumentEvent):
		if event == nil {
			current(DocumentEvent{})
		} else {
			current(*event)
		}
		return true
	case func(*DocumentEvent):
		current(event)
		return true
	case func(*DocumentNode, DocumentEvent):
		if event == nil {
			current(node, DocumentEvent{})
		} else {
			current(node, *event)
		}
		return true
	case func(*DocumentNode, *DocumentEvent):
		current(node, event)
		return true
	default:
		return false
	}
}

func documentEventPath(node *DocumentNode) []*DocumentNode {
	if node == nil {
		return nil
	}
	path := make([]*DocumentNode, 0, 4)
	for current := node; current != nil; current = current.Parent {
		path = append(path, current)
	}
	return path
}

func dispatchDocumentNodeEvent(event *DocumentEvent, path []*DocumentNode, handler func(*DocumentNode) interface{}) bool {
	if event == nil || len(path) == 0 || handler == nil {
		return false
	}
	handled := false
	for index, current := range path {
		if current == nil {
			continue
		}
		if index > 0 && !event.Bubbles {
			break
		}
		event.CurrentTarget = current
		if index == 0 {
			event.Phase = EventPhaseTarget
		} else {
			event.Phase = EventPhaseBubble
		}
		if dispatchDocumentNodeHandler(handler(current), current, event) {
			handled = true
		}
		if event.PropagationStopped() {
			break
		}
	}
	return handled
}
