package ui

import (
	"unicode"
	"unicode/utf8"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func (element *Element) text() string {
	if element != nil && element.isTextInput() {
		return element.Text
	}
	if element.Text != "" {
		return element.Text
	}
	return element.Label
}

type textWrapCache struct {
	text         string
	maxWidth     int
	charWidth    int
	fontKey      fontKey
	hasFont      bool
	whiteSpace   WhiteSpaceMode
	overflowWrap OverflowWrapMode
	wordBreak    WordBreakMode
	lines        []textLine
}

type textPreserveCache struct {
	text      string
	maxWidth  int
	charWidth int
	fontKey   fontKey
	hasFont   bool
	wrap      bool
	lines     []textLine
}

type textLine struct {
	text  string
	start int
	end   int
}

func (element *Element) clearTextCache() {
	if element == nil {
		return
	}
	releaseTextLines(element.wrapCache.lines)
	element.wrapCache = textWrapCache{}
	releaseTextLines(element.preserveCache.lines)
	element.preserveCache = textPreserveCache{}
}

func (element *Element) wrapTextLinesCached(text string, maxWidth int, font *ttfFont, charWidth int) []textLine {
	return element.wrapTextLinesCachedStyle(text, maxWidth, font, charWidth, Style{})
}

func (element *Element) wrapTextLinesCachedStyle(text string, maxWidth int, font *ttfFont, charWidth int, style Style) []textLine {
	if element == nil || FastNoTextCache {
		return wrapTextForStyle(text, maxWidth, font, charWidth, style)
	}
	cache := &element.wrapCache
	hasFont := font != nil
	key := fontKey{}
	if hasFont {
		key = fontKey{path: font.path, size: font.size}
	}
	whiteSpace := whiteSpaceForStyle(style)
	overflowWrap := overflowWrapForStyle(style)
	wordBreak := wordBreakForStyle(style)
	if cache.lines != nil && cache.text == text && cache.maxWidth == maxWidth && cache.charWidth == charWidth &&
		cache.hasFont == hasFont && cache.whiteSpace == whiteSpace &&
		cache.overflowWrap == overflowWrap && cache.wordBreak == wordBreak &&
		(!hasFont || cache.fontKey == key) {
		return cache.lines
	}
	if cache.lines != nil {
		releaseTextLines(cache.lines)
	}
	lines := wrapTextForStyle(text, maxWidth, font, charWidth, style)
	cache.text = text
	cache.maxWidth = maxWidth
	cache.charWidth = charWidth
	cache.fontKey = key
	cache.hasFont = hasFont
	cache.whiteSpace = whiteSpace
	cache.overflowWrap = overflowWrap
	cache.wordBreak = wordBreak
	cache.lines = lines
	return lines
}

func (element *Element) wrapTextPreserveCached(text string, maxWidth int, wrap bool, font *ttfFont, charWidth int) []textLine {
	if element == nil || FastNoTextCache {
		return wrapTextPreserve(text, maxWidth, wrap, font, charWidth)
	}
	cache := &element.preserveCache
	hasFont := font != nil
	key := fontKey{}
	if hasFont {
		key = fontKey{path: font.path, size: font.size}
	}
	if cache.lines != nil && cache.text == text && cache.maxWidth == maxWidth && cache.charWidth == charWidth &&
		cache.wrap == wrap && cache.hasFont == hasFont && (!hasFont || cache.fontKey == key) {
		return cache.lines
	}
	if cache.lines != nil {
		releaseTextLines(cache.lines)
	}
	lines := wrapTextPreserve(text, maxWidth, wrap, font, charWidth)
	cache.text = text
	cache.maxWidth = maxWidth
	cache.charWidth = charWidth
	cache.fontKey = key
	cache.hasFont = hasFont
	cache.wrap = wrap
	cache.lines = lines
	return lines
}

type textWrapOptions struct {
	preserve       bool
	wrap           bool
	breakLongWords bool
	breakAll       bool
}

func textWrapOptionsForStyle(style Style) textWrapOptions {
	options := textWrapOptions{
		wrap:           true,
		breakLongWords: overflowWrapForStyle(style) == OverflowWrapBreakWord,
	}
	switch whiteSpaceForStyle(style) {
	case WhiteSpaceNoWrap:
		options.wrap = false
	case WhiteSpacePre:
		options.preserve = true
		options.wrap = false
	case WhiteSpacePreWrap:
		options.preserve = true
	case WhiteSpacePreLine:
		options.wrap = true
	default:
		options.wrap = true
	}
	if wordBreakForStyle(style) == WordBreakBreakAll {
		options.breakAll = true
		options.breakLongWords = true
	}
	if !options.wrap {
		options.breakAll = false
		options.breakLongWords = false
	}
	return options
}

func wrapTextForStyle(text string, maxWidth int, font *ttfFont, charWidth int, style Style) []textLine {
	options := textWrapOptionsForStyle(style)
	if options.breakAll || options.preserve || !options.wrap {
		return wrapTextPreserve(text, maxWidth, options.wrap, font, charWidth)
	}
	return wrapTextLines(text, maxWidth, font, charWidth, options.breakLongWords)
}

func wrapTextLines(text string, maxWidth int, font *ttfFont, charWidth int, breakLongWords bool) []textLine {
	if font != nil {
		return wrapTextLinesFont(text, maxWidth, font.face, breakLongWords)
	}
	return wrapTextLinesMono(text, maxWidth, charWidth, breakLongWords)
}

func wrapTextLinesMono(text string, maxWidth int, charWidth int, breakLongWords bool) []textLine {
	if text == "" {
		return nil
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	maxChars := maxWidth / charWidth
	if maxChars <= 0 {
		lines := getTextLineSlice(1)
		return append(lines, textLine{text: text, start: 0, end: len(text)})
	}
	lines := getTextLineSlice(4)
	start := 0
	for {
		if start > len(text) {
			break
		}
		end := start
		for end < len(text) && text[end] != '\n' {
			end++
		}
		if start >= end {
			lines = append(lines, textLine{text: "", start: start, end: start})
		} else {
			rawLine := text[start:end]
			if isASCIIString(rawLine) {
				lines = appendWrapWordsASCII(lines, text, start, end, maxChars, breakLongWords)
			} else {
				lines = appendWrapWordsUnicode(lines, text, start, end, maxChars, breakLongWords)
			}
		}
		if end >= len(text) {
			break
		}
		start = end + 1
	}
	return lines
}

func wrapTextLinesFont(text string, maxWidth int, face xfont.Face, breakLongWords bool) []textLine {
	if text == "" {
		return nil
	}
	if face == nil {
		return wrapTextLinesMono(text, maxWidth, defaultCharWidth, breakLongWords)
	}
	if maxWidth <= 0 {
		lines := getTextLineSlice(1)
		return append(lines, textLine{text: text, start: 0, end: len(text)})
	}
	lines := getTextLineSlice(4)
	start := 0
	for {
		if start > len(text) {
			break
		}
		end := start
		for end < len(text) && text[end] != '\n' {
			end++
		}
		if start >= end {
			lines = append(lines, textLine{text: "", start: start, end: start})
		} else {
			lines = appendWrapWordsFont(lines, text, start, end, maxWidth, face, breakLongWords)
		}
		if end >= len(text) {
			break
		}
		start = end + 1
	}
	return lines
}

func appendWrapWordsFont(lines []textLine, text string, rawStart, rawEnd int, maxWidth int, face xfont.Face, breakLongWords bool) []textLine {
	if rawStart >= rawEnd {
		return append(lines, textLine{text: "", start: rawStart, end: rawStart})
	}
	if face == nil {
		return append(lines, textLine{text: text[rawStart:rawEnd], start: rawStart, end: rawEnd})
	}
	if maxWidth <= 0 {
		return append(lines, textLine{text: text[rawStart:rawEnd], start: rawStart, end: rawEnd})
	}
	maxWidthFixed := fixed.I(maxWidth)
	start := rawStart
	for start < rawEnd {
		r, size := utf8.DecodeRuneInString(text[start:rawEnd])
		if !unicode.IsSpace(r) {
			break
		}
		start += size
	}
	if start >= rawEnd {
		return append(lines, textLine{text: "", start: rawStart, end: rawStart})
	}
	lineStart := start
	lastBreak := -1
	lastNonSpaceEnd := lineStart
	var width fixed.Int26_6
	prev := rune(-1)
	for start < rawEnd {
		r, size := utf8.DecodeRuneInString(text[start:rawEnd])
		kern := fixed.Int26_6(0)
		if prev >= 0 {
			kern = face.Kern(prev, r)
		}
		advance, _ := face.GlyphAdvance(r)
		nextWidth := width + kern + advance
		if nextWidth > maxWidthFixed && width > 0 {
			if lastBreak >= 0 && lastNonSpaceEnd > lineStart {
				lines = append(lines, textLine{text: text[lineStart:lastNonSpaceEnd], start: lineStart, end: lastNonSpaceEnd})
				start = lastBreak
			} else if !breakLongWords {
				width = nextWidth
				if !unicode.IsSpace(r) {
					lastNonSpaceEnd = start + size
				}
				prev = r
				start += size
				continue
			} else {
				lines = append(lines, textLine{text: text[lineStart:start], start: lineStart, end: start})
			}
			for start < rawEnd {
				r, size = utf8.DecodeRuneInString(text[start:rawEnd])
				if !unicode.IsSpace(r) {
					break
				}
				start += size
			}
			lineStart = start
			lastBreak = -1
			lastNonSpaceEnd = lineStart
			width = 0
			prev = -1
			continue
		}
		width = nextWidth
		if unicode.IsSpace(r) {
			if lastNonSpaceEnd > lineStart {
				lastBreak = start
			}
		} else {
			lastNonSpaceEnd = start + size
		}
		prev = r
		start += size
	}
	if lineStart < rawEnd {
		lineEnd := lastNonSpaceEnd
		if lineEnd < lineStart {
			lineEnd = lineStart
		}
		lines = append(lines, textLine{text: text[lineStart:lineEnd], start: lineStart, end: lineEnd})
	}
	return lines
}

func appendWrapWordsASCII(lines []textLine, text string, rawStart, rawEnd, maxChars int, breakLongWords bool) []textLine {
	if rawStart >= rawEnd {
		return append(lines, textLine{text: "", start: rawStart, end: rawStart})
	}
	i := rawStart
	for i < rawEnd && isASCIIWhitespace(text[i]) {
		i++
	}
	if i >= rawEnd {
		return append(lines, textLine{text: "", start: rawStart, end: rawStart})
	}
	lineStart := i
	for i < rawEnd && !isASCIIWhitespace(text[i]) {
		i++
	}
	lineEnd := i
	lineLen := lineEnd - lineStart
	for {
		for i < rawEnd && isASCIIWhitespace(text[i]) {
			i++
		}
		if i >= rawEnd {
			break
		}
		wordStart := i
		for i < rawEnd && !isASCIIWhitespace(text[i]) {
			i++
		}
		wordEnd := i
		wordLen := wordEnd - wordStart
		gapLen := wordStart - lineEnd
		if lineLen+gapLen+wordLen <= maxChars {
			lineEnd = wordEnd
			lineLen = lineEnd - lineStart
			continue
		}
		lines = append(lines, textLine{text: text[lineStart:lineEnd], start: lineStart, end: lineEnd})
		if wordLen <= maxChars {
			lineStart = wordStart
			lineEnd = wordEnd
			lineLen = wordLen
			continue
		}
		if !breakLongWords {
			lineStart = wordStart
			lineEnd = wordEnd
			lineLen = wordLen
			continue
		}
		for wordLen > maxChars {
			chunkEnd := wordStart + maxChars
			if chunkEnd > wordEnd {
				chunkEnd = wordEnd
			}
			lines = append(lines, textLine{text: text[wordStart:chunkEnd], start: wordStart, end: chunkEnd})
			wordStart = chunkEnd
			wordLen = wordEnd - wordStart
		}
		lineStart = wordStart
		lineEnd = wordEnd
		lineLen = wordLen
	}
	return append(lines, textLine{text: text[lineStart:lineEnd], start: lineStart, end: lineEnd})
}

func appendWrapWordsUnicode(lines []textLine, text string, rawStart, rawEnd, maxChars int, breakLongWords bool) []textLine {
	if rawStart >= rawEnd {
		return append(lines, textLine{text: "", start: rawStart, end: rawStart})
	}
	i := rawStart
	for i < rawEnd {
		r, size := utf8.DecodeRuneInString(text[i:rawEnd])
		if !unicode.IsSpace(r) {
			break
		}
		i += size
	}
	if i >= rawEnd {
		return append(lines, textLine{text: "", start: rawStart, end: rawStart})
	}
	lineStart := i
	for i < rawEnd {
		r, size := utf8.DecodeRuneInString(text[i:rawEnd])
		if unicode.IsSpace(r) {
			break
		}
		i += size
	}
	lineEnd := i
	lineLen := lineEnd - lineStart
	for {
		for i < rawEnd {
			r, size := utf8.DecodeRuneInString(text[i:rawEnd])
			if !unicode.IsSpace(r) {
				break
			}
			i += size
		}
		if i >= rawEnd {
			break
		}
		wordStart := i
		for i < rawEnd {
			r, size := utf8.DecodeRuneInString(text[i:rawEnd])
			if unicode.IsSpace(r) {
				break
			}
			i += size
		}
		wordEnd := i
		wordLen := wordEnd - wordStart
		gapLen := wordStart - lineEnd
		if lineLen+gapLen+wordLen <= maxChars {
			lineEnd = wordEnd
			lineLen = lineEnd - lineStart
			continue
		}
		lines = append(lines, textLine{text: text[lineStart:lineEnd], start: lineStart, end: lineEnd})
		if wordLen <= maxChars {
			lineStart = wordStart
			lineEnd = wordEnd
			lineLen = wordLen
			continue
		}
		if !breakLongWords {
			lineStart = wordStart
			lineEnd = wordEnd
			lineLen = wordLen
			continue
		}
		for wordLen > maxChars {
			chunkEnd := wordStart + maxChars
			if chunkEnd > wordEnd {
				chunkEnd = wordEnd
			}
			lines = append(lines, textLine{text: text[wordStart:chunkEnd], start: wordStart, end: chunkEnd})
			wordStart = chunkEnd
			wordLen = wordEnd - wordStart
		}
		lineStart = wordStart
		lineEnd = wordEnd
		lineLen = wordLen
	}
	return append(lines, textLine{text: text[lineStart:lineEnd], start: lineStart, end: lineEnd})
}

func isASCIIWhitespace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

func isASCIIString(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func textColumnCount(value string) int {
	if value == "" {
		return 0
	}
	if isASCIIString(value) {
		return len(value)
	}
	return utf8.RuneCountInString(value)
}

func textColumnForByteIndex(value string, index int) int {
	if index <= 0 {
		return 0
	}
	if index >= len(value) {
		return textColumnCount(value)
	}
	if isASCIIString(value) {
		return index
	}
	count := 0
	for i := 0; i < len(value) && i < index; {
		_, size := utf8.DecodeRuneInString(value[i:])
		if size <= 0 || i+size > index {
			break
		}
		i += size
		count++
	}
	return count
}

func textByteIndexForColumn(value string, col int) int {
	if col <= 0 {
		return 0
	}
	if isASCIIString(value) {
		if col >= len(value) {
			return len(value)
		}
		return col
	}
	count := 0
	for i := 0; i < len(value); {
		if count >= col {
			return i
		}
		_, size := utf8.DecodeRuneInString(value[i:])
		if size <= 0 {
			return i
		}
		i += size
		count++
	}
	return len(value)
}

func textSliceColumns(value string, startCol int, endCol int) string {
	if value == "" {
		return value
	}
	if startCol < 0 {
		startCol = 0
	}
	if endCol < startCol {
		endCol = startCol
	}
	if isASCIIString(value) {
		if startCol > len(value) {
			startCol = len(value)
		}
		if endCol > len(value) {
			endCol = len(value)
		}
		return value[startCol:endCol]
	}
	start := textByteIndexForColumn(value, startCol)
	end := textByteIndexForColumn(value, endCol)
	if start > end {
		start = end
	}
	return value[start:end]
}

func textClampIndexToRuneBoundary(value string, index int) int {
	if index <= 0 {
		return 0
	}
	if index >= len(value) {
		return len(value)
	}
	if isASCIIString(value) {
		return index
	}
	col := textColumnForByteIndex(value, index)
	return textByteIndexForColumn(value, col)
}

func textPrevRuneIndex(value string, index int) int {
	if index <= 0 {
		return 0
	}
	if index > len(value) {
		index = len(value)
	}
	if isASCIIString(value) {
		return index - 1
	}
	index = textClampIndexToRuneBoundary(value, index)
	if index <= 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(value[:index])
	if size <= 0 {
		return index
	}
	return index - size
}

func textNextRuneIndex(value string, index int) int {
	if index < 0 {
		index = 0
	}
	if index >= len(value) {
		return len(value)
	}
	if isASCIIString(value) {
		return index + 1
	}
	index = textClampIndexToRuneBoundary(value, index)
	if index >= len(value) {
		return len(value)
	}
	_, size := utf8.DecodeRuneInString(value[index:])
	if size <= 0 {
		return index
	}
	return index + size
}

func wrapTextPreserve(text string, maxWidth int, wrap bool, font *ttfFont, charWidth int) []textLine {
	if font != nil {
		return wrapTextPreserveFont(text, maxWidth, wrap, font.face)
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	maxChars := 0
	if maxWidth > 0 {
		maxChars = maxWidth / charWidth
	}
	return wrapTextPreserveMono(text, maxChars, wrap)
}

func wrapTextPreserveFont(text string, maxWidth int, wrap bool, face xfont.Face) []textLine {
	if text == "" {
		lines := getTextLineSlice(1)
		return append(lines, textLine{text: "", start: 0, end: 0})
	}
	if !wrap || maxWidth <= 0 || face == nil {
		lines := getTextLineSlice(4)
		start := 0
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				lines = append(lines, textLine{text: text[start:i], start: start, end: i})
				start = i + 1
			}
		}
		lines = append(lines, textLine{text: text[start:], start: start, end: len(text)})
		return lines
	}
	maxWidthFixed := fixed.I(maxWidth)
	lines := getTextLineSlice(4)
	start := 0
	var width fixed.Int26_6
	prev := rune(-1)
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == '\n' {
			lines = append(lines, textLine{text: text[start:i], start: start, end: i})
			start = i + size
			i = start
			width = 0
			prev = -1
			continue
		}
		kern := fixed.Int26_6(0)
		if prev >= 0 {
			kern = face.Kern(prev, r)
		}
		advance, _ := face.GlyphAdvance(r)
		nextWidth := width + kern + advance
		if nextWidth > maxWidthFixed && width > 0 {
			lines = append(lines, textLine{text: text[start:i], start: start, end: i})
			start = i
			width = 0
			prev = -1
			continue
		}
		width = nextWidth
		prev = r
		i += size
	}
	lines = append(lines, textLine{text: text[start:], start: start, end: len(text)})
	return lines
}

func wrapTextPreserveMono(text string, maxChars int, wrap bool) []textLine {
	if text == "" {
		lines := getTextLineSlice(1)
		return append(lines, textLine{text: "", start: 0, end: 0})
	}
	if !wrap || maxChars <= 0 {
		lines := getTextLineSlice(4)
		start := 0
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				lines = append(lines, textLine{text: text[start:i], start: start, end: i})
				start = i + 1
			}
		}
		lines = append(lines, textLine{text: text[start:], start: start, end: len(text)})
		return lines
	}
	if isASCIIString(text) {
		lines := getTextLineSlice(4)
		start := 0
		count := 0
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				lines = append(lines, textLine{text: text[start:i], start: start, end: i})
				start = i + 1
				count = 0
				continue
			}
			if count >= maxChars {
				lines = append(lines, textLine{text: text[start:i], start: start, end: i})
				start = i
				count = 0
			}
			count++
		}
		lines = append(lines, textLine{text: text[start:], start: start, end: len(text)})
		return lines
	}
	lines := getTextLineSlice(4)
	start := 0
	count := 0
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == '\n' {
			lines = append(lines, textLine{text: text[start:i], start: start, end: i})
			start = i + size
			i = start
			count = 0
			continue
		}
		if count >= maxChars {
			lines = append(lines, textLine{text: text[start:i], start: start, end: i})
			start = i
			count = 0
			continue
		}
		i += size
		count++
	}
	lines = append(lines, textLine{text: text[start:], start: start, end: len(text)})
	return lines
}

