package main

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/internal/bootstrap"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
)

func main() {
	ctx := context.Background()

	app, err := bootstrap.NewApp()
	if err != nil {
		panic(err)
	}

	profiles, err := bootstrap.NewProfileRegistry()
	if err != nil {
		panic(err)
	}

	profile, exists := profiles.Get(dev.ProfileCode)
	if !exists {
		panic("profile not found")
	}

	runtimeFactory := kernel.NewSiteRuntimeFactory(app)

	runtime, err := runtimeFactory.Make(ctx, profile)
	if err != nil {
		panic(err)
	}

	fmt.Println("runtime created for profile:", runtime.Profile().Code())
}
