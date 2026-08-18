package app

import (
	"github.com/vernal96/go-cms/kernel/console"
)

func (a *App) CommandProviders() []console.Provider {
	if a == nil {
		return nil
	}

	return append([]console.Provider(nil), a.providers...)
}

func (a *App) collectModuleCommandProviders() {
	for _, profile := range a.definition.Profiles {
		for _, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				continue
			}

			a.addProvider(
				"module:"+string(profileModule.Module.Code()),
				profileModule.Module,
			)
		}
	}
}

func (a *App) addProvider(key string, candidate any) {
	provider, ok := candidate.(console.Provider)
	if !ok {
		return
	}

	for _, registered := range a.providers {
		if providerIdentity(registered) == key {
			return
		}
	}

	a.providers = append(a.providers, keyedProvider{
		key:      key,
		Provider: provider,
	})
}

type keyedProvider struct {
	key string
	console.Provider
}

func providerIdentity(provider console.Provider) string {
	if keyed, ok := provider.(keyedProvider); ok {
		return keyed.key
	}
	return ""
}
