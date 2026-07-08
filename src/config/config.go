package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds API credentials and options loaded from the environment.
// Do not log or print these values.
type Config struct {
	// ClientID is the Twitch application client ID (for IGDB OAuth).
	ClientID string
	// ClientSecret is the Twitch application client secret.
	ClientSecret string
	// BaseURL is the IGDB API base URL (default: https://api.igdb.com/v4).
	BaseURL string
	// AccessToken is an optional pre-generated IGDB access token.
	// Most callers should prefer using client credentials flow instead.
	AccessToken string
	// MaxLimit is the default "limit" (page size) to use in IGDB queries.
	// IGDB allows up to 500 results per page; we default to 500 unless overridden.
	MaxLimit int
	// MaxConcurrent is the default number of parallel page requests per entity fetch.
	MaxConcurrent int
	// FetchProfile is the default Apicalypse fetch profile (full, minimal, checksum).
	FetchProfile string
	// Verbose enables progress logging for IGDB client and fetchers when true.
	// Set via IGDB_VERBOSE or VARGAMES_VERBOSE (1, true, on), or -verbose flag.
	Verbose bool
}

// Env variable names. Use these when setting up your environment.
const (
	EnvClientID      = "IGDB_CLIENT_ID"
	EnvClientSecret  = "IGDB_CLIENT_SECRET"
	EnvBaseURL       = "IGDB_BASE_URL"
	EnvAccessToken   = "IGDB_ACCESS_TOKEN"
	EnvMaxLimit      = "IGDB_MAX_LIMIT"
	EnvMaxConcurrent = "IGDB_MAX_CONCURRENT"
	EnvFetchProfile  = "IGDB_FETCH_PROFILE"
	EnvVerbose       = "IGDB_VERBOSE"
	EnvVerboseAlt    = "VARGAMES_VERBOSE"
)

// DefaultBaseURL is the default IGDB API base URL.
const DefaultBaseURL = "https://api.igdb.com/v4"

// DefaultMaxLimit is the default IGDB "limit" (page size) used when none is provided.
// IGDB's maximum (and our default) is 500.
const DefaultMaxLimit = 500

// DefaultMaxConcurrent is the default number of parallel page requests per entity fetch.
const DefaultMaxConcurrent = 4

// DefaultFetchProfile is the default Apicalypse field profile for fetches.
const DefaultFetchProfile = "minimal"

// LoadEnv reads a .env file from the current working directory and sets
// environment variables for each KEY=VALUE line. Variables already set
// in the environment are left unchanged. Lines that are empty or start
// with # are skipped. The file is optional; if it does not exist, LoadEnv
// returns nil. Malformed lines cause an error with filename and line number.
func LoadEnv() error {
	const envFile = ".env"

	f, err := os.Open(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", envFile, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		before, after, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: invalid line (missing KEY=VALUE)", envFile, lineNum)
		}
		key := strings.TrimSpace(before)
		value := strings.TrimSpace(after)
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", envFile, lineNum)
		}
		// Remove optional surrounding quotes
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("setenv %s: %w", key, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", envFile, err)
	}
	return nil
}

// Load reads configuration from environment variables. If a .env file
// exists in the current working directory, it is loaded first (see LoadEnv).
// It returns an error if required credentials are missing.
func Load() (Config, error) {
	if err := LoadEnv(); err != nil {
		return Config{}, fmt.Errorf("load env: %w", err)
	}
	clientID := os.Getenv(EnvClientID)
	clientSecret := os.Getenv(EnvClientSecret)
	if clientID == "" {
		return Config{}, fmt.Errorf("missing required environment variable %s", EnvClientID)
	}
	if clientSecret == "" {
		return Config{}, fmt.Errorf("missing required environment variable %s", EnvClientSecret)
	}
	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	accessToken := os.Getenv(EnvAccessToken)
	maxLimit := parsePositiveInt(os.Getenv(EnvMaxLimit), DefaultMaxLimit)
	maxConcurrent := parsePositiveInt(os.Getenv(EnvMaxConcurrent), DefaultMaxConcurrent)
	fetchProfile := strings.TrimSpace(os.Getenv(EnvFetchProfile))
	if fetchProfile == "" {
		fetchProfile = DefaultFetchProfile
	}
	verbose := parseVerbose(os.Getenv(EnvVerbose)) || parseVerbose(os.Getenv(EnvVerboseAlt))
	return Config{
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		BaseURL:       baseURL,
		AccessToken:   accessToken,
		MaxLimit:      maxLimit,
		MaxConcurrent: maxConcurrent,
		FetchProfile:  fetchProfile,
		Verbose:       verbose,
	}, nil
}

// parsePositiveInt parses s as a positive int; returns defaultVal if s is empty or invalid.
func parsePositiveInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return defaultVal
}

// parseVerbose returns true for 1, true, on (case-insensitive).
func parseVerbose(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}
