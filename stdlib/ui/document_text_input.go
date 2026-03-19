package ui

import "kos"

func documentNodeIsTextInput(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	if node.Editable {
		return true
	}
	return node.Name == "input"
}

func documentNodeInputValue(node *DocumentNode) string {
	if node == nil {
		return ""
	}
	return node.Value
}

func documentNodeInputDisplayText(node *DocumentNode) (string, bool) {
	if node == nil {
		return "", false
	}
	if node.Value != "" {
		return node.Value, false
	}
	if node.Placeholder != "" {
		return node.Placeholder, true
	}
	return "", false
}

func documentNodeInputSetValue(node *DocumentNode, value string) bool {
	if node == nil || node.Value == value {
		return false
	}
	node.Value = value
	if node.inputCaret > len(value) {
		node.inputCaret = len(value)
	}
	node.inputCaret = textClampIndexToRuneBoundary(value, node.inputCaret)
	return true
}

func documentNodeInputSetCaret(node *DocumentNode, caret int) bool {
	if node == nil {
		return false
	}
	caret = textClampIndexToRuneBoundary(node.Value, caret)
	if caret < 0 {
		caret = 0
	}
	if caret > len(node.Value) {
		caret = len(node.Value)
	}
	if node.inputCaret == caret {
		return false
	}
	node.inputCaret = caret
	return true
}

func documentNodeInputLineMetrics(style Style) (*ttfFont, fontMetrics, int, int) {
	font, metrics := fontAndMetricsForStyle(style)
	charWidth := metrics.width
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	lineHeight := lineHeightForStyle(style, metrics.height)
	if lineHeight <= 0 {
		lineHeight = defaultFontHeight
	}
	return font, metrics, charWidth, lineHeight
}

func documentNodeInputContentRect(bounds Rect, style Style) Rect {
	return contentRectFor(bounds, style)
}

func documentNodeInputHeight(style Style) int {
	_, _, _, lineHeight := documentNodeInputLineMetrics(style)
	insets := boxInsets(style)
	height := insets.Top + lineHeight + insets.Bottom
	return clampHeightForStyle(style, height)
}

func documentNodeInputCaretRect(node *DocumentNode, bounds Rect, style Style, caretVisible bool) Rect {
	if node == nil || !documentNodeIsTextInput(node) || !caretVisible {
		return Rect{}
	}
	content := documentNodeInputContentRect(bounds, style)
	if content.Empty() {
		return Rect{}
	}
	font, _, charWidth, lineHeight := documentNodeInputLineMetrics(style)
	caret := textClampIndexToRuneBoundary(node.Value, node.inputCaret)
	caretX := content.X + textWidthForColumns(node.Value, textColumnForByteIndex(node.Value, caret), font, charWidth) - node.inputScrollX
	return IntersectRect(Rect{X: caretX, Y: content.Y, Width: 1, Height: lineHeight}, content)
}

func documentNodeInputEnsureCaretVisible(node *DocumentNode, bounds Rect, style Style) {
	if node == nil || !documentNodeIsTextInput(node) {
		return
	}
	content := documentNodeInputContentRect(bounds, style)
	if content.Empty() {
		node.inputScrollX = 0
		return
	}
	font, _, charWidth, _ := documentNodeInputLineMetrics(style)
	caret := textClampIndexToRuneBoundary(node.Value, node.inputCaret)
	caretCol := textColumnForByteIndex(node.Value, caret)
	caretX := textWidthForColumns(node.Value, caretCol, font, charWidth)
	if caretX < node.inputScrollX {
		node.inputScrollX = caretX
		return
	}
	if content.Width > 0 && caretX > node.inputScrollX+content.Width-charWidth {
		node.inputScrollX = caretX - (content.Width - charWidth)
		if node.inputScrollX < 0 {
			node.inputScrollX = 0
		}
		return
	}
	maxScroll := textWidthWithFont(node.Value, font, charWidth) - content.Width
	if maxScroll < 0 {
		maxScroll = 0
	}
	if node.inputScrollX > maxScroll {
		node.inputScrollX = maxScroll
	}
}

func documentNodeInputCaretFromX(node *DocumentNode, bounds Rect, style Style, x int) int {
	if node == nil {
		return 0
	}
	content := documentNodeInputContentRect(bounds, style)
	if content.Empty() {
		return len(node.Value)
	}
	font, _, charWidth, _ := documentNodeInputLineMetrics(style)
	target := x - content.X + node.inputScrollX
	if target < 0 {
		target = 0
	}
	col := textColumnForX(node.Value, target, font, charWidth)
	return textByteIndexForColumn(node.Value, col)
}

func documentNodeInputHandleKey(node *DocumentNode, key kos.KeyEvent) (changed bool, submitted bool) {
	if node == nil || !documentNodeIsTextInput(node) {
		return false, false
	}
	value := node.Value
	caret := textClampIndexToRuneBoundary(value, node.inputCaret)
	switch {
	case key.Code == 13:
		node.inputCaret = caret
		return false, true
	case key.Code == 8:
		if caret <= 0 {
			return false, false
		}
		prev := textPrevRuneIndex(value, caret)
		node.Value = value[:prev] + value[caret:]
		node.inputCaret = prev
		return true, false
	case key.Code == 127 || key.ScanCode == 0x53:
		if caret >= len(value) {
			return false, false
		}
		next := textNextRuneIndex(value, caret)
		node.Value = value[:caret] + value[next:]
		node.inputCaret = caret
		return true, false
	case key.ScanCode == 0x4B:
		node.inputCaret = textPrevRuneIndex(value, caret)
		return false, false
	case key.ScanCode == 0x4D:
		node.inputCaret = textNextRuneIndex(value, caret)
		return false, false
	case key.ScanCode == 0x47:
		node.inputCaret = 0
		return false, false
	case key.ScanCode == 0x4F:
		node.inputCaret = len(value)
		return false, false
	default:
		if key.Code >= 32 && key.Code != 127 {
			if inserted := keyCodeToString(key.Code); inserted != "" {
				node.Value = value[:caret] + inserted + value[caret:]
				node.inputCaret = caret + len(inserted)
				return true, false
			}
		}
	}
	return false, false
}

func documentNodeCaretVisible(node *DocumentNode) bool {
	if node == nil || !documentNodeIsTextInput(node) || !node.focused {
		return false
	}
	document := node.document
	if document == nil || document.host == nil || document.host.window == nil {
		return true
	}
	return document.host.window.caretBlinkVisible()
}
