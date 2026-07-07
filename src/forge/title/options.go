package title

// Options configures title generation.
type Options struct {
	// Strategy selects the generation approach: "concat" (default) or "pick" (reservoir sample, unmutated).
	Strategy string
	// Seed controls determinism; 0 uses a random seed.
	Seed int64
}
