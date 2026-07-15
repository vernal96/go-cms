package site_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type domainResolverStub struct {
	resolveByDomain func(ctx context.Context, domain string) (site.Site, bool, error)
}

func (r *domainResolverStub) ResolveByDomain(ctx context.Context, domain string) (site.Site, bool, error) {
	return r.resolveByDomain(ctx, domain)
}

type providerTestProfile struct {
	code    kernel.ProfileCode
	modules []kernel.ProfileModule
}

func (p *providerTestProfile) Code() kernel.ProfileCode {
	return p.code
}

func (p *providerTestProfile) Modules() []kernel.ProfileModule {
	return p.modules
}

type providerTestModule struct {
	code       kernel.ModuleCode
	bootCount  int
	bootConfig any
}

func (m *providerTestModule) Code() kernel.ModuleCode {
	return m.code
}

func (m *providerTestModule) Register(registry kernel.Registry) error {
	_ = registry
	return nil
}

func (m *providerTestModule) Boot(ctx context.Context, moduleContext kernel.ModuleContext) error {
	_ = ctx

	m.bootCount++
	m.bootConfig = moduleContext.ModuleConfig()

	return nil
}

func TestNewRuntimeProvider_ValidatesDependencies(t *testing.T) {
	resolver := &domainResolverStub{
		resolveByDomain: func(ctx context.Context, domain string) (site.Site, bool, error) {
			return site.Site{}, false, nil
		},
	}
	profiles := kernel.NewProfileRegistryManager()
	factory := kernel.NewSiteRuntimeFactory(kernel.NewApp(kernel.AppConfig{}))

	tests := []struct {
		name            string
		resolver        site.DomainResolver
		profiles        kernel.ProfileRegistry
		runtimeFactory  *kernel.SiteRuntimeFactory
	}{
		{
			name:           "nil domain resolver",
			profiles:       profiles,
			runtimeFactory: factory,
		},
		{
			name:           "nil profile registry",
			resolver:       resolver,
			runtimeFactory: factory,
		},
		{
			name:     "nil runtime factory",
			resolver: resolver,
			profiles: profiles,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := site.NewRuntimeProvider(
				tt.resolver,
				tt.profiles,
				tt.runtimeFactory,
			)

			if provider != nil {
				t.Fatal("expected nil runtime provider")
			}

			if err == nil {
				t.Fatal("expected dependency validation error")
			}
		})
	}
}

func TestRuntimeProvider_RuntimeByDomainReturnsNotFound(t *testing.T) {
	resolver := &domainResolverStub{
		resolveByDomain: func(ctx context.Context, domain string) (site.Site, bool, error) {
			return site.Site{}, false, nil
		},
	}

	provider, err := site.NewRuntimeProvider(
		resolver,
		kernel.NewProfileRegistryManager(),
		kernel.NewSiteRuntimeFactory(kernel.NewApp(kernel.AppConfig{})),
	)
	if err != nil {
		t.Fatalf("create runtime provider: %v", err)
	}

	runtime, exists, err := provider.RuntimeByDomain(context.Background(), "missing.example.com")
	if err != nil {
		t.Fatalf("resolve runtime: %v", err)
	}

	if exists {
		t.Fatal("unexpected runtime found")
	}

	if runtime != nil {
		t.Fatal("expected nil runtime")
	}
}

func TestRuntimeProvider_RuntimeByDomainReturnsResolverError(t *testing.T) {
	resolverErr := errors.New("resolver failed")
	resolver := &domainResolverStub{
		resolveByDomain: func(ctx context.Context, domain string) (site.Site, bool, error) {
			return site.Site{}, false, resolverErr
		},
	}

	provider, err := site.NewRuntimeProvider(
		resolver,
		kernel.NewProfileRegistryManager(),
		kernel.NewSiteRuntimeFactory(kernel.NewApp(kernel.AppConfig{})),
	)
	if err != nil {
		t.Fatalf("create runtime provider: %v", err)
	}

	_, _, err = provider.RuntimeByDomain(context.Background(), "example.com")
	if !errors.Is(err, resolverErr) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

func TestRuntimeProvider_RuntimeByDomainReturnsErrorForUnknownProfile(t *testing.T) {
	resolver := &domainResolverStub{
		resolveByDomain: func(ctx context.Context, domain string) (site.Site, bool, error) {
			return site.Site{
				ID:          1,
				ProfileCode: "missing",
				Domain:      domain,
			}, true, nil
		},
	}

	provider, err := site.NewRuntimeProvider(
		resolver,
		kernel.NewProfileRegistryManager(),
		kernel.NewSiteRuntimeFactory(kernel.NewApp(kernel.AppConfig{})),
	)
	if err != nil {
		t.Fatalf("create runtime provider: %v", err)
	}

	runtime, exists, err := provider.RuntimeByDomain(context.Background(), "example.com")

	if runtime != nil {
		t.Fatal("expected nil runtime")
	}

	if exists {
		t.Fatal("unexpected runtime found")
	}

	if err == nil || !strings.Contains(err.Error(), `profile "missing" not found`) {
		t.Fatalf("expected profile not found error, got %v", err)
	}
}

func TestRuntimeProvider_RuntimeByDomainBuildsRuntimeForSiteProfile(t *testing.T) {
	type contextKey string

	ctx := context.WithValue(context.Background(), contextKey("request"), "test")
	wantDomain := "example.com"
	wantConfig := struct {
		Enabled bool
	}{
		Enabled: true,
	}

	module := &providerTestModule{
		code: "test",
	}
	profile := &providerTestProfile{
		code: "public",
		modules: []kernel.ProfileModule{
			{
				Module:       module,
				ModuleConfig: wantConfig,
			},
		},
	}

	profiles := kernel.NewProfileRegistryManager()
	if err := profiles.Register(profile); err != nil {
		t.Fatalf("register profile: %v", err)
	}

	resolver := &domainResolverStub{
		resolveByDomain: func(gotContext context.Context, gotDomain string) (site.Site, bool, error) {
			if gotContext != ctx {
				t.Fatal("resolver received a different context")
			}

			if gotDomain != wantDomain {
				t.Fatalf("unexpected domain: got %q, want %q", gotDomain, wantDomain)
			}

			return site.Site{
				ID:          1,
				ProfileCode: profile.Code(),
				Domain:      gotDomain,
				Locale:      "ru-RU",
			}, true, nil
		},
	}

	app := kernel.NewApp(kernel.AppConfig{})
	provider, err := site.NewRuntimeProvider(
		resolver,
		profiles,
		kernel.NewSiteRuntimeFactory(app),
	)
	if err != nil {
		t.Fatalf("create runtime provider: %v", err)
	}

	runtime, exists, err := provider.RuntimeByDomain(ctx, wantDomain)
	if err != nil {
		t.Fatalf("resolve runtime: %v", err)
	}

	if !exists {
		t.Fatal("expected runtime to exist")
	}

	if runtime == nil {
		t.Fatal("runtime is nil")
	}

	if runtime.App() != app {
		t.Fatal("runtime contains a different app")
	}

	if runtime.Profile() != profile {
		t.Fatal("runtime contains a different profile")
	}

	if module.bootCount != 1 {
		t.Fatalf("unexpected module boot count: got %d, want 1", module.bootCount)
	}

	if module.bootConfig != wantConfig {
		t.Fatalf("unexpected module config: got %#v, want %#v", module.bootConfig, wantConfig)
	}
}

