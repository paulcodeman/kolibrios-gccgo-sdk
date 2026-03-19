package ui

import (
	"bytes"
	"kos"
)

func (element *Element) isTextInput() bool {
	if element == nil {
		return false
	}
	return element.kind == ElementKindInput || element.kind == ElementKindTextarea
}

func (element *Element) editLines(content Rect, style Style, font *ttfFont, charWidth int) []textLine {
	text := element.text()
	maxWidth := 0
	if content.Width > 0 {
		maxWidth = content.Width
	}
	wrap := element.kind == ElementKindTextarea
	if element.kind == ElementKindTextarea {
		overflowX := overflowModeFor(style, "x")
		if overflowX == OverflowScroll || overflowX == OverflowAuto {
			wrap = false
		}
	}
	return element.wrapTextPreserveCached(text, maxWidth, wrap, font, charWidth)
}

func maxLineLength(lines []textLine) int {
	maxLen := 0
	for _, line := range lines {
		if line.columns > maxLen {
			maxLen = line.columns
		}
	}
	return maxLen
}

const defaultSelectionBackground = kos.Color(0xCCE8FF)

type scrollbarStyle struct {
	width   int
	track   kos.Color
	thumb   kos.Color
	radius  int
	padding Spacing
}

type textInputLayout struct {
	baseContent Rect
	content     Rect
	lines       []textLine
	linesCached bool
	totalHeight int
	totalWidth  int
	lineHeight  int
	charWidth   int
	font        *ttfFont
	showV       bool
	scrollbar   scrollbarStyle
}

func (layout *textInputLayout) release() {
	if layout == nil {
		return
	}
	if !layout.linesCached {
		releaseTextLines(layout.lines)
	}
	layout.lines = nil
	layout.linesCached = false
}

func overflowModeFor(style Style, axis string) OverflowMode {
	switch axis {
	case "x":
		if value, ok := resolveOverflow(style.overflowX); ok {
			return value
		}
	case "y":
		if value, ok := resolveOverflow(style.overflowY); ok {
			return value
		}
	}
	if value, ok := resolveOverflow(style.overflow); ok {
		return value
	}
	return OverflowVisible
}

func resolveScrollbarStyle(style Style) scrollbarStyle {
	width, ok := resolveScrollbarWidth(style.scrollbarWidth)
	if !ok {
		width = defaultScrollbarWidth
	}
	track, ok := resolveColor(style.scrollbarTrack)
	if !ok {
		track = Silver
	}
	thumb, ok := resolveColor(style.scrollbarThumb)
	if !ok {
		thumb = Gray
	}
	radius, ok := resolveScrollbarRadius(style.scrollbarRadius)
	if !ok {
		radius = width / 2
	}
	if radius < 0 {
		radius = 0
	}
	padding := Spacing{}
	if value, ok := resolveSpacingNormalized(style.scrollbarPadding); ok {
		padding = value
	}
	return scrollbarStyle{
		width:   width,
		track:   track,
		thumb:   thumb,
		radius:  radius,
		padding: padding,
	}
}

func (element *Element) textInputLayout(rect Rect, style Style) textInputLayout {
	layout := textInputLayout{}
	if element == nil || !element.isTextInput() {
		return layout
	}
	base := contentRectFor(rect, style)
	layout.baseContent = base
	layout.content = base
	layout.scrollbar = resolveScrollbarStyle(style)
	font, metrics := fontAndMetricsForStyle(style)
	layout.lineHeight = lineHeightForStyle(style, metrics.height)
	layout.charWidth = metrics.width
	layout.font = font
	if layout.charWidth <= 0 {
		layout.charWidth = defaultCharWidth
	}

	lines := element.editLines(base, style, font, layout.charWidth)
	linesCached := true
	if len(lines) == 0 {
		lines = getTextLineSlice(1)
		lines = append(lines, textLine{text: "", start: 0, end: 0})
		linesCached = false
	}
	ensureTextLineMetrics(lines, font, layout.charWidth)

	if element.kind == ElementKindTextarea {
		totalHeight := len(lines) * layout.lineHeight
		overflowY := overflowModeFor(style, "y")
		showV := false
		minWidth := layout.scrollbar.width + layout.scrollbar.padding.Left + layout.scrollbar.padding.Right
		if layout.scrollbar.width > 0 && base.Width > minWidth && base.Height > 0 {
			switch overflowY {
			case OverflowScroll:
				showV = true
			case OverflowAuto:
				showV = totalHeight > base.Height
			}
		}
		if showV {
			content := base
			gapLeft := layout.scrollbar.padding.Left
			gapRight := layout.scrollbar.padding.Right
			content.Width -= layout.scrollbar.width + gapLeft + gapRight
			if content.Width < 0 {
				content.Width = 0
			}
			layout.content = content
			if !linesCached {
				releaseTextLines(lines)
			}
			lines = element.editLines(content, style, font, layout.charWidth)
			linesCached = true
			if len(lines) == 0 {
				lines = getTextLineSlice(1)
				lines = append(lines, textLine{text: "", start: 0, end: 0})
				linesCached = false
			}
			ensureTextLineMetrics(lines, font, layout.charWidth)
			totalHeight = len(lines) * layout.lineHeight
		}
		layout.lines = lines
		layout.linesCached = linesCached
		layout.totalHeight = totalHeight
		layout.showV = showV
		return layout
	}

	layout.lines = lines
	layout.linesCached = linesCached
	layout.totalWidth = maxLineWidth(lines, font, layout.charWidth)
	return layout
}

