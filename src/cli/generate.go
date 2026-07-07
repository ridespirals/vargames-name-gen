package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"vargames-name-gen/src/forge"
	"vargames-name-gen/src/forge/identity"
	"vargames-name-gen/src/forge/title"
)

// TitleConfig holds options for generating game-style titles.
type TitleConfig struct {
	DataDir  string
	Seed     int64
	Count    int
	Strategy string
}

// IdentityConfig holds options for generating character-style names.
type IdentityConfig struct {
	DataDir  string
	Seed     int64
	Count    int
	Strategy string
	GenreID  *int
}

// GenerateIdentities loads the corpus and returns one or more forged character names.
func GenerateIdentities(cfg IdentityConfig) ([]string, error) {
	if cfg.Count < 1 {
		return nil, errors.New("count must be at least 1")
	}
	dir := ResolveDataDir(cfg.DataDir)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		return nil, fmt.Errorf("load corpus from %s: %w", dir, err)
	}

	gen := identity.New(corpus)
	opts := identity.Options{
		GenreID:  cfg.GenreID,
		Strategy: cfg.Strategy,
		Seed:     cfg.Seed,
	}

	out := make([]string, 0, cfg.Count)
	seen := make(map[string]struct{}, cfg.Count)
	for i := 0; i < cfg.Count; i++ {
		attemptOpts := opts
		if opts.Seed != 0 {
			attemptOpts.Seed = opts.Seed + int64(i)
		}
		name, err := gen.CharacterName(attemptOpts)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[name]; dup {
			i--
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// GenerateTitles loads the corpus and returns one or more forged game titles.
func GenerateTitles(cfg TitleConfig) ([]string, error) {
	if cfg.Count < 1 {
		return nil, errors.New("count must be at least 1")
	}
	dir := ResolveDataDir(cfg.DataDir)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		return nil, fmt.Errorf("load corpus from %s: %w", dir, err)
	}

	gen := title.New(corpus)
	opts := title.Options{
		Strategy: cfg.Strategy,
		Seed:     cfg.Seed,
	}

	out := make([]string, 0, cfg.Count)
	seen := make(map[string]struct{}, cfg.Count)
	for i := 0; i < cfg.Count; i++ {
		attemptOpts := opts
		if opts.Seed != 0 {
			attemptOpts.Seed = opts.Seed + int64(i)
		}
		name, err := gen.GameTitle(attemptOpts)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[name]; dup {
			i--
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// RunGenerate runs forge generation subcommands: title, character (future).
func RunGenerate(args []string) error {
	if len(args) == 0 {
		PrintGenerateUsage(os.Stderr)
		return errors.New("missing generate subcommand")
	}
	switch args[0] {
	case "help", "-h", "--help":
		PrintGenerateUsage(os.Stdout)
		return nil
	case "title", "game":
		return runTitleCommand(args[1:])
	case "character", "identity":
		return runCharacterCommand(args[1:])
	default:
		return fmt.Errorf("unknown generate subcommand %q (try: title)", args[0])
	}
}

func runCharacterCommand(args []string) error {
	fs := flag.NewFlagSet("character", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataDir := fs.String("data-dir", "", "corpus directory (default: testdata/corpus or data/)")
	seed := fs.Int64("seed", 0, "random seed (0 = non-deterministic)")
	count := fs.Int("count", 1, "number of names to generate")
	strategy := fs.String("strategy", "", "generation strategy: concat (default) or pick")
	genre := fs.Int("genre", 0, "IGDB genre ID to filter source characters (0 = all)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := IdentityConfig{
		DataDir:  *dataDir,
		Seed:     *seed,
		Count:    *count,
		Strategy: *strategy,
	}
	if *genre > 0 {
		g := *genre
		cfg.GenreID = &g
	}

	names, err := GenerateIdentities(cfg)
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

func runTitleCommand(args []string) error {
	fs := flag.NewFlagSet("title", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataDir := fs.String("data-dir", "", "corpus directory (default: testdata/corpus or data/)")
	seed := fs.Int64("seed", 0, "random seed (0 = non-deterministic)")
	count := fs.Int("count", 1, "number of titles to generate")
	strategy := fs.String("strategy", "", "generation strategy: concat (default) or pick")
	if err := fs.Parse(args); err != nil {
		return err
	}

	names, err := GenerateTitles(TitleConfig{
		DataDir:  *dataDir,
		Seed:     *seed,
		Count:    *count,
		Strategy: *strategy,
	})
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

// PrintGenerateUsage writes generate subcommand help to w.
func PrintGenerateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  generate title [flags]")
	fmt.Fprintln(w, "  generate character [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags for title and character:")
	fmt.Fprintln(w, "  -data-dir   corpus directory (default: testdata/corpus or data/)")
	fmt.Fprintln(w, "  -seed       random seed (0 = random)")
	fmt.Fprintln(w, "  -count      number of results (default 1)")
	fmt.Fprintln(w, "  -strategy   concat (default) or pick")
	fmt.Fprintln(w, "  -genre      IGDB genre ID filter (character only; 0 = all)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Environment:")
	fmt.Fprintf(w, "  %s   default corpus directory\n", EnvDataDir)
}

// PrintForgeUsage writes help for the forge binary (no generate prefix).
func PrintForgeUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  forge title [flags]")
	fmt.Fprintln(w, "  forge character [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags for title and character:")
	fmt.Fprintln(w, "  -data-dir   corpus directory (default: testdata/corpus or data/)")
	fmt.Fprintln(w, "  -seed       random seed (0 = random)")
	fmt.Fprintln(w, "  -count      number of results (default 1)")
	fmt.Fprintln(w, "  -strategy   concat (default) or pick")
	fmt.Fprintln(w, "  -genre      IGDB genre ID filter (character only; 0 = all)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "No IGDB credentials required — only local JSON corpus files.")
}

// RunForge is like RunGenerate but for the forge binary (subcommand without "generate" prefix).
func RunForge(args []string) error {
	if len(args) == 0 {
		PrintForgeUsage(os.Stderr)
		return errors.New("missing subcommand")
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		PrintForgeUsage(os.Stdout)
		return nil
	}
	return RunGenerate(args)
}

// IsGenerateCommand reports whether args invoke generation (no IGDB config needed).
func IsGenerateCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return strings.EqualFold(args[0], "generate")
}
