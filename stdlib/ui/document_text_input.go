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
	node.selectAnchor = node.inputCaret
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

func documentNodeHasSelection(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	return node.selectAnchor != node.inputCaret
}

func documentNodeSelectionRange(node *DocumentNode) (int, int, bool) {
	if node == nil || !documentNodeHasSelection(node) {
		return 0, 0, false
	}
	start := node.selectAnchor
	end := node.inputCaret
	if start > end {
		start, end = end, start
	}
	start = textClampIndexToRuneBoundary(node.Value, start)
	end = textClampIndexToRuneBoundary(node.Value, end)
	if start >= end {
		return 0, 0, false
	}
	return start, end, true
}

func documentNodeClearSelection(node *DocumentNode) bool {
	if node == nil || !documentNodeHasSelection(node) {
		return false
	}
	node.selectAnchor = node.inputCaret
	return true
}

func documentNodeMoveCaret(node *DocumentNode, caret int, selectMode bool) bool {
	if node == nil {
		return false
	}
	if !selectMode {
		changed := documentNodeInputSetCaret(node, caret)
		node.selectAnchor = node.inputCaret
		return changed
	}
	if !documentNodeHasSelection(node) {
		node.selectAnchor = node.inputCaret
	}
	return documentNodeInputSetCaret(node, caret)
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

func documentNodeInputSelectionRect(node *DocumentNode, bounds Rect, style Style) Rect {
	if node == nil {
		return Rect{}
	}
	start, end, ok := documentNodeSelectionRange(node)
	if !ok {
		return Rect{}
	}
	content := documentNodeInputContentRect(bounds, style)
	if content.Empty() {
		return Rect{}
	}
	font, _, charWidth, lineHeight := documentNodeInputLineMetrics(style)
	startCol := textColumnForByteIndex(node.Value, start)
	endCol := textColumnForByteIndex(node.Value, end)
	xStart := content.X + textWidthForColumns(node.Value, startCol, font, charWidth) - node.inputScrollX
	xEnd := content.X + textWidthForColumns(node.Value, endCol, font, charWidth) - node.inputScrollX
	return IntersectRect(Rect{X: xStart, Y: content.Y, Width: xEnd - xStart, Height: lineHeight}, content)
}

func documentNodeInputDirtyRect(node *DocumentNode, bounds Rect, style Style) Rect {
	if node == nil {
		return Rect{}
	}
	rect := documentNodeInputContentRect(bounds, style)
	if rect.Empty() {
		rect = bounds
	}
	return rect
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

func documentNodeInputDeleteRange(node *DocumentNode, start int, end int) bool {
	if node == nil || start >= end {
		return false
	}
	value := node.Value
	if start < 0 {
		start = 0
	}
	if end > len(value) {
		end = len(value)
	}
	start = textClampIndexToRuneBoundary(value, start)
	end = textClampIndexToRuneBoundary(value, end)
	if start >= end {
		return false
	}
	node.Value = value[:start] + value[end:]
	node.inputCaret = start
	node.selectAnchor = node.inputCaret
	return true
}

func documentNodeInputDeleteSelection(node *DocumentNode) bool {
	start, end, ok := documentNodeSelectionRange(node)
	if !ok {
		return false
	}
	return documentNodeInputDeleteRange(node, start, end)
}

func documentNodeInputInsertAtCaret(node *DocumentNode, value string) bool {
	if node == nil || value == "" {
		return false
	}
	if documentNodeHasSelection(node) {
		documentNodeInputDeleteSelection(node)
	}
	text := node.Value
	caret := textClampIndexToRuneBoundary(text, node.inputCaret)
	node.Value = text[:caret] + value + text[caret:]
	node.inputCaret = caret + len(value)
	node.selectAnchor = node.inputCaret
	return true
}

func documentNodeInputDeleteBackward(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	if documentNodeHasSelection(node) {
		return documentNodeInputDeleteSelection(node)
	}
	caret := textClampIndexToRuneBoundary(node.Value, node.inputCaret)
	if caret <= 0 {
		return false
	}
	prev := textPrevRuneIndex(node.Value, caret)
	return documentNodeInputDeleteRange(node, prev, caret)
}

func documentNodeInputDeleteForward(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	if documentNodeHasSelection(node) {
		return documentNodeInputDeleteSelection(node)
	}
	caret := textClampIndexToRuneBoundary(node.Value, node.inputCaret)
	if caret >= len(node.Value) {
		return false
	}
	next := textNextRuneIndex(node.Value, caret)
	return documentNodeInputDeleteRange(node, caret, next)
}

func documentNodeInputMoveCaretHorizontal(node *DocumentNode, delta int, selectMode bool) bool {
	if node == nil {
		return false
	}
	if !selectMode {
		if start, end, ok := documentNodeSelectionRange(node); ok {
			target := start
			if delta > 0 {
				target = end
			}
			return documentNodeMoveCaret(node, target, false)
		}
	}
	target := node.inputCaret
	if delta < 0 {
		target = textPrevRuneIndex(node.Value, target)
	} else if delta > 0 {
		target = textNextRuneIndex(node.Value, target)
	}
	return documentNodeMoveCaret(node, target, selectMode)
}

func documentNodeInputMoveCaretBoundary(node *DocumentNode, toStart bool, selectMode bool) bool {
	if node == nil {
		return false
	}
	target := len(node.Value)
	if toStart {
		target = 0
	}
	return documentNodeMoveCaret(node, target, selectMode)
}

func documentNodeInputSelectAll(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	if len(node.Value) == 0 {
		return documentNodeClearSelection(node)
	}
	node.selectAnchor = 0
	node.inputCaret = len(node.Value)
	return true
}

func documentNodeInputCopySelection(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	start, end, ok := documentNodeSelectionRange(node)
	if !ok || start >= end {
		return false
	}
	return kos.ClipboardCopyText(node.Value[start:end]) == kos.ClipboardOK
}

func documentNodeInputCutSelection(node *DocumentNode) bool {
	if !documentNodeInputCopySelection(node) {
		return false
	}
	return documentNodeInputDeleteSelection(node)
}

func documentNodeInputPaste(node *DocumentNode) bool {
	if node == nil {
		return false
	}
	value, ok := readClipboardText()
	if !ok || value == "" {
		return false
	}
	value = sanitizeSingleLine(value)
	if value == "" {
		return false
	}
	return documentNodeInputInsertAtCaret(node, value)
}

func documentNodeInputHandleKey(node *DocumentNode, key kos.KeyEvent) (valueChanged bool, submitted bool, stateChanged bool) {
	if node == nil || !documentNodeIsTextInput(node) {
		return false, false, false
	}
	control := kos.ControlKeysStatus()
	shiftPressed := control.Shift()
	ctrlPressed := control.Ctrl()
	altPressed := control.Alt()
	winPressed := control&(kos.ControlWinLeft|kos.ControlWinRight) != 0
	if ctrlPressed {
		switch {
		case keyMatchesLetter(key, 'a', scanCodeA):
			return false, false, documentNodeInputSelectAll(node)
		case keyMatchesLetter(key, 'c', scanCodeC):
			return false, false, false
		case keyMatchesLetter(key, 'x', scanCodeX):
			return documentNodeInputCutSelection(node), false, false
		case keyMatchesLetter(key, 'v', scanCodeV):
			return documentNodeInputPaste(node), false, false
		}
	}
	if key.ScanCode == scanCodeWinLeft || key.ScanCode == scanCodeWinRight {
		return false, false, false
	}
	if ctrlPressed || altPressed || winPressed {
		return false, false, false
	}
	switch {
	case key.Code == 13:
		node.inputCaret = textClampIndexToRuneBoundary(node.Value, node.inputCaret)
		return false, true, false
	case key.Code == 8:
		return documentNodeInputDeleteBackward(node), false, false
	case key.Code == 127 || key.ScanCode == 0x53:
		return documentNodeInputDeleteForward(node), false, false
	case key.ScanCode == 0x4B:
		return false, false, documentNodeInputMoveCaretHorizontal(node, -1, shiftPressed)
	case key.ScanCode == 0x4D:
		return false, false, documentNodeInputMoveCaretHorizontal(node, 1, shiftPressed)
	case key.ScanCode == 0x47:
		return false, false, documentNodeInputMoveCaretBoundary(node, true, shiftPressed)
	case key.ScanCode == 0x4F:
		return false, false, documentNodeInputMoveCaretBoundary(node, false, shiftPressed)
	default:
		if key.Code >= 32 && key.Code != 127 {
			if inserted := keyCodeToString(key.Code); inserted != "" {
				return documentNodeInputInsertAtCaret(node, inserted), false, false
			}
		}
	}
	return false, false, false
}

func documentNodeCaretVisible(node *DocumentNode) bool {
	if node == nil || !documentNodeIsTextInput(node) || !node.focused || documentNodeHasSelection(node) {
		return false
	}
	document := node.document
	if document == nil || document.host == nil || document.host.window == nil {
		return true
	}
	return document.host.window.caretBlinkVisible()
}