func (element *Element) verticalScrollbarLayout(layout textInputLayout) (Rect, Rect, int, bool) {
	if element == nil || !layout.showV || layout.scrollbar.width <= 0 {
		return Rect{}, Rect{}, 0, false
	}
	base := layout.baseContent
	pad := layout.scrollbar.padding
	track := Rect{
		X:      base.X + base.Width - layout.scrollbar.width - pad.Right,
		Y:      base.Y + pad.Top,
		Width:  layout.scrollbar.width,
		Height: base.Height - pad.Top - pad.Bottom,
	}
	if track.Width <= 0 || track.Height <= 0 {
		return track, Rect{}, 0, false
	}
	viewHeight := layout.content.Height
	totalHeight := layout.totalHeight
	maxScroll := 0
	if viewHeight > 0 && totalHeight > viewHeight {
		maxScroll = totalHeight - viewHeight
	}
	thumbHeight := track.Height
	if totalHeight > 0 && viewHeight > 0 && totalHeight > viewHeight {
		thumbHeight = track.Height * viewHeight / totalHeight
		minThumb := layout.scrollbar.width * 2
		if minThumb < defaultScrollbarMinThumb {
			minThumb = defaultScrollbarMinThumb
		}
		if thumbHeight < minThumb {
			thumbHeight = minThumb
		}
		if thumbHeight > track.Height {
			thumbHeight = track.Height
		}
	}
	thumbY := track.Y
	if maxScroll > 0 && track.Height > thumbHeight {
		offsetRange := track.Height - thumbHeight
		thumbY = track.Y + element.scrollY*offsetRange/maxScroll
	}
	thumb := Rect{
		X:      track.X,
		Y:      thumbY,
		Width:  track.Width,
		Height: thumbHeight,
	}
	return track, thumb, maxScroll, true
}

func (element *Element) caretLineAndColumn(lines []textLine) (int, int) {
	if element == nil || len(lines) == 0 {
		return 0, 0
	}
	pos := element.caret
	for i, line := range lines {
		if pos <= line.end {
			col := pos - line.start
			if col < 0 {
				col = 0
			}
			return i, textColumnForByteIndex(line.text, col)
		}
	}
	last := lines[len(lines)-1]
	return len(lines) - 1, last.columns
}

func caretIndexForLineColumn(lines []textLine, line int, col int) int {
	if len(lines) == 0 {
		return 0
	}
	if line < 0 {
		line = 0
	} else if line >= len(lines) {
		line = len(lines) - 1
	}
	target := lines[line]
	if col < 0 {
		col = 0
	}
	maxCol := target.columns
	if col > maxCol {
		col = maxCol
	}
	return target.start + textByteIndexForColumn(target.text, col)
}

func (element *Element) setCaret(pos int) bool {
	if element == nil {
		return false
	}
	text := element.text()
	textLen := len(text)
	if pos < 0 {
		pos = 0
	} else if pos > textLen {
		pos = textLen
	}
	pos = textClampIndexToRuneBoundary(text, pos)
	if element.caret == pos {
		return false
	}
	element.caret = pos
	element.desiredCol = -1
	element.markDirty()
	return true
}

func (element *Element) hasSelection() bool {
	if element == nil {
		return false
	}
	return element.selectAnchor != element.caret
}

func (element *Element) selectionRange() (int, int, bool) {
	if element == nil || !element.hasSelection() {
		return 0, 0, false
	}
	start := element.selectAnchor
	end := element.caret
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	textLen := len(element.text())
	if end > textLen {
		end = textLen
	}
	if start >= end {
		return 0, 0, false
	}
	return start, end, true
}

func (element *Element) clearSelection() bool {
	if element == nil || !element.hasSelection() {
		return false
	}
	element.selectAnchor = element.caret
	element.markDirty()
	return true
}

