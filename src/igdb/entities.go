package igdb

// Entity identifies an IGDB API entity (endpoint path segment).
type Entity string

const (
	EntityGames            Entity = "games"
	EntityCharacters       Entity = "characters"
	EntityGenres           Entity = "genres"
	EntityPlatforms        Entity = "platforms"
	EntityCollections      Entity = "collections"
	EntityCompanies        Entity = "companies"
	EntityAlternativeNames Entity = "alternative_names"
)

// DefaultQueryPrefix is the default Apicalypse query fragment used before the paging
// clauses (limit/offset) are appended.
//
// Apicalypse queries are statement-based and IGDB typically expects semicolon-separated
// clauses, so this string includes a trailing ';'.
const DefaultQueryPrefix = "fields *;"

// QueryPrefixForEntity returns the Apicalypse query text that will be inserted before
// paging clauses.
//
// For now, every entity uses the same default, but keeping this as a hook makes it easy
// to customize per-entity later (e.g. fetch only id/summary for cheap pagination).
func QueryPrefixForEntity(_ Entity) string { return DefaultQueryPrefix }

// AllEntities returns every entity type supported by the fetcher (matches Bruno requests).
func AllEntities() []Entity {
	return []Entity{
		EntityGames,
		EntityCharacters,
		EntityGenres,
		EntityPlatforms,
		EntityCollections,
		EntityCompanies,
		EntityAlternativeNames,
	}
}

// ValidEntity returns true if s is a known entity name (e.g. "games", "alternative_names").
func ValidEntity(s string) bool {
	for _, e := range AllEntities() {
		if string(e) == s {
			return true
		}
	}
	return false
}
