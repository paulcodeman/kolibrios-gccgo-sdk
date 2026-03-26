package main

import "testing"

func TestInlinePieceBuilderPreservesLeadingWhitespaceBetweenInlineNodes(t *testing.T) {
	builder := inlinePieceBuilder{}
	style := inlineTextStyle{}

	builder.appendText("Tagix Browser", style)
	builder.appendText(" is a web browser", style)

	got := inlinePieceTexts(builder.pieces)
	want := []string{"Tagix", " ", "Browser", " ", "is", " ", "a", " ", "web", " ", "browser"}
	if len(got) != len(want) {
		t.Fatalf("piece count mismatch: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("piece %d mismatch: got %q want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func inlinePieceTexts(pieces []inlinePiece) []string {
	texts := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		texts = append(texts, piece.text)
	}
	return texts
}