func (element *Element) ensureCaretVisible(rect Rect, style Style) bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	if layout.content.Empty() {
		return false
	}
	overflowX := overflowModeFor(style, "x")
	return element.ensureCaretVisibleWithLines(layout.content, layout.lines, overflowX, layout.font, layout.charWidth, layout.lineHeight)
}

func (element *Element) ensureCaretVisibleWithLines(content Rect, lines []textLine, overflowX OverflowMode, font *ttfFont, charWidth int, lineHeight int) bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if lineHeight <= 0 {
		lineHeight = defaultFontHeight
	}
	if len(lines) == 0 {
		lines = []textLine{{text: "", start: 0, end: 0}}
	}
	ensureTextLineMetrics(lines, font, charWidth)
	hScroll := element.kind == ElementKindInput ||
		(element.kind == ElementKindTextarea && (overflowX == OverflowScroll || overflowX == OverflowAuto))
	changed := false
	text := element.text()
	textLen := len(text)
	if element.caret < 0 {
		element.caret = 0
		changed = true
	} else if element.caret > textLen {
		element.caret = textLen
		changed = true
	}
	if clamped := textClampIndexToRuneBoundary(text, element.caret); clamped != element.caret {
		element.caret = clamped
		changed = true
	}
	if element.kind == ElementKindInput {
		textWidth := textWidthWithFont(text, font, charWidth)
		maxScroll := 0
		if content.Width > 0 && textWidth > content.Width {
			maxScroll = textWidth - content.Width
		}
		caretCol := textColumnForByteIndex(text, element.caret)
		caretX := textWidthForColumns(text, caretCol, font, charWidth)
		if caretX < element.scrollX {
			element.scrollX = caretX
			changed = true
		} else if content.Width > 0 && caretX > element.scrollX+content.Width-charWidth {
			element.scrollX = caretX - (content.Width - charWidth)
			changed = true
		}
		if element.scrollX < 0 {
			element.scrollX = 0
			changed = true
		} else if element.scrollX > maxScroll {
			element.scrollX = maxScroll
			changed = true
		}
		if element.scrollY != 0 {
			element.scrollY = 0
			changed = true
		}
		return changed
	}
	line, col := element.caretLineAndColumn(lines)
	caretY := line * lineHeight
	totalHeight := len(lines) * lineHeight
	maxScroll := 0
	if content.Height > 0 && totalHeight > content.Height {
		maxScroll = totalHeight - content.Height
	}
	if caretY < element.scrollY {
		element.scrollY = caretY
		changed = true
	} else if content.Height > 0 && caretY > element.scrollY+content.Height-lineHeight {
		element.scrollY = caretY - (content.Height - lineHeight)
		changed = true
	}
	if element.scrollY < 0 {
		element.scrollY = 0
		changed = true
	} else if element.scrollY > maxScroll {
		element.scrollY = maxScroll
		changed = true
	}
	if hScroll {
		textWidth := maxLineWidth(lines, font, charWidth)
		maxScrollX := 0
		if content.Width > 0 && textWidth > content.Width {
			maxScrollX = textWidth - content.Width
		}
		lineText := ""
		if line >= 0 && line < len(lines) {
			lineText = lines[line].text
		}
		caretX := textWidthForColumns(lineText, col, font, charWidth)
		if caretX < element.scrollX {
			element.scrollX = caretX
			changed = true
		} else if content.Width > 0 && caretX > element.scrollX+content.Width-charWidth {
			element.scrollX = caretX - (content.Width - charWidth)
			changed = true
		}
		if element.scrollX < 0 {
			element.scrollX = 0
			changed = true
		} else if element.scrollX > maxScrollX {
			element.scrollX = maxScrollX
			changed = true
		}
	} else if element.scrollX != 0 {
		element.scrollX = 0
		changed = true
	}
	return changed
}

func (element *Element) setCaretFromPoint(x int, y int, rect Rect, style Style) bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	content := layout.content
	if content.Empty() {
		return false
	}
	lines := layout.lines
	if len(lines) == 0 {
		lines = []textLine{{text: "", start: 0, end: 0}}
	}
	if element.kind == ElementKindTextarea && layout.showV {
		if x >= content.X+content.Width {
			return false
		}
	}
	overflowX := overflowModeFor(style, "x")
	hScroll := element.kind == ElementKindInput ||
		(element.kind == ElementKindTextarea && (overflowX == OverflowScroll || overflowX == OverflowAuto))
	if element.kind == ElementKindInput {
		col := textColumnForX(element.text(), x-content.X+element.scrollX, layout.font, layout.charWidth)
		caret := textByteIndexForColumn(element.text(), col)
		changed := element.setCaret(caret)
		element.ensureCaretVisibleWithLines(content, lines, overflowX, layout.font, layout.charWidth, layout.lineHeight)
		return changed
	}
	line := 0
	if y > content.Y {
		line = (y - content.Y + element.scrollY) / layout.lineHeight
	}
	if line < 0 {
		line = 0
	} else if line >= len(lines) {
		line = len(lines) - 1
	}
	lineText := ""
	if line >= 0 && line < len(lines) {
		lineText = lines[line].text
	}
	colX := x - content.X
	if hScroll {
		colX += element.scrollX
	}
	col := textColumnForX(lineText, colX, layout.font, layout.charWidth)
	if col < 0 {
		col = 0
	}
	caret := caretIndexForLineColumn(lines, line, col)
	changed := element.setCaret(caret)
	element.ensureCaretVisibleWithLines(content, lines, overflowX, layout.font, layout.charWidth, layout.lineHeight)
	return changed
}

