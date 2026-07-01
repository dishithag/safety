// Package api is the HTTP delivery layer for analyticsapi. It adapts inbound
// HTTP requests to the usecase layer and serves the responses.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/domain"
)

const rootPath = "/zero-trust-analytics"

// NarrativeService is the usecase capability the API depends on.
type NarrativeService interface {
	Get(ctx context.Context, id string) (string, error)
}

// Manager wires HTTP routes to the usecase layer.
type Manager struct {
	logger     *slog.Logger
	narratives NarrativeService
}

// NewManager returns a new API Manager.
func NewManager(logger *slog.Logger, narratives NarrativeService) *Manager {
	return &Manager{logger: logger, narratives: narratives}
}

// Routes returns the HTTP handler exposing all of this service's endpoints.
func (m *Manager) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", m.handleHealth)
	mux.HandleFunc("GET "+rootPath+"/narratives/{id}", m.handleNarrative)
	return mux
}

func (m *Manager) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (m *Manager) handleNarrative(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := m.narratives.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNarrativeNotFound) {
			http.Error(w, "narrative not found", http.StatusNotFound)
			return
		}
		m.logger.Error("failed to get narrative", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(body))
}
