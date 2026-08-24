package application

import (
	"context"
	"sort"

	"proxynth/payment-sandbox/internal/webhook/domain"
)

type ListEndpoints struct{ repository Repository }

func NewListEndpoints(repository Repository) *ListEndpoints {
	return &ListEndpoints{repository: repository}
}

func (l *ListEndpoints) Execute(ctx context.Context) ([]*domain.Endpoint, error) {
	endpoints, err := l.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].ID() < endpoints[j].ID()
	})

	return endpoints, nil
}
