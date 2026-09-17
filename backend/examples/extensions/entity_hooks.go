package extensions

import (
	"context"
	"errors"
	"strings"

	"github.com/vernal96/go-cms-kernel"
	"github.com/vernal96/go-cms-kernel/entityhooks"
)

// Product and its extension point belong to this example module. No kernel
// package, ModuleContext field or core entity registry knows this type.
type Product struct{ Name string }

var BeforeProductCreate = entityhooks.NewKey[Product]("catalog_example", "catalog_example.before_create", entityhooks.Site)

type CatalogModule struct{}

func (CatalogModule) Code() kernel.ModuleCode { return "catalog_example" }
func (CatalogModule) Build(_ context.Context, ctx kernel.ModuleContext) (kernel.ModuleRuntime, error) {
	return &CatalogRuntime{hooks: ctx.EntityHooks().Dispatcher()}, nil
}

type CatalogRuntime struct{ hooks entityhooks.Dispatcher }

func (*CatalogRuntime) ModuleCode() kernel.ModuleCode { return "catalog_example" }
func (r *CatalogRuntime) PrepareProduct(ctx context.Context, product Product) (Product, error) {
	release, err := r.hooks.Acquire()
	if err != nil {
		return Product{}, err
	}
	defer release()
	if err := entityhooks.Before(ctx, r.hooks, BeforeProductCreate, &product); err != nil {
		return Product{}, err
	}
	product.Name = strings.TrimSpace(product.Name)
	if product.Name == "" {
		return Product{}, errors.New("product name is required")
	}
	return product, nil
}

type CatalogPolicyModule struct{}

func (CatalogPolicyModule) Code() kernel.ModuleCode { return "catalog_policy_example" }
func (CatalogPolicyModule) Dependencies() []kernel.ModuleCode {
	return []kernel.ModuleCode{"catalog_example"}
}
func (CatalogPolicyModule) Build(_ context.Context, ctx kernel.ModuleContext) (kernel.ModuleRuntime, error) {
	if err := entityhooks.RegisterBefore(ctx.EntityHooks(), BeforeProductCreate, "default_name", func(_ context.Context, product *Product) error {
		if product.Name == "" {
			product.Name = "New product"
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return catalogPolicyRuntime{}, nil
}

type catalogPolicyRuntime struct{}

func (catalogPolicyRuntime) ModuleCode() kernel.ModuleCode { return "catalog_policy_example" }
