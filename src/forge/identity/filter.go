package identity

import (
	"slices"

	"vargames-name-gen/src/forge"
)

func filterCharacterPool(characters []forge.Character, genreID *int, gameGenres map[int][]int) []forge.Character {
	if genreID == nil {
		return append([]forge.Character(nil), characters...)
	}

	out := make([]forge.Character, 0, len(characters))
	for _, character := range characters {
		if characterMatchesGenre(character, *genreID, gameGenres) {
			out = append(out, character)
		}
	}
	return out
}

func characterMatchesGenre(character forge.Character, genreID int, gameGenres map[int][]int) bool {
	for _, gameID := range character.Games {
		if containsInt(gameGenres[gameID], genreID) {
			return true
		}
	}
	return false
}

func gameGenreIndex(games []forge.Game) map[int][]int {
	index := make(map[int][]int, len(games))
	for _, game := range games {
		index[game.ID] = append([]int(nil), game.Genres...)
	}
	return index
}

func characterNames(characters []forge.Character) []string {
	pool := make([]string, 0, len(characters))
	for _, character := range characters {
		if name := forge.NormalizeTitle(character.Name); name != "" {
			pool = append(pool, name)
		}
	}
	return pool
}

func containsInt(values []int, want int) bool {
	return slices.Contains(values, want)
}
