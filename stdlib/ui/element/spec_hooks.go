package ui

import "kos"

func (element *Element) specInit() {
	if element == nil {
		return
	}
	if spec := element.Spec(); spec != nil {
		if init := spec.initFunc(); init != nil {
			init(element)
		}
	}
}

func (element *Element) measureContext(ctx LayoutContext, container Rect, style Style, text string, font *ttfFont, charWidth int, lineHeight int, textWidth int, textHeight int) ElementMeasureContext {
	return ElementMeasureContext{
		Layout:     ctx,
		Container:  container,
		Style:      style,
		Text:       text,
		Insets:     boxInsets(style),
		Font:       font,
		CharWidth:  charWidth,
		LineHeight: lineHeight,
		TextWidth:  textWidth,
		TextHeight: textHeight,
	}
}

func (element *Element) measureWidthWithSpec(ctx LayoutContext, container Rect, style Style, text string, font *ttfFont, charWidth int, lineHeight int, textWidth int, baseWidth int) int {
	if element == nil {
		return baseWidth
	}
	if spec := element.Spec(); spec != nil {
		if measure := spec.measureWidthFunc(); measure != nil {
			return measure(element, element.measureContext(ctx, container, style, text, font, charWidth, lineHeight, textWidth, 0), baseWidth)
		}
	}
	return baseWidth
}

func (element *Element) measureHeightWithSpec(ctx LayoutContext, container Rect, style Style, text string, font *ttfFont, charWidth int, lineHeight int, textWidth int, textHeight int, baseHeight int) int {
	if element == nil {
		return baseHeight
	}
	if spec := element.Spec(); spec != nil {
		if measure := spec.measureHeightFunc(); measure != nil {
			return measure(element, element.measureContext(ctx, container, style, text, font, charWidth, lineHeight, textWidth, textHeight), baseHeight)
		}
	}
	return baseHeight
}

func (element *Element) drawRawWithSpec(style Style) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if draw := spec.drawRawFunc(); draw != nil {
			return draw(element, style)
		}
	}
	return false
}

func (element *Element) paintWithSpec(canvas *Canvas, rect Rect, style Style) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if paint := spec.paintFunc(); paint != nil {
			return paint(element, canvas, rect, style)
		}
	}
	return false
}

func (element *Element) handleClickWithSpec(event *Event) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if handle := spec.handleClickFunc(); handle != nil {
			return handle(element, event)
		}
	}
	return false
}

func (element *Element) handleMouseMoveWithSpec(x int, y int, buttons PointerButtons) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if handle := spec.handleMouseMoveFunc(); handle != nil {
			return handle(element, x, y, buttons)
		}
	}
	return false
}

func (element *Element) handleMouseDownWithSpec(x int, y int, button MouseButton, buttons PointerButtons) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if handle := spec.handleMouseDownFunc(); handle != nil {
			return handle(element, x, y, button, buttons)
		}
	}
	return false
}

func (element *Element) handleMouseUpWithSpec(x int, y int, button MouseButton, buttons PointerButtons) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if handle := spec.handleMouseUpFunc(); handle != nil {
			return handle(element, x, y, button, buttons)
		}
	}
	return false
}

func (element *Element) handleKeyWithSpec(key kos.KeyEvent) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if handle := spec.handleKeyFunc(); handle != nil {
			return handle(element, key)
		}
	}
	return false
}

func (element *Element) handleScrollWithSpec(deltaX int, deltaY int) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		if handle := spec.handleScrollFunc(); handle != nil {
			return handle(element, deltaX, deltaY)
		}
	}
	return false
}
