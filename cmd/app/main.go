package main

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
)

func main() {
	ctx := context.Background()

	app := kernel.NewApp(kernel.AppConfig{})

	profile := dev.Profile{}

	runtimeFactory := kernel.NewSiteRuntimeFactory(app)

	runtime, err := runtimeFactory.Make(ctx, profile)
	if err != nil {
		panic(err)
	}

	fmt.Println("runtime created for profile:", runtime.Profile().Code())
}
