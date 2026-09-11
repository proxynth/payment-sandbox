package application

import "context"

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(ctx context.Context) error,
	) error
}

type TransactionalRepository interface {
	WithinContext(context.Context, func(context.Context) error) error
}

func withinRepositoryTransaction(ctx context.Context, repository Repository, fn func(context.Context) error) error {
	if tx, ok := repository.(TransactionalRepository); ok {
		return tx.WithinContext(ctx, fn)
	}
	return fn(ctx)
}
