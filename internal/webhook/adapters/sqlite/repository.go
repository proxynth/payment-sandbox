package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/webhook/application"
	"proxynth/payment-sandbox/internal/webhook/domain"
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ db executor }

func NewRepository(db executor) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, endpoint *domain.Endpoint) error {
	if endpoint == nil {
		return fmt.Errorf("nil webhook endpoint")
	}
	exec := r.executor(ctx)
	result, err := exec.ExecContext(ctx, `INSERT INTO webhook_endpoints(id,url) VALUES (?,?) ON CONFLICT(id) DO NOTHING`, endpoint.ID(), endpoint.URL())
	if err != nil {
		return fmt.Errorf("save webhook endpoint %q: %w", endpoint.ID(), err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return application.ErrEndpointAlreadyExists
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id domain.EndpointID) (*domain.Endpoint, error) {
	var url string
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT url FROM webhook_endpoints WHERE id = ?`, id).Scan(&url)
	if err == sql.ErrNoRows {
		return nil, application.ErrEndpointNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find webhook endpoint %q: %w", id, err)
	}
	return domain.NewEndpoint(id, url)
}

func (r *Repository) List(ctx context.Context) ([]*domain.Endpoint, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT id,url FROM webhook_endpoints ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list webhook endpoints: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]*domain.Endpoint, 0)
	for rows.Next() {
		var id, url string
		if err := rows.Scan(&id, &url); err != nil {
			return nil, err
		}
		endpoint, err := domain.NewEndpoint(domain.EndpointID(id), url)
		if err != nil {
			return nil, err
		}
		items = append(items, endpoint)
	}
	return items, rows.Err()
}

func (r *Repository) executor(ctx context.Context) executor {
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}
