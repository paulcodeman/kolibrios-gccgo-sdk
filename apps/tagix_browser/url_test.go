package main

import (
	"strings"
	"testing"
)

func TestNormalizeURLCanonicalizesBuiltinPages(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "welcome",
			in:   "TAGIX://WELCOME?Query=1#Top",
			want: "tagix://welcome?Query=1#Top",
		},
		{
			name: "legacy home alias",
			in:   "about:tagix",
			want: defaultURL,
		},
		{
			name: "legacy forms alias",
			in:   "about:forms?source=demo",
			want: tagixFormsURL + "?source=demo",
		},
	}

	for _, test := range tests {
		if got := normalizeURL(test.in); got != test.want {
			t.Fatalf("%s: normalizeURL(%q) = %q, want %q", test.name, test.in, got, test.want)
		}
	}
}

func TestBuiltinPageSourceServesTagixPages(t *testing.T) {
	title, status, body, ok := builtinPageSource(defaultURL + "?from=test")
	if !ok {
		t.Fatalf("expected builtin welcome page")
	}
	if title != "Tagix Browser" {
		t.Fatalf("title mismatch: got %q", title)
	}
	if status != "Built-in page" {
		t.Fatalf("status mismatch: got %q", status)
	}
	if !strings.Contains(body, "tagix://forms") {
		t.Fatalf("welcome page should link to tagix://forms")
	}
}

func TestBuiltinPageSourceReturnsBuiltin404ForUnknownTagixPage(t *testing.T) {
	title, status, body, ok := builtinPageSource("tagix://missing")
	if !ok {
		t.Fatalf("expected builtin 404 page")
	}
	if title != "404 Not Found" || status != "404 Not Found" {
		t.Fatalf("unexpected builtin 404 metadata: title=%q status=%q", title, status)
	}
	if !strings.Contains(body, "tagix://missing") {
		t.Fatalf("missing page body should mention requested URL")
	}
}
