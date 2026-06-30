package domain

import (
	"context"
	"errors"
	"fmt"
)

// ErrNarrativeNotFound indicates no stored narrative exists for a CID.
var ErrNarrativeNotFound = errors.New("narrative not found")

// NarrativeStore retrieves stored narrative content by CID.
type NarrativeStore interface {
	Get(ctx context.Context, cid string) (string, error)
}

// NarrativeObjectKey returns the object-store key for a CID summary.
func NarrativeObjectKey(cid string) string {
	return fmt.Sprintf("summary/cids/%s.md", cid)
}
