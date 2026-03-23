package ui

func retainedLayerStyleChanged(oldStyle Style, newStyle Style) bool {
	return !equalBackgroundAttachmentPtr(oldStyle.backgroundAttachment, newStyle.backgroundAttachment) ||
		!equalContainPtr(oldStyle.contain, newStyle.contain) ||
		!equalWillChangePtr(oldStyle.willChange, newStyle.willChange)
}

func (element *Element) styleTransitionNeedsDirtyNode(oldStyle Style, newStyle Style) bool {
	if element == nil {
		return true
	}
	if retainedLayerStyleChanged(oldStyle, newStyle) {
		return true
	}
	if styleChangeAffectsLayout(element, oldStyle, newStyle) {
		return true
	}
	oldKey := element.renderKeyFor(oldStyle)
	newKey := element.renderKeyFor(newStyle)
	if !equalDisplayPtr(oldKey.display, newKey.display) {
		return true
	}
	if !clipVisualKeyEqual(oldKey.visual, newKey.visual) {
		return true
	}
	return false
}

func (element *Element) invalidateStyleTransition(oldStyle Style, newStyle Style) bool {
	if element == nil {
		return false
	}
	if element.window == nil {
		element.markDirty()
		return true
	}
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	oldVisual := element.visualBoundsFor(rect, oldStyle)
	newVisual := element.visualBoundsFor(rect, newStyle)
	dirty := oldVisual
	if dirty.Empty() {
		dirty = newVisual
	} else if !newVisual.Empty() {
		dirty = UnionRect(dirty, newVisual)
	}
	if dirty.Empty() {
		dirty = rect
	}
	if dirty.Empty() {
		return false
	}
	element.renderKey = element.renderKeyFor(newStyle)
	element.window.noteRetainedLayerDirty(element, dirty)
	element.window.InvalidateVisualContent(dirty)
	return true
}

func (element *Element) applyEffectiveStyleChange(oldStyle Style, newStyle Style) bool {
	if element == nil {
		return false
	}
	if retainedLayerStyleChanged(oldStyle, newStyle) {
		element.invalidateRetainedLayerState()
	}
	if !styleChangeAffectsLayout(element, oldStyle, newStyle) &&
		styleVisualKeyEqual(visualKeyFor(oldStyle), visualKeyFor(newStyle)) {
		return false
	}
	element.invalidateBoundsCache()
	if !element.styleTransitionNeedsDirtyNode(oldStyle, newStyle) {
		return element.invalidateStyleTransition(oldStyle, newStyle)
	}
	element.markDirty()
	return true
}

func (element *Element) invalidateRetainedLayerState() {
	if element == nil {
		return
	}
	element.invalidateRetainedLayerChain()
	if element.window != nil && !element.window.LockRenderList {
		element.window.renderListValid = false
	}
}

// UpdateStyle mutates the base style and redraws only if the effective style changes.
func (element *Element) UpdateStyle(update func(style *Style)) bool {
	if element == nil || update == nil {
		return false
	}
	oldStyle := element.effectiveStyle()
	update(&element.Style)
	element.invalidateEffectiveStyleCache()
	newStyle := element.effectiveStyle()
	if retainedLayerStyleChanged(oldStyle, newStyle) {
		element.invalidateRetainedLayerState()
	}
	if !styleChangeAffectsLayout(element, oldStyle, newStyle) &&
		styleVisualKeyEqual(visualKeyFor(oldStyle), visualKeyFor(newStyle)) {
		return false
	}
	element.markDirty()
	return true
}

// UpdateHoverStyle mutates the hover style and redraws only if it affects current hover state.
func (element *Element) UpdateHoverStyle(update func(style *Style)) bool {
	if element == nil || update == nil {
		return false
	}
	wasHovered := element.hovered
	oldStyle := element.effectiveStyle()
	update(&element.StyleHover)
	if !wasHovered {
		return false
	}
	element.invalidateEffectiveStyleCache()
	newStyle := element.effectiveStyle()
	return element.applyEffectiveStyleChange(oldStyle, newStyle)
}

