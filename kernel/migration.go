package kernel

import (
	"context"
	"errors"
	"fmt"
)

type MigrationProviderCode string

type MigrationDirection string

const (
	MigrationDirectionUp   MigrationDirection = "up"
	MigrationDirectionDown MigrationDirection = "down"
)

type MigrationProvider interface {
	Code() MigrationProviderCode
	ModuleCode() ModuleCode
	AdapterCode() AdapterCode

	Up(ctx context.Context) error
	Down(ctx context.Context) error
}

type MigrationRegistry interface {
	AddProvider(provider MigrationProvider) error
	Provider(code MigrationProviderCode) (MigrationProvider, bool)
	Providers() []MigrationProvider
}

type DefaultMigrationRegistry struct {
	providers     map[MigrationProviderCode]MigrationProvider
	providerCodes []MigrationProviderCode
}

func NewDefaultMigrationRegistry() *DefaultMigrationRegistry {
	return &DefaultMigrationRegistry{
		providers:     make(map[MigrationProviderCode]MigrationProvider),
		providerCodes: make([]MigrationProviderCode, 0),
	}
}

func (r *DefaultMigrationRegistry) AddProvider(provider MigrationProvider) error {
	if provider == nil {
		return errors.New("migration provider is nil")
	}

	code := provider.Code()
	if code == "" {
		return errors.New("migration provider code is empty")
	}

	if _, exists := r.providers[code]; exists {
		return fmt.Errorf("migration provider %q already registered", code)
	}

	r.providers[code] = provider
	r.providerCodes = append(r.providerCodes, code)

	return nil
}

func (r *DefaultMigrationRegistry) Provider(code MigrationProviderCode) (MigrationProvider, bool) {
	provider, exists := r.providers[code]
	return provider, exists
}

func (r *DefaultMigrationRegistry) Providers() []MigrationProvider {
	providers := make([]MigrationProvider, 0, len(r.providerCodes))

	for _, code := range r.providerCodes {
		providers = append(providers, r.providers[code])
	}

	return providers
}

var _ MigrationRegistry = (*DefaultMigrationRegistry)(nil)
