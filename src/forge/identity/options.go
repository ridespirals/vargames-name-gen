package identity

// Options configures identity generation.
type Options struct {
	// GenreID filters characters linked to games with this IGDB genre ID.
	GenreID *int
	// Strategy selects the generation approach: "concat" (default) or "pick" (reservoir sample, unmutated).
	Strategy string
	// Seed controls determinism; 0 uses a random seed.
	Seed int64
}
