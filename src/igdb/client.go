package igdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"vargames-name-gen/src/config"
)

const (
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	// DefaultMaxRetries is the number of retries for transient failures (fetchers: wait as long as practical).
	DefaultMaxRetries = 10
	// DefaultRetryMinBackoff is the initial backoff after a retriable error.
	DefaultRetryMinBackoff = 1 * time.Second
	// DefaultRetryMaxBackoff caps backoff between retries.
	DefaultRetryMaxBackoff = 5 * time.Minute
	// DefaultMaxRetryDuration is the maximum total time spent in backoff before giving up.
	DefaultMaxRetryDuration = 1 * time.Hour
)

// Client performs authenticated requests to the IGDB API with retry logic.
type Client struct {
	cfg              config.Config
	http             *http.Client
	mu               sync.Mutex
	token            string
	expiry           time.Time
	retries          int
	minBackoff       time.Duration
	maxBackoff       time.Duration
	maxRetryDuration time.Duration // max total time in backoff before giving up
	log              Logger
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithRetries sets max retries (default 10).
func WithRetries(n int) ClientOption {
	return func(c *Client) { c.retries = n }
}

// WithBackoff sets min and max backoff for retries (exponential: double each time, capped at max).
func WithBackoff(min, max time.Duration) ClientOption {
	return func(c *Client) {
		c.minBackoff = min
		c.maxBackoff = max
	}
}

// WithMaxRetryDuration sets the maximum total time spent in backoff before failing (default 1h).
func WithMaxRetryDuration(d time.Duration) ClientOption {
	return func(c *Client) { c.maxRetryDuration = d }
}

// WithLogger sets an optional logger for progress and retry messages.
func WithLogger(l Logger) ClientOption {
	return func(c *Client) { c.log = l }
}

// NewClient builds an IGDB client from config. Optional opts customize retries/backoff.
func NewClient(cfg config.Config, opts ...ClientOption) *Client {
	c := &Client{
		cfg:              cfg,
		http:             &http.Client{Timeout: 30 * time.Second},
		retries:          DefaultMaxRetries,
		minBackoff:       DefaultRetryMinBackoff,
		maxBackoff:       DefaultRetryMaxBackoff,
		maxRetryDuration: DefaultMaxRetryDuration,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// MaxLimit returns the configured maximum request limit (for pagination).
func (c *Client) MaxLimit() int {
	if c.cfg.MaxLimit > 0 {
		return c.cfg.MaxLimit
	}
	return 500
}

// tokenResponse is the Twitch OAuth2 token response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// getToken returns a valid Bearer token, refreshing from Twitch if needed or using cfg.AccessToken.
func (c *Client) getToken(ctx context.Context) (string, error) {
	if c.cfg.AccessToken != "" {
		return c.cfg.AccessToken, nil
	}
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.expiry) {
		tok := c.token
		c.mu.Unlock()
		return tok, nil
	}
	c.mu.Unlock()

	if c.log != nil {
		c.log.Logf("token: refreshing from Twitch")
	}
	u, err := url.Parse(twitchTokenURL)
	if err != nil {
		return "", fmt.Errorf("parse token url: %w", err)
	}
	q := u.Query()
	q.Set("client_id", c.cfg.ClientID)
	q.Set("client_secret", c.cfg.ClientSecret)
	q.Set("grant_type", "client_credentials")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request status %d: %s", resp.StatusCode, string(body))
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	token := tr.AccessToken
	if token == "" {
		return "", fmt.Errorf("empty access_token in response")
	}
	expiry := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	// Refresh a bit early
	if tr.ExpiresIn > 60 {
		expiry = expiry.Add(-60 * time.Second)
	}
	c.mu.Lock()
	c.token = token
	c.expiry = expiry
	c.mu.Unlock()
	if c.log != nil {
		c.log.Logf("token: refreshed, expires in %ds", tr.ExpiresIn)
	}
	return token, nil
}

// Post sends a POST request to the given IGDB endpoint with body, using Bearer auth and retries.
// Retries use exponential backoff (double each time, capped at maxBackoff). Retrying stops when
// max retries are reached or when total time spent in backoff would exceed maxRetryDuration.
func (c *Client) Post(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	var lastErr error
	backoff := c.minBackoff
	deadline := time.Now().Add(c.maxRetryDuration)
	if c.maxRetryDuration <= 0 {
		deadline = time.Time{} // no deadline
	}
	for attempt := 0; attempt <= c.retries; attempt++ {
		resp, err := c.doPost(ctx, endpoint, body)
		if err != nil {
			lastErr = err
			if !isRetriable(err) || attempt == c.retries {
				return nil, lastErr
			}
			// Cap sleep so we don't exceed max retry duration
			sleep := backoff
			if !deadline.IsZero() && time.Now().Add(sleep).After(deadline) {
				remaining := time.Until(deadline)
				if remaining <= 0 {
					if c.log != nil {
						c.log.Logf("igdb %s: giving up after max retry duration", endpoint)
					}
					return nil, lastErr
				}
				sleep = remaining
			}
			if c.log != nil {
				c.log.Logf("igdb %s: retry %d/%d in %v after error: %v", endpoint, attempt+1, c.retries, sleep.Round(time.Second), err)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleep):
				backoff = nextBackoff(backoff, c.maxBackoff)
			}
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

func (c *Client) doPost(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}
	u := c.cfg.BaseURL + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Client-Id", c.cfg.ClientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/plain")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, &retriableError{status: resp.StatusCode, body: string(out)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("igdb %s: status %d: %s", endpoint, resp.StatusCode, string(out))
	}
	if c.log != nil {
		c.log.Logf("igdb %s: OK %d bytes", endpoint, len(out))
	}
	return out, nil
}

type retriableError struct {
	status int
	body   string
}

func (e *retriableError) Error() string {
	return fmt.Sprintf("retriable status %d: %s", e.status, e.body)
}

func isRetriable(err error) bool {
	_, ok := err.(*retriableError)
	return ok
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}