func (element *Element) drawEditableTextLines(layout textInputLayout, style Style, drawLine func(x int, y int, text string), drawCaret func(x int, y int), drawSelection func(x int, y int, width int, height int)) {
	if element == nil || drawLine == nil {
		return
	}
	content := layout.content
	if content.Empty() {
		return
	}
	lines := layout.lines
	if len(lines) == 0 {
		lines = []textLine{{text: "", start: 0, end: 0}}
	}
	scrollX := element.scrollX
	scrollY := element.scrollY
	lineHeight := layout.lineHeight
	charWidth := layout.charWidth
	font := layout.font
	startLine := 0
	y := content.Y
	if element.kind == ElementKindTextarea && scrollY > 0 {
		startLine = scrollY / lineHeight
		y -= scrollY % lineHeight
	}
	hasHClip := content.Width > 0
	visibleLeft := 0
	visibleRight := 0
	if hasHClip {
		visibleLeft = scrollX
		visibleRight = scrollX + content.Width
	}
	selectionStart, selectionEnd, hasSelection := element.selectionRange()
	for i := startLine; i < len(lines); i++ {
		if y >= content.Y+content.Height {
			break
		}
		if y+lineHeight <= content.Y {
			y += lineHeight
			continue
		}
		lineText := lines[i].text
		lineCols := lines[i].columns
		if hasSelection && drawSelection != nil {
			line := lines[i]
			lineStart := line.start
			lineEnd := line.end
			if selectionStart < lineEnd && selectionEnd > lineStart {
				selStart := selectionStart
				if selStart < lineStart {
					selStart = lineStart
				}
				selEnd := selectionEnd
				if selEnd > lineEnd {
					selEnd = lineEnd
				}
				if selStart < selEnd {
					colStart := textColumnForByteIndex(lineText, selStart-lineStart)
					colEnd := textColumnForByteIndex(lineText, selEnd-lineStart)
					xStart := textWidthForColumns(lineText, colStart, font, charWidth)
					xEnd := textWidthForColumns(lineText, colEnd, font, charWidth)
					drawStart := xStart
					drawEnd := xEnd
					if hasHClip {
						if drawEnd <= visibleLeft || drawStart >= visibleRight {
							drawStart = drawEnd
						} else {
							if drawStart < visibleLeft {
								drawStart = visibleLeft
							}
							if drawEnd > visibleRight {
								drawEnd = visibleRight
							}
						}
					}
					if drawStart < drawEnd {
						x := content.X + drawStart - scrollX
						width := drawEnd - drawStart
						drawSelection(x, y, width, lineHeight)
					}
				}
			}
		}
		if lineText != "" {
			startCol := 0
			endCol := lineCols
			drawX := content.X - scrollX
			if hasHClip {
				startCol = textColumnForX(lineText, visibleLeft, font, charWidth)
				endCol = textColumnForX(lineText, visibleRight, font, charWidth)
				if endCol < startCol {
					endCol = startCol
				}
				drawX = content.X + textWidthForColumns(lineText, startCol, font, charWidth) - scrollX
			}
			if startCol < endCol {
				drawLine(drawX, y, textSliceColumns(lineText, startCol, endCol))
			}
		}
		y += lineHeight
	}
	if drawCaret != nil && element.caretVisible() {
		line, col := element.caretLineAndColumn(lines)
		lineText := ""
		if line >= 0 && line < len(lines) {
			lineText = lines[line].text
		}
		caretX := content.X + textWidthForColumns(lineText, col, font, charWidth) - scrollX
		caretY := content.Y + line*lineHeight - scrollY
		if caretX >= content.X && caretX < content.X+content.Width &&
			caretY < content.Y+content.Height && caretY+lineHeight > content.Y {
			drawCaret(caretX, caretY)
		}
	}
}

