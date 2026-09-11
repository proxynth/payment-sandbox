package sqlite

import (
	"context"
	"database/sql"
)

// RuntimeSequenceExecutor is the minimal SQLite executor needed to allocate a
// durable global runtime sequence value.
type RuntimeSequenceExecutor interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// NextRuntimeSequence allocates one monotonically increasing sequence value.
// The allocation is atomic in SQLite; callers include the returned value in
// their insert so assigning it never requires updating the inserted row.
func NextRuntimeSequence(ctx context.Context, exec RuntimeSequenceExecutor) (uint64, error) {
	var sequence uint64
	err := exec.QueryRowContext(ctx, `
		INSERT INTO runtime_sequence(id, value) VALUES (1, 1)
		ON CONFLICT(id) DO UPDATE SET value = runtime_sequence.value + 1
		RETURNING value`).Scan(&sequence)
	if err != nil {
		return 0, err
	}
	return sequence, nil
}
