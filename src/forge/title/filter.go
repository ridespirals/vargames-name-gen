package title

import (
	"slices"

	"vargames-name-gen/src/forge"
)

func filterGamePool(games []forge.Game, genreID *int) []forge.Game {
	if genreID == nil {
		return append([]forge.Game(nil), games...)
	}

	out := make([]forge.Game, 0, len(games))
	for _, game := range games {
		if !containsInt(game.Genres, *genreID) {
			continue
		}
		out = append(out, game)
	}
	return out
}

func gameNames(games []forge.Game) []string {
	pool := make([]string, 0, len(games))
	for _, game := range games {
		if name := forge.NormalizeTitle(game.Name); name != "" {
			pool = append(pool, name)
		}
	}
	return pool
}

func containsInt(values []int, want int) bool {
	return slices.Contains(values, want)
}
