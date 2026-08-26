package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type Metadata struct {
	CorrelationID string
	CausationID   string
}

type contextKey struct{}

func WithMetadata(ctx context.Context, metadata Metadata) context.Context {
	return context.WithValue(ctx, contextKey{}, metadata)
}

func MetadataFromContext(ctx context.Context) Metadata {
	metadata, _ := ctx.Value(contextKey{}).(Metadata)
	return metadata
}

func NewCorrelationID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate correlation id: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