func (element *Element) caretRectFromLayout(layout textInputLayout) Rect {
	if element == nil || !element.isTextInput() {
		return Rect{}
	}
	content := layout.content
	if content.Empty() {
		return Rect{}
	}
	lines := layout.lines
	if len(lines) == 0 {
		lines = []textLine{{text: "", start: 0, end: 0}}
	}
	charWidth := layout.charWidth
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	ensureTextLineMetrics(lines, layout.font, charWidth)
	lineHeight := layout.lineHeight
	if lineHeight <= 0 {
		lineHeight = defaultFontHeight
	}
	line, col := element.caretLineAndColumn(lines)
	lineText := ""
	if line >= 0 && line < len(lines) {
		lineText = lines[line].text
	}
	caretX := content.X + textWidthForColumns(lineText, col, layout.font, charWidth) - element.scrollX
	caretY := content.Y + line*lineHeight - element.scrollY
	caret := Rect{
		X:      caretX,
		Y:      caretY,
		Width:  1,
		Height: lineHeight,
	}
	return IntersectRect(caret, content)
}

func (element *Element) caretDirtyRect(rect Rect, style Style) Rect {
	if element == nil || !element.isTextInput() {
		return Rect{}
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	return element.caretRectFromLayout(layout)
}

func (element *Element) textInputDirtyClipRect(style Style) Rect {
	if element == nil || !element.isTextInput() {
		return Rect{}
	}
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	return layout.content
}

func (element *Element) caretVisible() bool {
	if element == nil || !element.focused {
		return false
	}
	if element.hasSelection() {
		return false
	}
	if element.window == nil {
		return true
	}
	return element.window.caretBlinkVisible()
}

func (element *Element) drawInputScrollbars(canvas *Canvas, layout textInputLayout) {
	if element == nil || canvas == nil || !element.isTextInput() {
		return
	}
	if element.kind != ElementKindTextarea {
		return
	}
	track, thumb, _, ok := element.verticalScrollbarLayout(layout)
	if !ok {
		return
	}
	radius := layout.scrollbar.radius
	radii := CornerRadii{}
	if radius > 0 {
		radii = CornerRadii{
			TopLeft:     radius,
			TopRight:    radius,
			BottomRight: radius,
			BottomLeft:  radius,
		}
	}
	canvas.FillRoundedRect(track.X, track.Y, track.Width, track.Height, radii, layout.scrollbar.track)
	if thumb.Height > 0 && thumb.Width > 0 {
		canvas.FillRoundedRect(thumb.X, thumb.Y, thumb.Width, thumb.Height, radii, layout.scrollbar.thumb)
	}
}

func (element *Element) handleScrollbarClick(x int, y int, rect Rect, style Style) bool {
	if element == nil || element.kind != ElementKindTextarea {
		return false
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	track, thumb, maxScroll, ok := element.verticalScrollbarLayout(layout)
	if !ok || !track.Contains(x, y) {
		return false
	}
	if maxScroll <= 0 {
		return true
	}
	thumbHeight := thumb.Height
	trackRange := track.Height - thumbHeight
	if trackRange <= 0 {
		if element.scrollY != 0 {
			element.scrollY = 0
			element.markDirty()
		}
		return true
	}
	target := y - track.Y - thumbHeight/2
	if target < 0 {
		target = 0
	} else if target > trackRange {
		target = trackRange
	}
	next := target * maxScroll / trackRange
	if next != element.scrollY {
		element.scrollY = next
		element.markDirty()
	}
	return true
}

func (element *Element) handleTextMouseDown(x int, y int) bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	element.dragMode = textDragNone
	element.dragMoved = false

	if element.kind == ElementKindTextarea && layout.showV {
		track, thumb, maxScroll, ok := element.verticalScrollbarLayout(layout)
		if ok && track.Contains(x, y) {
			element.dragMoved = true
			if thumb.Contains(x, y) {
				element.dragMode = textDragScroll
				element.dragScrollOffset = y - thumb.Y
				return false
			}
			if maxScroll > 0 {
				if element.handleScrollbarClick(x, y, rect, style) {
					element.dragMoved = true
					return true
				}
			}
			return false
		}
	}

	shiftPressed := kos.ControlKeysStatus().Shift()
	hadSelection := element.hasSelection()
	anchor := element.selectAnchor
	if !hadSelection {
		anchor = element.caret
	}
	changed := element.setCaretFromPoint(x, y, rect, style)
	if shiftPressed {
		if !hadSelection {
			element.selectAnchor = anchor
		}
	} else {
		element.selectAnchor = element.caret
	}
	if !shiftPressed && hadSelection && !changed {
		element.markDirty()
	}
	if changed {
		element.dragMoved = true
	}
	element.dragMode = textDragSelect
	return changed
}

func (element *Element) handleTextMouseDrag(x int, y int) bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	switch element.dragMode {
	case textDragScroll:
		layout := element.textInputLayout(rect, style)
		defer layout.release()
		track, thumb, maxScroll, ok := element.verticalScrollbarLayout(layout)
		if !ok || maxScroll <= 0 {
			return false
		}
		trackRange := track.Height - thumb.Height
		if trackRange <= 0 {
			return false
		}
		target := y - element.dragScrollOffset - track.Y
		if target < 0 {
			target = 0
		} else if target > trackRange {
			target = trackRange
		}
		next := target * maxScroll / trackRange
		if next != element.scrollY {
			element.scrollY = next
			element.markDirty()
			element.dragMoved = true
			return true
		}
		return false
	case textDragSelect:
		changed := element.setCaretFromPoint(x, y, rect, style)
		if changed {
			element.dragMoved = true
		}
		return changed
	}
	return false
}

