package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
)

var _ application.EventLog = (*EventLogRepository)(nil)

type eventExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type EventLogRepository struct {
	db eventExecutor
}

const sqliteTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func NewEventLogRepository(db eventExecutor) *EventLogRepository {
	return &EventLogRepository{db: db}
}

func (r *EventLogRepository) Append(ctx context.Context, event domain.BusinessEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	const query = `
		INSERT INTO event_log(
			id,
			aggregate_id,
			event_type,
			occurred_at,
			aggregate_version,
			correlation_id,
			causation_id,
			payload,
			runtime_sequence
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`

	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	runtimeSequence, err := persistencesqlite.NextRuntimeSequence(ctx, exec)
	if err != nil {
		return fmt.Errorf("allocate runtime sequence for event %q: %w", event.ID(), err)
	}
	result, err := exec.ExecContext(
		ctx,
		query,
		event.ID(),
		event.AggregateID(),
		event.Type(),
		event.OccurredAt().UTC().Format(sqliteTimestampLayout),
		event.AggregateVersion(),
		event.CorrelationID(),
		event.CausationID(),
		string(event.Payload()),
		runtimeSequence,
	)
	if err != nil {
		return fmt.Errorf("append event %q: %w", event.ID(), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows for event %q: %w", event.ID(), err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("append event %q: %w", event.ID(), application.ErrEventAlreadyExists)
	}

	return nil
}

func (r *EventLogRepository) ListByAggregate(
	ctx context.Context,
	aggregateID domain.ID,
) ([]domain.BusinessEvent, error) {
	const query = `
		SELECT
			id,
			aggregate_id,
			event_type,
			occurred_at,
			aggregate_version,
			correlation_id,
			causation_id,
			payload,
			runtime_sequence
		FROM event_log
		WHERE aggregate_id = ?
			ORDER BY occurred_at ASC, aggregate_version ASC, id ASC`

	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	rows, err := exec.QueryContext(ctx, query, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("list events for aggregate %q: %w", aggregateID, err)
	}
	defer func() { _ = rows.Close() }()

	events := make([]domain.BusinessEvent, 0)
	for rows.Next() {
		var (
			id               string
			storedAggregate  string
			eventType        string
			occurredAt       string
			aggregateVersion uint64
			correlationID    string
			causationID      string
			payload          []byte
			runtimeSequence  uint64
		)

		if err := rows.Scan(
			&id,
			&storedAggregate,
			&eventType,
			&occurredAt,
			&aggregateVersion,
			&correlationID,
			&causationID,
			&payload,
			&runtimeSequence,
		); err != nil {
			return nil, fmt.Errorf("scan event for aggregate %q: %w", aggregateID, err)
		}

		parsedAt, err := time.Parse(time.RFC3339Nano, occurredAt)
		if err != nil {
			return nil, fmt.Errorf("parse event %q timestamp: %w", id, err)
		}

		event, err := domain.NewBusinessEventWithPayload(
			domain.EventID(id),
			domain.ID(storedAggregate),
			domain.EventType(eventType),
			parsedAt,
			aggregateVersion,
			correlationID,
			domain.EventID(causationID),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("restore event %q: %w", id, err)
		}
		event = event.WithRuntimeSequence(runtimeSequence)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events for aggregate %q: %w", aggregateID, err)
	}

	return events, nil
}
