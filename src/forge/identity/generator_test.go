package identity

import (
	"testing"

	"vargames-name-gen/src/forge"
)

func TestNew(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	gen := New(corpus)
	if gen.Corpus() != corpus {
		t.Fatal("expected generator to hold corpus reference")
	}
	if len(gen.Corpus().Characters) != 10 {
		t.Fatalf("got %d characters, want 10", len(gen.Corpus().Characters))
	}
}
