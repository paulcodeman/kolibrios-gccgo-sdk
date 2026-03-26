package main

import "testing"

func TestRegisterFontFamilyPathPrefersRegularVariant(t *testing.T) {
	registry := []fontFamilyEntry{}
	registry = registerFontFamilyPath(registry, "sourcesanspro", "assets/fonts/SourceSansPro-Bold.ttf")
	registry = registerFontFamilyPath(registry, "sourcesanspro", "assets/fonts/SourceSansPro-Regular.ttf")
	if len(registry) != 1 {
		t.Fatalf("unexpected registry size: %d", len(registry))
	}
	if got := registry[0].path; got != "assets/fonts/SourceSansPro-Regular.ttf" {
		t.Fatalf("family path mismatch: got %q want %q", got, "assets/fonts/SourceSansPro-Regular.ttf")
	}
}