func (element *Element) handleTextMouseUp() bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	if element.dragMode != textDragNone {
		element.dragMode = textDragNone
		return false
	}
	return false
}

func (element *Element) Handle(event Event) bool {
	if event.Type != EventClick {
		return false
	}
	handled := false
	if element.isTextInput() {
		style := element.effectiveStyle()
		rect := element.layoutRect
		if rect.Empty() {
			rect = element.Bounds()
		}
		if element.handleScrollbarClick(event.X, event.Y, rect, style) {
			handled = true
		} else if element.setCaretFromPoint(event.X, event.Y, rect, style) {
			handled = true
		}
	}
	if element.OnClick == nil {
		return handled
	}
	switch handler := element.OnClick.(type) {
	case func():
		handler()
		return true
	case func(Event):
		handler(event)
		return true
	default:
		return handled
	}
}

func (element *Element) HandleKey(key kos.KeyEvent) bool {
	if element == nil || !element.isTextInput() || !element.focused {
		return false
	}
	if key.Empty || key.Hotkey {
		return false
	}
	control := kos.ControlKeysStatus()
	shiftPressed := control.Shift()
	ctrlPressed := control.Ctrl()
	altPressed := control.Alt()
	winPressed := control&(kos.ControlWinLeft|kos.ControlWinRight) != 0
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	layout := element.textInputLayout(rect, style)
	lines := layout.lines
	changed := false

	if ctrlPressed {
		switch {
		case keyMatchesLetter(key, 'a', scanCodeA):
			changed = element.selectAll()
		case keyMatchesLetter(key, 'c', scanCodeC):
			changed = element.copySelection()
		case keyMatchesLetter(key, 'x', scanCodeX):
			changed = element.cutSelection()
		case keyMatchesLetter(key, 'v', scanCodeV):
			changed = element.pasteFromClipboard()
		}
		if changed {
			layout.release()
			layout = element.textInputLayout(rect, style)
			overflowX := overflowModeFor(style, "x")
			element.ensureCaretVisibleWithLines(layout.content, layout.lines, overflowX, layout.font, layout.charWidth, layout.lineHeight)
			layout.release()
			return true
		}
	}
	if key.ScanCode == scanCodeWinLeft || key.ScanCode == scanCodeWinRight {
		layout.release()
		return false
	}
	if ctrlPressed || altPressed || winPressed {
		layout.release()
		return false
	}

	switch {
	case key.Code == 8:
		changed = element.deleteBackward()
	case key.Code == 13:
		if element.kind == ElementKindTextarea {
			changed = element.insertAtCaret("\n")
		}
	case key.Code == 127 || key.ScanCode == 0x53:
		changed = element.deleteForward()
	case key.ScanCode == 0x4B:
		changed = element.moveCaretHorizontal(-1, shiftPressed)
	case key.ScanCode == 0x4D:
		changed = element.moveCaretHorizontal(1, shiftPressed)
	case key.ScanCode == 0x47:
		changed = element.moveCaretLineBoundary(lines, true, shiftPressed)
	case key.ScanCode == 0x4F:
		changed = element.moveCaretLineBoundary(lines, false, shiftPressed)
	case key.ScanCode == 0x48:
		if element.kind == ElementKindTextarea {
			changed = element.moveCaretVertical(lines, -1, shiftPressed)
		}
	case key.ScanCode == 0x50:
		if element.kind == ElementKindTextarea {
			changed = element.moveCaretVertical(lines, 1, shiftPressed)
		}
	default:
		if key.Code >= 32 && key.Code != 127 {
			if value := keyCodeToString(key.Code); value != "" {
				changed = element.insertAtCaret(value)
			}
		}
	}

	if changed {
		layout.release()
		layout = element.textInputLayout(rect, style)
		overflowX := overflowModeFor(style, "x")
		element.ensureCaretVisibleWithLines(layout.content, layout.lines, overflowX, layout.font, layout.charWidth, layout.lineHeight)
	}
	layout.release()
	return changed
}

