package main

import (
	"testing"
	"ui"
)

func TestApplyInlineStyleRuleMapsInsetBorderShadowToOutline(t *testing.T) {
	style := ui.Style{}

	applyInlineStyleRule(&style, "box-shadow", "inset 0 0 0 1px rgba(0,0,0,0.25)")

	width, ok := style.GetOutlineWidth()
	if !ok || width != 1 {
		t.Fatalf("outline width mismatch: got %d set=%v", width, ok)
	}
	offset, ok := style.GetOutlineOffset()
	if !ok || offset != -1 {
		t.Fatalf("outline offset mismatch: got %d set=%v", offset, ok)
	}
	if _, ok := style.GetOutlineColor(); !ok {
		t.Fatalf("expected outline color from inset box shadow")
	}
	if _, ok := style.GetBorderWidth(); ok {
		t.Fatalf("did not expect border width from inset box shadow")
	}
	if _, ok := style.GetShadow(); ok {
		t.Fatalf("did not expect regular shadow for inset border shadow")
	}
}

func TestApplyInlineStyleRuleParsesWidthPercent(t *testing.T) {
	style := ui.Style{}

	applyInlineStyleRule(&style, "width", "10.638%")

	value, ok := style.GetWidthPercent()
	if !ok || value != 10638 {
		t.Fatalf("width percent mismatch: got %d set=%v", value, ok)
	}
	if _, ok := style.GetWidth(); ok {
		t.Fatalf("did not expect pixel width when percent width is used")
	}
}
