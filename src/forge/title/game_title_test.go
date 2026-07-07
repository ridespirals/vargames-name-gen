package title

import (
	"testing"

	"vargames-name-gen/src/forge"
)

func TestGameTitle_deterministic(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)

	a, err := gen.GameTitle(Options{Seed: 99})
	if err != nil {
		t.Fatalf("GameTitle: %v", err)
	}
	b, err := gen.GameTitle(Options{Seed: 99})
	if err != nil {
		t.Fatalf("GameTitle: %v", err)
	}
	if a != b {
		t.Fatalf("same seed produced different titles: %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("expected non-empty title")
	}
}

func TestGameTitle_notExactCorpusMatch(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)
	existing := gen.sourceTitleSet()

	for seed := int64(1); seed <= 20; seed++ {
		title, err := gen.GameTitle(Options{Seed: seed})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if forge.RejectTitle(title, existing) {
			t.Fatalf("seed %d produced corpus duplicate or rejected title %q", seed, title)
		}
	}
}

func TestGameTitle_nilGenerator(t *testing.T) {
	var gen *Generator
	if _, err := gen.GameTitle(Options{}); err == nil {
		t.Fatal("expected error for nil generator")
	}
}

func TestGameTitle_emptyCorpus(t *testing.T) {
	gen := New(&forge.Corpus{})
	if _, err := gen.GameTitle(Options{}); err == nil {
		t.Fatal("expected error for empty corpus")
	}
}

func TestGameTitle_unknownStrategy(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)
	if _, err := gen.GameTitle(Options{Strategy: "markov", Seed: 1}); err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}
