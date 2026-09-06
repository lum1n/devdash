package tui

import "testing"

func TestMdPrefixAndWrap(t *testing.T) {
	got := mdPrefixLine("hello", 0, "# ", true)
	if got != "# hello" {
		t.Fatalf("heading=%q", got)
	}
	got = mdPrefixLine("## hello", 0, "# ", true)
	if got != "# hello" {
		t.Fatalf("replace heading=%q", got)
	}
	got = mdWrap("say foo now", "foo", "**", "**", "bold")
	if got != "say **foo** now" {
		t.Fatalf("wrap=%q", got)
	}
}