func (element *Element) forEachTextLine(rect Rect, style Style, fn func(x, y int, line string)) {
	if element == nil || fn == nil {
		return
	}
	text := element.text()
	if text == "" || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	font, metrics := fontAndMetricsForStyle(style)
	charWidth := metrics.width
	lineHeight := lineHeightForStyle(style, metrics.height)
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	leftPad, topPad, rightPad, availableW := textPaddingAndWidth(rect, style)
	lines := element.wrapTextLinesCachedStyle(text, availableW, font, charWidth, style)
	if len(lines) == 0 {
		return
	}
	for i, line := range lines {
		if line.text == "" {
			continue
		}
		x := textLineX(rect, style, leftPad, rightPad, availableW, line.text, font, charWidth)
		y := rect.Y + topPad + i*lineHeight
		fn(x, y, line.text)
	}
}

func textPaddingAndWidth(rect Rect, style Style) (int, int, int, int) {
	insets := boxInsets(style)
	leftPad := insets.Left
	topPad := insets.Top
	rightPad := insets.Right
	availableW := rect.Width - leftPad - rightPad
	if availableW < 0 {
		availableW = 0
	}
	return leftPad, topPad, rightPad, availableW
}

func textLineX(rect Rect, style Style, leftPad, rightPad, availableW int, line string, font *ttfFont, charWidth int) int {
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	lineWidth := textWidthWithFont(line, font, charWidth)
	x := rect.X + leftPad
	align := TextAlignLeft
	if value, ok := resolveTextAlign(style.textAlign); ok {
		align = value
	}
	if align == TextAlignCenter && availableW > lineWidth {
		x = rect.X + leftPad + (availableW-lineWidth)/2
	} else if align == TextAlignRight {
		x = rect.X + rect.Width - rightPad - lineWidth
	}
	return x
}

func (element *Element) textPosition(rect Rect, style Style) (int, int) {
	font, metrics := fontAndMetricsForStyle(style)
	charWidth := metrics.width
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	leftPad, topPad, rightPad, availableW := textPaddingAndWidth(rect, style)
	text := element.text()
	line := text
	if text != "" {
		lines := element.wrapTextLinesCachedStyle(text, availableW, font, charWidth, style)
		if len(lines) > 0 {
			line = lines[0].text
		}
	}
	x := textLineX(rect, style, leftPad, rightPad, availableW, line, font, charWidth)
	y := rect.Y + topPad
	return x, y
}