// UpdateActiveStyle mutates the active style and redraws only if it affects current active state.
func (element *Element) UpdateActiveStyle(update func(style *Style)) bool {
	if element == nil || update == nil {
		return false
	}
	wasActive := element.active
	oldStyle := element.effectiveStyle()
	update(&element.StyleActive)
	if !wasActive {
		return false
	}
	element.invalidateEffectiveStyleCache()
	newStyle := element.effectiveStyle()
	return element.applyEffectiveStyleChange(oldStyle, newStyle)
}

// UpdateFocusStyle mutates the focus style and redraws only if it affects current focus state.
func (element *Element) UpdateFocusStyle(update func(style *Style)) bool {
	if element == nil || update == nil {
		return false
	}
	wasFocused := element.focused
	oldStyle := element.effectiveStyle()
	update(&element.StyleFocus)
	if !wasFocused {
		return false
	}
	element.invalidateEffectiveStyleCache()
	newStyle := element.effectiveStyle()
	if retainedLayerStyleChanged(oldStyle, newStyle) {
		element.invalidateRetainedLayerState()
	}
	if !styleChangeAffectsLayout(element, oldStyle, newStyle) &&
		styleVisualKeyEqual(visualKeyFor(oldStyle), visualKeyFor(newStyle)) {
		return false
	}
	element.markDirty()
	return true
}

func styleChangeAffectsLayout(element *Element, oldStyle Style, newStyle Style) bool {
	if element == nil {
		return true
	}
	container := element.layoutContainer()
	if container.Width == 0 && container.Height == 0 {
		return true
	}
	oldKey := element.layoutKeyFor(oldStyle, container)
	newKey := element.layoutKeyFor(newStyle, container)
	return !elementLayoutKeyEqual(oldKey, newKey)
}

func (element *Element) SetHover(hover bool) bool {
	if element == nil || element.hovered == hover {
		return false
	}
	oldStyle := element.effectiveStyle()
	element.hovered = hover
	element.invalidateEffectiveStyleCache()
	if element.StyleHover.IsZero() {
		return false
	}
	newStyle := element.effectiveStyle()
	return element.applyEffectiveStyleChange(oldStyle, newStyle)
}

func (element *Element) SetActive(active bool) bool {
	if element == nil || element.active == active {
		return false
	}
	oldStyle := element.effectiveStyle()
	element.active = active
	element.invalidateEffectiveStyleCache()
	if element.StyleActive.IsZero() {
		return false
	}
	newStyle := element.effectiveStyle()
	return element.applyEffectiveStyleChange(oldStyle, newStyle)
}

func (element *Element) SetFocus(focus bool) bool {
	if element == nil || element.focused == focus {
		return false
	}
	oldStyle := element.effectiveStyle()
	element.focused = focus
	element.invalidateEffectiveStyleCache()
	if element.isTextInput() {
		if focus {
			textLen := len(element.text())
			if element.caret < 0 {
				element.caret = 0
			} else if element.caret > textLen {
				element.caret = textLen
			}
			element.desiredCol = -1
		}
		if !focus {
			element.selectAnchor = element.caret
			element.dragMode = textDragNone
			element.dragMoved = false
		} else if !element.hasSelection() {
			element.selectAnchor = element.caret
		}
	}
	newStyle := element.effectiveStyle()
	if retainedLayerStyleChanged(oldStyle, newStyle) {
		element.invalidateRetainedLayerState()
	}
	if focus {
		element.dispatchTargetOnlyEvent(EventFocus)
		element.dispatchFocusTransitionEvent(EventFocusIn, true)
	} else {
		element.dispatchTargetOnlyEvent(EventBlur)
		element.dispatchFocusTransitionEvent(EventFocusOut, true)
	}
	element.markDirty()
	return true
}

func (element *Element) Focused() bool {
	if element == nil {
		return false
	}
	return element.focused
}

