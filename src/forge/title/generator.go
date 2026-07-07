package title

import "vargames-name-gen/src/forge"

// Generator generates title-family artifacts from a loaded corpus.
type Generator struct {
	corpus *forge.Corpus
}

// New returns a title generator backed by corpus.
func New(corpus *forge.Corpus) *Generator {
	return &Generator{corpus: corpus}
}

// Corpus returns the backing corpus, or nil if the generator is nil.
func (g *Generator) Corpus() *forge.Corpus {
	if g == nil {
		return nil
	}
	return g.corpus
}
