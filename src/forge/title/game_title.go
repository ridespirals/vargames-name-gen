package title

import (
	"errors"
	"fmt"

	"vargames-name-gen/src/forge"
)

// GameTitle generates a game-style title from the corpus using the configured strategy.
func (g *Generator) GameTitle(opts Options) (string, error) {
	if g == nil || g.corpus == nil {
		return "", errors.New("title: nil generator")
	}
	if len(g.corpus.Games) == 0 {
		return "", errors.New("title: no games in corpus")
	}

	pool := g.gameTitlePool(opts)
	existing := g.sourceTitleSet()
	rng := forge.NewRand(opts.Seed)
	strategy := strategyName(opts)

	const maxAttempts = 32
	var lastErr error
	for range maxAttempts {
		title, err := generateTitle(rng, pool, existing, strategy)
		if err != nil {
			lastErr = err
			continue
		}
		if title != "" {
			return title, nil
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("title: %w", lastErr)
	}
	return "", errors.New("title: could not generate acceptable game title")
}

func (g *Generator) gameTitlePool(opts Options) []string {
	filtered := filterGamePool(g.corpus.Games, opts.GenreID)
	pool := gameNames(filtered)
	if len(pool) > 0 {
		return pool
	}
	return gameNames(g.corpus.Games)
}

func (g *Generator) sourceTitleSet() map[string]struct{} {
	titles := make([]string, 0, len(g.corpus.Games)+len(g.corpus.AlternativeNames)+len(g.corpus.Collections))
	for _, game := range g.corpus.Games {
		titles = append(titles, game.Name)
	}
	for _, alt := range g.corpus.AlternativeNames {
		titles = append(titles, alt.Name)
	}
	for _, col := range g.corpus.Collections {
		titles = append(titles, col.Name)
	}
	return forge.SourceTitleSet(titles...)
}
