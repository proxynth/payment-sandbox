package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
)

var _ application.EventLog = (*EventLogRepository)(nil)

type eventExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type EventLogRepository struct {
	db eventExecutor
}

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
			causation_id
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`

	result, err := r.db.ExecContext(
		ctx,
		query,
		event.ID(),
		event.AggregateID(),
		event.Type(),
		event.OccurredAt().UTC().Format(time.RFC3339Nano),
		event.AggregateVersion(),
		event.CorrelationID(),
		event.CausationID(),
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
			causation_id
		FROM event_log
		WHERE aggregate_id = ?
		ORDER BY occurred_at ASC, aggregate_version ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, query, aggregateID)
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
		)

		if err := rows.Scan(
			&id,
			&storedAggregate,
			&eventType,
			&occurredAt,
			&aggregateVersion,
			&correlationID,
			&causationID,
		); err != nil {
			return nil, fmt.Errorf("scan event for aggregate %q: %w", aggregateID, err)
		}

		parsedAt, err := time.Parse(time.RFC3339Nano, occurredAt)
		if err != nil {
			return nil, fmt.Errorf("parse event %q timestamp: %w", id, err)
		}

		event, err := domain.NewBusinessEvent(
			domain.EventID(id),
			domain.ID(storedAggregate),
			domain.EventType(eventType),
			parsedAt,
			aggregateVersion,
			correlationID,
			domain.EventID(causationID),
		)
		if err != nil {
			return nil, fmt.Errorf("restore event %q: %w", id, err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events for aggregate %q: %w", aggregateID, err)
	}

	return events, nil
}
