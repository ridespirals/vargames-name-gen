package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"vargames-name-gen/src/config"
)

type scriptedRoundTripper struct {
	t *testing.T

	statuses []int
	bodies   []string

	attempts int

	checkHeaders func(req *http.Request, attempt int)
}

func (rt *scriptedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// If context is canceled, behave like http would and let the client wrap it.
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	attempt := rt.attempts
	rt.attempts++

	if rt.checkHeaders != nil {
		rt.checkHeaders(req, attempt)
	}

	status := 500
	body := ""
	if attempt < len(rt.statuses) {
		status = rt.statuses[attempt]
	}
	if attempt < len(rt.bodies) {
		body = rt.bodies[attempt]
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

type scriptedRoute struct {
	statuses []int
	bodies   []string

	checkHeaders func(req *http.Request, attempt int)
	checkURL      func(req *http.Request)
}

type pathScriptedRoundTripper struct {
	t *testing.T

	routes map[string]scriptedRoute // keyed by req.URL.Path

	mu       sync.Mutex
	attempts map[string]int
}

func (rt *pathScriptedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	path := req.URL.Path

	rt.mu.Lock()
	attempt := rt.attempts[path]
	rt.attempts[path] = attempt + 1
	route, ok := rt.routes[path]
	rt.mu.Unlock()

	if !ok {
		rt.t.Fatalf("unexpected request path %q", path)
	}

	if route.checkURL != nil {
		route.checkURL(req)
	}
	if route.checkHeaders != nil {
		route.checkHeaders(req, attempt)
	}

	status := 500
	body := ""
	if attempt < len(route.statuses) {
		status = route.statuses[attempt]
	}
	if attempt < len(route.bodies) {
		body = route.bodies[attempt]
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestPost_Success(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}

	rt := &scriptedRoundTripper{
		t: t,
		statuses: []int{http.StatusOK},
		bodies:   []string{`[{"id":1,"name":"Game One"}]`},
		checkHeaders: func(req *http.Request, attempt int) {
			if req.Method != http.MethodPost || req.URL.Path != "/games" {
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
			}
			if req.Header.Get("Authorization") != "Bearer test-token" || req.Header.Get("Client-Id") != "cid" {
				t.Fatalf("missing or wrong auth headers")
			}
		},
	}

	client := NewClient(cfg, WithRetries(1))
	client.http.Transport = rt

	ctx := context.Background()
	out, err := client.Post(ctx, "games", []byte("fields *; limit 10; offset 0;"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(arr) != 1 || arr[0]["name"] != "Game One" {
		t.Errorf("unexpected body: %v", arr)
	}
}

func TestPost_RetryThenSuccess(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	rt := &scriptedRoundTripper{
		statuses: []int{http.StatusInternalServerError, http.StatusOK},
		bodies:   []string{"server error", `[]`},
		t:         t,
	}
	client := NewClient(cfg, WithRetries(3), WithBackoff(1*time.Millisecond, 1*time.Millisecond))
	client.http.Transport = rt
	ctx := context.Background()
	out, err := client.Post(ctx, "games", []byte("fields *;"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if len(out) != 2 || string(out) != "[]" {
		t.Errorf("unexpected body: %s", out)
	}
	if rt.attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", rt.attempts)
	}
}

func TestPost_RetryExhausted(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	rt := &scriptedRoundTripper{
		statuses: []int{http.StatusGatewayTimeout, http.StatusGatewayTimeout, http.StatusGatewayTimeout},
		bodies:   []string{"timeout", "timeout", "timeout"},
		t:         t,
	}
	client := NewClient(cfg, WithRetries(2), WithBackoff(1*time.Millisecond, 1*time.Millisecond))
	client.http.Transport = rt
	ctx := context.Background()
	_, err := client.Post(ctx, "games", []byte("fields *;"))
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if rt.attempts != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", rt.attempts)
	}
}

func TestPost_NonRetriable(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	rt := &scriptedRoundTripper{
		statuses: []int{http.StatusBadRequest},
		bodies:   []string{"bad request"},
		t:         t,
	}
	client := NewClient(cfg, WithRetries(3))
	client.http.Transport = rt
	ctx := context.Background()
	_, err := client.Post(ctx, "games", []byte("fields *;"))
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if rt.attempts != 1 {
		t.Errorf("expected no retry for 400, got %d attempts", rt.attempts)
	}
}

func TestPost_TokenRefresh_WhenAccessTokenEmpty(t *testing.T) {
	const token = "refreshed-token"
	rt := &pathScriptedRoundTripper{
		t: t,
		routes: map[string]scriptedRoute{
			"/oauth2/token": {
				statuses: []int{http.StatusOK},
				bodies:   []string{`{"access_token":"` + token + `","expires_in":3600}`},
				checkURL: func(req *http.Request) {
					if req.Method != http.MethodPost {
						t.Fatalf("unexpected token method %s", req.Method)
					}
					q := req.URL.Query()
					if q.Get("grant_type") != "client_credentials" {
						t.Fatalf("unexpected grant_type: %q", q.Get("grant_type"))
					}
				},
			},
			"/games": {
				statuses: []int{http.StatusOK},
				bodies:   []string{`[]`},
				checkHeaders: func(req *http.Request, attempt int) {
					if req.Header.Get("Authorization") != "Bearer "+token {
						t.Fatalf("missing or wrong Authorization header: %q", req.Header.Get("Authorization"))
					}
					if req.Header.Get("Client-Id") != "cid" {
						t.Fatalf("missing or wrong Client-Id header: %q", req.Header.Get("Client-Id"))
					}
				},
			},
		},
		attempts: make(map[string]int),
	}

	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "",
		MaxLimit:     500,
	}

	client := NewClient(cfg, WithRetries(1))
	client.http.Transport = rt

	out, err := client.Post(context.Background(), "games", []byte("fields *;"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if string(out) != "[]" {
		t.Fatalf("unexpected body: %s", out)
	}
}

func TestPost_RetriesOn429(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}

	rt := &pathScriptedRoundTripper{
		t: t,
		routes: map[string]scriptedRoute{
			"/games": {
				statuses: []int{http.StatusTooManyRequests, http.StatusOK},
				bodies:   []string{"rate limited", `[]`},
			},
		},
		attempts: make(map[string]int),
	}

	client := NewClient(cfg, WithRetries(5), WithBackoff(1*time.Millisecond, 1*time.Millisecond))
	client.http.Transport = rt

	_, err := client.Post(context.Background(), "games", []byte("fields *;"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if rt.attempts["/games"] != 2 {
		t.Fatalf("expected 2 attempts on 429 (initial + 1 retry), got %d", rt.attempts["/games"])
	}
}

func TestPost_MaxRetryDurationStopsRetries(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}

	statuses := make([]int, 100)
	bodies := make([]string, 100)
	for i := 0; i < 100; i++ {
		statuses[i] = http.StatusInternalServerError
		bodies[i] = "server error"
	}

	rt := &pathScriptedRoundTripper{
		t: t,
		routes: map[string]scriptedRoute{
			"/games": {
				statuses: statuses,
				bodies:   bodies,
			},
		},
		attempts: make(map[string]int),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(
		cfg,
		WithRetries(50),
		WithBackoff(1*time.Millisecond, 1*time.Millisecond),
		WithMaxRetryDuration(10*time.Millisecond),
	)
	client.http.Transport = rt

	_, err := client.Post(ctx, "games", []byte("fields *;"))
	if err == nil {
		t.Fatal("expected error")
	}

	attempts := rt.attempts["/games"]
	if attempts < 2 {
		t.Fatalf("expected at least 2 attempts before max retry duration, got %d", attempts)
	}
	if attempts > 25 {
		t.Fatalf("expected attempts to stop fairly quickly due to max retry duration, got %d", attempts)
	}
}

func TestPost_MetricsRecordsRetries(t *testing.T) {
	metrics := NewMetrics("games")
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}

	rt := &pathScriptedRoundTripper{
		t: t,
		routes: map[string]scriptedRoute{
			"/games": {
				statuses: []int{http.StatusInternalServerError, http.StatusOK},
				bodies:   []string{"server error", `[]`},
			},
		},
		attempts: make(map[string]int),
	}

	client := NewClient(cfg,
		WithRetries(5),
		WithBackoff(1*time.Millisecond, 1*time.Millisecond),
		WithMetrics(metrics),
	)
	client.http.Transport = rt

	_, err := client.Post(context.Background(), "games", []byte("fields *;"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}

	entity, posts := metrics.Snapshot()
	if entity != "games" {
		t.Fatalf("expected metrics entity games, got %q", entity)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 metrics record for single Post invocation, got %d", len(posts))
	}
	if posts[0].Endpoint != "games" {
		t.Fatalf("expected metrics endpoint games, got %q", posts[0].Endpoint)
	}
	// First attempt fails (retriable), second attempt succeeds => 1 retry => Retries field should be 1.
	if posts[0].Retries != 1 {
		t.Fatalf("expected metrics Retries=1, got %d", posts[0].Retries)
	}
}

func TestPost_ContextCanceled(t *testing.T) {
	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	rt := &scriptedRoundTripper{
		statuses: []int{http.StatusInternalServerError},
		bodies:   []string{"server error"},
		t:         t,
	}
	client := NewClient(cfg, WithRetries(5), WithBackoff(10*time.Millisecond, 10*time.Millisecond))
	client.http.Transport = rt
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Post(ctx, "games", nil)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled (or wrapped), got %v", err)
	}
}

