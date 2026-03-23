package ui

import "kos"

func initRangedElement(element *Element) {
	if element == nil {
		return
	}
	element.minValue = 0
	element.maxValue = 100
	element.stepValue = 1
}

func controlIndicatorSizeForLineHeight(lineHeight int) int {
	size := controlIndicatorMin
	if lineHeight > 0 && lineHeight-2 > size {
		size = lineHeight - 2
	}
	if size > 18 {
		size = 18
	}
	return size
}

func measureButtonWidth(element *Element, ctx ElementMeasureContext, baseWidth int) int {
	minWidth := ctx.TextWidth + defaultButtonWidthPadding + ctx.Insets.Left + ctx.Insets.Right
	if baseWidth < minWidth {
		baseWidth = minWidth
	}
	return baseWidth
}

func measureButtonHeight(element *Element, ctx ElementMeasureContext, baseHeight int) int {
	if baseHeight < defaultButtonHeight {
		baseHeight = defaultButtonHeight
	}
	return baseHeight
}

func measureCheckableWidth(element *Element, ctx ElementMeasureContext, baseWidth int) int {
	width := ctx.Insets.Left + ctx.Insets.Right + controlIndicatorSizeForLineHeight(ctx.LineHeight)
	if ctx.TextWidth > 0 {
		width += controlIndicatorGap + ctx.TextWidth
	}
	return width
}

func measureCheckableHeight(element *Element, ctx ElementMeasureContext, baseHeight int) int {
	indicatorHeight := controlIndicatorSizeForLineHeight(ctx.LineHeight) + ctx.Insets.Top + ctx.Insets.Bottom
	if baseHeight < indicatorHeight {
		baseHeight = indicatorHeight
	}
	return baseHeight
}

func measureProgressWidth(element *Element, ctx ElementMeasureContext, baseWidth int) int {
	if baseWidth < 180 {
		baseWidth = 180
	}
	return baseWidth
}

func measureProgressHeight(element *Element, ctx ElementMeasureContext, baseHeight int) int {
	if baseHeight < 18 {
		baseHeight = 18
	}
	return baseHeight
}

func measureRangeWidth(element *Element, ctx ElementMeasureContext, baseWidth int) int {
	if baseWidth < 180 {
		baseWidth = 180
	}
	return baseWidth
}

func measureRangeHeight(element *Element, ctx ElementMeasureContext, baseHeight int) int {
	if baseHeight < 24 {
		baseHeight = 24
	}
	return baseHeight
}

func drawButtonRaw(element *Element, style Style) bool {
	if element == nil {
		return false
	}
	x, y := element.rawPosition(style)
	background, ok := resolveColor(style.background)
	if !ok {
		background = Silver
	}
	width := element.resolvedWidth(style)
	height := element.resolvedHeight(style)
	kos.DrawButton(x, y, width, height, element.ID, background)
	element.drawRawTextRect(Rect{X: x, Y: y, Width: width, Height: height}, style)
	return true
}

