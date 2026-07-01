package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestManager() *Manager {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewManager(logger, stubNarratives{})
}

type stubNarratives struct{}

func (stubNarratives) Get(_ context.Context, _ string) (string, error) { return "Hello World", nil }

func TestHandleNarrative(t *testing.T) {
	srv := httptest.NewServer(newTestManager().Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/zero-trust-analytics/narratives/abc123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if got, want := string(body), "Hello World"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := httptest.NewServer(newTestManager().Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
