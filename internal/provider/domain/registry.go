package domain

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
)

// Registry stores provider plugins by their stable identity. It owns
// selection, but not provider lifecycle or configuration.
type Registry struct {
	mu        sync.RWMutex
	providers map[ProviderID]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[ProviderID]Provider)}
}

// Register adds a provider without replacing an existing provider with the
// same identity.
func (r *Registry) Register(provider Provider) error {
	if isNilProvider(provider) {
		return ErrNilProvider
	}

	identity := provider.Identity()
	if err := identity.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.providers == nil {
		r.providers = make(map[ProviderID]Provider)
	}

	if _, exists := r.providers[identity.ID]; exists {
		return fmt.Errorf("%w: %s", ErrProviderAlreadyExists, identity.ID)
	}

	r.providers[identity.ID] = provider

	return nil
}

// Resolve returns the provider registered for the requested identity.
func (r *Registry) Resolve(id ProviderID) (Provider, error) {
	if !id.Valid() {
		return nil, ErrInvalidProviderID
	}

	r.mu.RLock()
	provider, exists := r.providers[id]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, id)
	}

	return provider, nil
}

// IDs returns registered provider identities in a stable order.
func (r *Registry) IDs() []ProviderID {
	r.mu.RLock()
	ids := make([]ProviderID, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	r.mu.RUnlock()

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	return ids
}

func isNilProvider(provider Provider) bool {
	if provider == nil {
		return true
	}

	value := reflect.ValueOf(provider)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
