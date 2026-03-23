package ui

// DisplayListOcclusionCulling skips paint for nodes and fragments that are
// fully covered by one later opaque rectangular item.
var DisplayListOcclusionCulling = true

func rectCoveredByAny(rect Rect, covers []Rect) bool {
	if rect.Empty() {
		return false
	}
	for _, cover := range covers {
		if rectContainsRect(cover, rect) {
			return true
		}
	}
	return false
}

func styleProvidesOpaqueBoxCover(style Style) bool {
	if !styleVisible(style) {
		return false
	}
	if resolveBorderRadius(style).Active() {
		return false
	}
	if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
		return false
	}
	if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
		return false
	}
	if _, ok := resolveColor(style.background); ok {
		return true
	}
	if _, ok := resolveGradient(style.gradient); ok {
		return true
	}
	return false
}

func nodeOpaqueCoverRect(node Node) (Rect, bool) {
	switch current := node.(type) {
	case *Element:
		if current == nil {
			return Rect{}, false
		}
		style := current.effectiveStyle()
		if !styleProvidesOpaqueBoxCover(style) {
			return Rect{}, false
		}
		rect := current.layoutRect
		if rect.Empty() {
			rect = current.Bounds()
		}
		if rect.Empty() {
			return Rect{}, false
		}
		return rect, true
	case *DocumentView:
		if current == nil {
			return Rect{}, false
		}
		style := current.effectiveStyle()
		if !styleProvidesOpaqueBoxCover(style) {
			return Rect{}, false
		}
		rect := current.layoutRect
		if rect.Empty() {
			rect = current.Bounds()
		}
		if rect.Empty() {
			return Rect{}, false
		}
		return rect, true
	default:
		return Rect{}, false
	}
}

func fragmentOpaqueCoverRect(fragment *Fragment) (Rect, bool) {
	if fragment == nil || fragment.Kind != FragmentKindBlock {
		return Rect{}, false
	}
	style := fragment.effectiveStyle()
	if !styleProvidesOpaqueBoxCover(style) {
		return Rect{}, false
	}
	if fragment.Bounds.Empty() {
		return Rect{}, false
	}
	return fragment.Bounds, true
}