func (element *Element) effectiveStyle() Style {
	if element == nil {
		return Style{}
	}
	if element.effectiveStyleValid {
		return element.effectiveStyleCache
	}
	style := element.Style
	if element.focused && !element.StyleFocus.IsZero() {
		style = mergeStyle(style, element.StyleFocus)
	}
	if element.active && !element.StyleActive.IsZero() {
		style = mergeStyle(style, element.StyleActive)
	} else if element.hovered && !element.StyleHover.IsZero() {
		style = mergeStyle(style, element.StyleHover)
	}
	element.effectiveStyleCache = style
	element.effectiveStyleValid = true
	return style
}

func (element *Element) updateRenderKey(style Style) {
	if element == nil {
		return
	}
	key := element.renderKeyFor(style)
	if !elementRenderKeyEqual(key, element.renderKey) {
		if element.window != nil && (!clipVisualKeyEqual(key.visual, element.renderKey.visual) ||
			!equalDisplayPtr(key.display, element.renderKey.display)) {
			if !element.window.LockRenderList {
				element.window.renderListValid = false
			}
		}
		element.dirty = true
		element.renderKey = key
	}
}

func (element *Element) renderKeyFor(style Style) elementRenderKey {
	var display *DisplayMode
	if value, ok := resolveDisplay(style.display); ok {
		v := value
		display = &v
	}
	return elementRenderKey{
		kind:    element.kind,
		text:    element.text(),
		checked: element.checked,
		value:   element.value,
		min:     element.minValue,
		max:     element.maxValue,
		display: display,
		focus:   elementShowsDefaultFocusRing(element),
		visual:  visualKeyFor(style),
	}
}

func resolveBorderRadius(style Style) CornerRadii {
	if radii, ok := resolveCornerRadii(style.borderRadius); ok && radii != nil {
		value := *radii
		if value.TopLeft < 0 {
			value.TopLeft = 0
		}
		if value.TopRight < 0 {
			value.TopRight = 0
		}
		if value.BottomRight < 0 {
			value.BottomRight = 0
		}
		if value.BottomLeft < 0 {
			value.BottomLeft = 0
		}
		return value
	}
	return CornerRadii{}
}

func elementRenderKeyEqual(a elementRenderKey, b elementRenderKey) bool {
	return a.kind == b.kind &&
		a.text == b.text &&
		a.checked == b.checked &&
		a.value == b.value &&
		a.min == b.min &&
		a.max == b.max &&
		equalDisplayPtr(a.display, b.display) &&
		a.focus == b.focus &&
		styleVisualKeyEqual(a.visual, b.visual)
}

func clipVisualKeyEqual(a styleVisualKey, b styleVisualKey) bool {
	return equalOverflowPtr(a.overflow, b.overflow) &&
		equalOverflowPtr(a.overflowX, b.overflowX) &&
		equalOverflowPtr(a.overflowY, b.overflowY) &&
		equalSpacingPtr(a.padding, b.padding) &&
		equalIntPtr(a.borderWidth, b.borderWidth)
}

func elementLayoutKeyEqual(a elementLayoutKey, b elementLayoutKey) bool {
	return a.kind == b.kind &&
		equalPositionPtr(a.position, b.position) &&
		equalDisplayPtr(a.display, b.display) &&
		a.containerX == b.containerX &&
		a.containerY == b.containerY &&
		a.containerW == b.containerW &&
		a.containerH == b.containerH &&
		equalIntPtr(a.left, b.left) &&
		equalIntPtr(a.top, b.top) &&
		equalIntPtr(a.right, b.right) &&
		equalIntPtr(a.bottom, b.bottom) &&
		a.width == b.width &&
		a.height == b.height &&
		equalIntPtr(a.styleWidth, b.styleWidth) &&
		equalIntPtr(a.styleHeight, b.styleHeight) &&
		equalSpacingPtr(a.margin, b.margin) &&
		a.flowSet == b.flowSet &&
		a.flowX == b.flowX &&
		a.flowY == b.flowY
}
