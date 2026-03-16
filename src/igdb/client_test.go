package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vargames-name-gen/src/config"
)

func TestPost_Success(t *testing.T) {
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/games" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Client-Id") != "cid" {
			t.Error("missing or wrong auth headers")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"name":"Game One"}]`))
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      igdb.URL,
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	client := NewClient(cfg, WithRetries(1))
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
	attempts := 0
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      igdb.URL,
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	client := NewClient(cfg, WithRetries(3), WithBackoff(1, 5)) // minimal backoff for test
	ctx := context.Background()
	out, err := client.Post(ctx, "games", []byte("fields *;"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if len(out) != 2 || string(out) != "[]" {
		t.Errorf("unexpected body: %s", out)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestPost_RetryExhausted(t *testing.T) {
	attempts := 0
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte("timeout"))
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      igdb.URL,
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	client := NewClient(cfg, WithRetries(2), WithBackoff(1, 5))
	ctx := context.Background()
	_, err := client.Post(ctx, "games", []byte("fields *;"))
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", attempts)
	}
}

func TestPost_NonRetriable(t *testing.T) {
	attempts := 0
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      igdb.URL,
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	client := NewClient(cfg, WithRetries(3))
	ctx := context.Background()
	_, err := client.Post(ctx, "games", []byte("fields *;"))
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if attempts != 1 {
		t.Errorf("expected no retry for 400, got %d attempts", attempts)
	}
}

func TestPost_ContextCanceled(t *testing.T) {
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      igdb.URL,
		AccessToken:  "test-token",
		MaxLimit:     500,
	}
	client := NewClient(cfg, WithRetries(5), WithBackoff(10, 10))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Post(ctx, "games", nil)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled (or wrapped), got %v", err)
	}
}
