package domain

import "context"

// ExecutionMetadata identifies one durable job attempt while it is handled.
// It is context-scoped so handlers can audit their side effects without being
// coupled to the scheduler worker implementation.
type ExecutionMetadata struct {
	JobID   JobID
	Attempt uint64
}

type executionMetadataKey struct{}

// WithExecutionMetadata adds the currently executing job attempt to ctx.
func WithExecutionMetadata(ctx context.Context, metadata ExecutionMetadata) context.Context {
	return context.WithValue(ctx, executionMetadataKey{}, metadata)
}

// ExecutionMetadataFromContext returns the durable job attempt, if ctx was
// created by a scheduler worker.
func ExecutionMetadataFromContext(ctx context.Context) (ExecutionMetadata, bool) {
	metadata, ok := ctx.Value(executionMetadataKey{}).(ExecutionMetadata)
	return metadata, ok
}
