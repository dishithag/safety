// Package narrative contains the application logic for retrieving LLM-generated
// narratives. It depends only on domain interfaces, never on concrete gateways.
package narrative

import (
	"context"
	"log/slog"

	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/domain"
)

// Service orchestrates retrieval of narratives.
type Service struct {
	logger *slog.Logger
	store  domain.NarrativeStore
}

// NewService returns a new narrative Service.
func NewService(logger *slog.Logger, store domain.NarrativeStore) *Service {
	return &Service{logger: logger, store: store}
}

// Get returns the narrative for the given assessment id.
func (s *Service) Get(ctx context.Context, id string) (string, error) {
	return s.store.Get(ctx, id)
}