func (element *Element) HandleScroll(deltaX int, deltaY int) bool {
	if element == nil || !element.isTextInput() {
		return false
	}
	if deltaX == 0 && deltaY == 0 {
		return false
	}
	style := element.effectiveStyle()
	overflowY := overflowModeFor(style, "y")
	overflowX := overflowModeFor(style, "x")
	allowHScroll := element.kind == ElementKindInput ||
		(element.kind == ElementKindTextarea && (overflowX == OverflowScroll || overflowX == OverflowAuto))
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	layout := element.textInputLayout(rect, style)
	defer layout.release()
	content := layout.content
	totalHeight := layout.totalHeight
	changed := false
	if element.kind == ElementKindTextarea {
		if overflowY != OverflowHidden && overflowY != OverflowVisible {
			maxScrollY := 0
			if content.Height > 0 && totalHeight > content.Height {
				maxScrollY = totalHeight - content.Height
			}
			if maxScrollY > 0 && deltaY != 0 {
				step := layout.lineHeight * 3
				if step < layout.lineHeight {
					step = layout.lineHeight
				}
				prev := element.scrollY
				element.scrollY += deltaY * step
				if element.scrollY < 0 {
					element.scrollY = 0
				}
				if element.scrollY > maxScrollY {
					element.scrollY = maxScrollY
				}
				if element.scrollY != prev {
					changed = true
				}
			}
		}
	}
	if allowHScroll && deltaX != 0 {
		lines := layout.lines
		if len(lines) == 0 {
			lines = []textLine{{text: "", start: 0, end: 0}}
		}
		textWidth := maxLineWidth(lines, layout.font, layout.charWidth)
		maxScrollX := 0
		if content.Width > 0 && textWidth > content.Width {
			maxScrollX = textWidth - content.Width
		}
		if maxScrollX > 0 {
			step := layout.charWidth * 3
			if step < layout.charWidth {
				step = layout.charWidth
			}
			prev := element.scrollX
			element.scrollX += deltaX * step
			if element.scrollX < 0 {
				element.scrollX = 0
			}
			if element.scrollX > maxScrollX {
				element.scrollX = maxScrollX
			}
			if element.scrollX != prev {
				changed = true
			}
		}
	}
	if !changed {
		return false
	}
	element.markDirty()
	return true
}

func (element *Element) selectAll() bool {
	if element == nil {
		return false
	}
	textLen := len(element.text())
	if textLen == 0 {
		return element.clearSelection()
	}
	element.selectAnchor = 0
	element.caret = textLen
	element.desiredCol = -1
	element.markDirty()
	return true
}

func (element *Element) copySelection() bool {
	start, end, ok := element.selectionRange()
	if !ok {
		return false
	}
	selection := element.text()[start:end]
	if selection == "" {
		return false
	}
	return kos.ClipboardCopyText(selection) == kos.ClipboardOK
}

func (element *Element) cutSelection() bool {
	if !element.copySelection() {
		return false
	}
	return element.deleteSelection()
}

func (element *Element) pasteFromClipboard() bool {
	if element == nil {
		return false
	}
	value, ok := readClipboardText()
	if !ok || value == "" {
		return false
	}
	if element.kind == ElementKindInput {
		value = sanitizeSingleLine(value)
	}
	if value == "" {
		return false
	}
	if element.hasSelection() {
		element.deleteSelection()
	}
	return element.insertAtCaret(value)
}

func readClipboardText() (string, bool) {
	count, status := kos.ClipboardSlotCount()
	if status != kos.ClipboardOK || count <= 0 {
		return "", false
	}
	ptr, status := kos.ClipboardSlotData(count - 1)
	if status != kos.ClipboardOK || ptr == 0 {
		return "", false
	}
	size := kos.ReadUint32Raw(ptr, 0)
	kind := kos.ReadUint32Raw(ptr, 4)
	if kind != uint32(kos.ClipboardTypeText) && kind != uint32(kos.ClipboardTypeTextBlock) {
		return "", false
	}
	offset := uint32(12)
	if size <= offset {
		return "", false
	}
	data := kos.CopyBytesRaw(ptr+offset, size-offset)
	if len(data) == 0 {
		return "", false
	}
	if idx := bytes.IndexByte(data, 0); idx >= 0 {
		data = data[:idx]
	}
	if len(data) == 0 {
		return "", false
	}
	return string(data), true
}

func sanitizeSingleLine(value string) string {
	if value == "" {
		return value
	}
	raw := []byte(value)
	out := raw[:0]
	for _, b := range raw {
		if b == '\n' || b == '\r' {
			out = append(out, ' ')
			continue
		}
		out = append(out, b)
	}
	return string(out)
}

func (element *Element) deleteSelection() bool {
	start, end, ok := element.selectionRange()
	if !ok {
		return false
	}
	return element.deleteRange(start, end)
}