func drawTextInputRaw(element *Element, style Style) bool {
	if element == nil {
		return false
	}
	x, y := element.rawPosition(style)
	rect := Rect{X: x, Y: y, Width: element.resolvedWidth(style), Height: element.resolvedHeight(style)}
	foreground, ok := resolveColor(style.foreground)
	if !ok {
		foreground = Black
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	selectionColor := defaultSelectionBackground
	element.drawEditableTextLines(layout, style,
		func(tx int, ty int, line string) {
			kos.DrawText(tx, ty, foreground, line)
			drawTextDecorationsRaw(tx, ty, line, style, layout.font, layout.charWidth, foreground)
		},
		func(cx int, cy int) {
			height := layout.lineHeight
			if height <= 0 {
				height = defaultFontHeight
			}
			kos.DrawBar(cx, cy, 1, height, uint32(foreground))
		},
		func(x int, y int, width int, height int) {
			if width <= 0 || height <= 0 {
				return
			}
			kos.DrawBar(x, y, width, height, uint32(selectionColor))
		},
	)
	return true
}

func drawTinyGLRaw(element *Element, style Style) bool {
	return true
}

func drawElementViaSurfaceRaw(element *Element, style Style) bool {
	if element == nil {
		return false
	}
	x, y := element.rawPosition(style)
	width := element.resolvedWidth(style)
	height := element.resolvedHeight(style)
	if width <= 0 || height <= 0 {
		return true
	}
	canvas := NewCanvasAlpha(width, height)
	element.drawToRect(canvas, Rect{X: 0, Y: 0, Width: width, Height: height}, style)
	canvas.BlitToWindow(x, y)
	return true
}

func paintTextInputElement(element *Element, canvas *Canvas, rect Rect, style Style) bool {
	if element == nil || canvas == nil {
		return false
	}
	if FastNoText {
		if elementShowsDefaultFocusRing(element) {
			drawDefaultFocusRing(canvas, rect, style)
		}
		return true
	}
	foreground, ok := resolveColor(style.foreground)
	if !ok {
		foreground = Black
	}
	font := fontForStyle(style)
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	selectionColor := defaultSelectionBackground
	canvas.PushClip(layout.content)
	element.drawEditableTextLines(layout, style,
		func(x int, y int, text string) {
			if font != nil {
				canvas.DrawTextFont(x, y, foreground, text, font)
			} else {
				canvas.DrawText(x, y, foreground, text)
			}
			drawTextDecorations(canvas, x, y, text, style, font, layout.charWidth, foreground)
		},
		func(x int, y int) {
			height := layout.lineHeight
			if height <= 0 {
				height = defaultFontHeight
			}
			canvas.FillRect(x, y, 1, height, foreground)
		},
		func(x int, y int, width int, height int) {
			if width <= 0 || height <= 0 {
				return
			}
			canvas.FillRect(x, y, width, height, selectionColor)
		},
	)
	canvas.PopClip()
	element.drawInputScrollbars(canvas, layout)
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
	return true
}

func paintTinyGLElement(element *Element, canvas *Canvas, rect Rect, style Style) bool {
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
	return true
}

func paintCheckboxElement(element *Element, canvas *Canvas, rect Rect, style Style) bool {
	if element == nil || canvas == nil {
		return false
	}
	element.drawCheckboxOrRadio(canvas, rect, style)
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
	return true
}

func paintProgressElement(element *Element, canvas *Canvas, rect Rect, style Style) bool {
	if element == nil || canvas == nil {
		return false
	}
	element.drawProgress(canvas, rect, style)
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
	return true
}

func paintRangeElement(element *Element, canvas *Canvas, rect Rect, style Style) bool {
	if element == nil || canvas == nil {
		return false
	}
	element.drawRange(canvas, rect, style)
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
	return true
}

func handleTextInputClick(element *Element, event *Event) bool {
	if element == nil || event == nil {
		return false
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	if element.handleScrollbarClick(event.X, event.Y, rect, style) {
		return true
	}
	return element.setCaretFromPoint(event.X, event.Y, rect, style)
}

func handleTextInputMouseMove(element *Element, x int, y int, buttons PointerButtons) bool {
	return element.handleTextMouseDrag(x, y)
}

func handleTextInputMouseDown(element *Element, x int, y int, button MouseButton, buttons PointerButtons) bool {
	return element.handleTextMouseDown(x, y)
}

func handleTextInputMouseUp(element *Element, x int, y int, button MouseButton, buttons PointerButtons) bool {
	return element.handleTextMouseUp()
}

func handleTextInputKey(element *Element, key kos.KeyEvent) bool {
	return element.handleTextInputKeyAction(key)
}

func handleTextInputScroll(element *Element, deltaX int, deltaY int) bool {
	return element.handleTextInputScrollAction(deltaX, deltaY)
}

func handleCheckableClick(element *Element, event *Event) bool {
	if element == nil {
		return false
	}
	if element.ToggleChecked() {
		element.dispatchInputEvent()
		element.dispatchChange()
		return true
	}
	return false
}

func handleRangeClick(element *Element, event *Event) bool {
	if element == nil {
		return false
	}
	return element.handleControlClick(event)
}

func handleRangeMouseDownSpec(element *Element, x int, y int, button MouseButton, buttons PointerButtons) bool {
	return element.handleRangeMouseDown(x, y)
}

func handleRangeMouseMoveSpec(element *Element, x int, y int, buttons PointerButtons) bool {
	return element.handleRangeMouseDrag(x, y)
}

func handleRangeMouseUpSpec(element *Element, x int, y int, button MouseButton, buttons PointerButtons) bool {
	return element.handleRangeMouseUp()
}

func handleControlKeySpec(element *Element, key kos.KeyEvent) bool {
	return element.handleControlKey(key)
}
