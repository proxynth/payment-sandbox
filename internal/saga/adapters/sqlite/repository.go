package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"proxynth/payment-sandbox/internal/saga/domain"
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ db executor }

func NewRepository(db executor) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, instance domain.Instance) error {
	completed, err := json.Marshal(instance.CompletedSteps)
	if err != nil {
		return fmt.Errorf("marshal completed saga steps: %w", err)
	}
	compensation, err := json.Marshal(instance.Compensation)
	if err != nil {
		return fmt.Errorf("marshal compensation saga steps: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO saga_instances(id, payment_id, payload, status, current_step, completed_steps, compensation_steps, seed, version, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT(id) DO UPDATE SET
			payload=excluded.payload,
			status=excluded.status, current_step=excluded.current_step,
			completed_steps=excluded.completed_steps, compensation_steps=excluded.compensation_steps,
			seed=excluded.seed, version=excluded.version, updated_at=excluded.updated_at
		WHERE saga_instances.version = excluded.version - 1`,
		instance.ID, instance.PaymentID, instance.Payload, instance.Status, instance.CurrentStep,
		string(completed), string(compensation), instance.Seed, instance.Version, instance.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save saga %q: %w", instance.ID, err)
	}
	return nil
}

func (r *Repository) Find(ctx context.Context, id domain.ID) (domain.Instance, error) {
	var instance domain.Instance
	var completed, compensation, updated string
	err := r.db.QueryRowContext(ctx, `SELECT id,payment_id,payload,status,current_step,completed_steps,compensation_steps,seed,version,updated_at FROM saga_instances WHERE id=$1`, id).
		Scan(&instance.ID, &instance.PaymentID, &instance.Payload, &instance.Status, &instance.CurrentStep, &completed, &compensation, &instance.Seed, &instance.Version, &updated)
	if err != nil {
		return domain.Instance{}, fmt.Errorf("find saga %q: %w", id, err)
	}
	if err := json.Unmarshal([]byte(completed), &instance.CompletedSteps); err != nil {
		return domain.Instance{}, err
	}
	if err := json.Unmarshal([]byte(compensation), &instance.Compensation); err != nil {
		return domain.Instance{}, err
	}
	instance.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return domain.Instance{}, err
	}
	return instance, nil
}