func (element *Element) insertAtCaret(value string) bool {
	if element == nil || value == "" {
		return false
	}
	if element.hasSelection() {
		element.deleteSelection()
	}
	text := element.text()
	pos := element.caret
	if pos < 0 {
		pos = 0
	} else if pos > len(text) {
		pos = len(text)
	}
	pos = textClampIndexToRuneBoundary(text, pos)
	updated := text[:pos] + value + text[pos:]
	element.Text = updated
	element.clearTextCache()
	element.caret = pos + len(value)
	element.desiredCol = -1
	element.selectAnchor = element.caret
	element.markDirty()
	return true
}

func (element *Element) deleteRange(start int, end int) bool {
	if element == nil || start >= end {
		return false
	}
	text := element.text()
	if start < 0 {
		start = 0
	}
	if end > len(text) {
		end = len(text)
	}
	start = textClampIndexToRuneBoundary(text, start)
	end = textClampIndexToRuneBoundary(text, end)
	if start >= end {
		return false
	}
	updated := text[:start] + text[end:]
	element.Text = updated
	element.clearTextCache()
	element.caret = start
	element.desiredCol = -1
	element.selectAnchor = element.caret
	element.markDirty()
	return true
}

func (element *Element) deleteBackward() bool {
	if element.hasSelection() {
		return element.deleteSelection()
	}
	if element == nil || element.caret <= 0 {
		return false
	}
	text := element.text()
	start := textPrevRuneIndex(text, element.caret)
	if start == element.caret {
		return false
	}
	return element.deleteRange(start, element.caret)
}

func (element *Element) deleteForward() bool {
	if element == nil {
		return false
	}
	if element.hasSelection() {
		return element.deleteSelection()
	}
	if element.caret >= len(element.text()) {
		return false
	}
	text := element.text()
	end := textNextRuneIndex(text, element.caret)
	if end == element.caret {
		return false
	}
	return element.deleteRange(element.caret, end)
}

func (element *Element) moveCaretHorizontal(delta int, selectMode bool) bool {
	if element == nil {
		return false
	}
	if !selectMode {
		if start, end, ok := element.selectionRange(); ok {
			target := start
			if delta > 0 {
				target = end
			}
			changed := element.setCaret(target)
			element.selectAnchor = element.caret
			if !changed {
				element.markDirty()
			}
			return true
		}
		target := element.caret
		text := element.text()
		if delta < 0 {
			target = textPrevRuneIndex(text, target)
		} else if delta > 0 {
			target = textNextRuneIndex(text, target)
		}
		changed := element.setCaret(target)
		if changed {
			element.selectAnchor = element.caret
		}
		return changed
	}
	if !element.hasSelection() {
		element.selectAnchor = element.caret
	}
	target := element.caret
	text := element.text()
	if delta < 0 {
		target = textPrevRuneIndex(text, target)
	} else if delta > 0 {
		target = textNextRuneIndex(text, target)
	}
	return element.setCaret(target)
}

func (element *Element) moveCaretVertical(lines []textLine, delta int, selectMode bool) bool {
	if element == nil || len(lines) == 0 {
		return false
	}
	if !selectMode {
		if start, end, ok := element.selectionRange(); ok {
			target := start
			if delta > 0 {
				target = end
			}
			changed := element.setCaret(target)
			element.selectAnchor = element.caret
			if !changed {
				element.markDirty()
			}
			return true
		}
	}
	if selectMode && !element.hasSelection() {
		element.selectAnchor = element.caret
	}
	line, col := element.caretLineAndColumn(lines)
	if element.desiredCol < 0 {
		element.desiredCol = col
	}
	targetLine := line + delta
	if targetLine < 0 {
		targetLine = 0
	} else if targetLine >= len(lines) {
		targetLine = len(lines) - 1
	}
	targetCol := element.desiredCol
	newCaret := caretIndexForLineColumn(lines, targetLine, targetCol)
	if newCaret == element.caret {
		return false
	}
	element.caret = newCaret
	element.markDirty()
	if !selectMode {
		element.selectAnchor = element.caret
	}
	return true
}

func (element *Element) moveCaretLineBoundary(lines []textLine, toStart bool, selectMode bool) bool {
	if element == nil || len(lines) == 0 {
		return false
	}
	line, _ := element.caretLineAndColumn(lines)
	target := element.caret
	if toStart {
		target = lines[line].start
	} else {
		target = lines[line].end
	}
	if !selectMode {
		if changed := element.setCaret(target); changed {
			element.selectAnchor = element.caret
			return true
		}
		return element.clearSelection()
	}
	if !element.hasSelection() {
		element.selectAnchor = element.caret
	}
	return element.setCaret(target)
}
