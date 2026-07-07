package forge

import (
	"testing"
)

func TestNormalizeTitle_collapsesWhitespace(t *testing.T) {
	got := NormalizeTitle("  Dark   Souls  ")
	if got != "Dark Souls" {
		t.Fatalf("got %q", got)
	}
}

func TestTokenizeTitle_zeldaExample(t *testing.T) {
	tokens := TokenizeTitle("The Legend of Zelda: Breath of the Wild")
	want := []string{"The", "Legend", "of", "Zelda", "Breath", "of", "the", "Wild"}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens %v, want %d", len(tokens), tokens, len(want))
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token[%d] = %q, want %q (all: %v)", i, tokens[i], want[i], tokens)
		}
	}
}

func TestRejectTitle(t *testing.T) {
	existing := SourceTitleSet("Dark Souls")
	tests := []struct {
		name      string
		candidate string
		want      bool
	}{
		{"empty", "", true},
		{"too short", "ab", true},
		{"duplicate", "Dark Souls", true},
		{"duplicate case", "dark souls", true},
		{"all caps noise", "SUPER LOUD GAME", true},
		{"acceptable", "Dark Ring", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RejectTitle(tc.candidate, existing); got != tc.want {
				t.Fatalf("RejectTitle(%q) = %v, want %v", tc.candidate, got, tc.want)
			}
		})
	}
}

func TestNewRand_deterministic(t *testing.T) {
	a := NewRand(42)
	b := NewRand(42)
	if a.IntN(100000) != b.IntN(100000) {
		t.Fatal("expected same seed to produce same sequence")
	}
}
