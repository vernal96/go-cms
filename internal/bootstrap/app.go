package bootstrap

import "github.com/vernal96/go-cms/kernel"

func NewApp() (*kernel.App, error) {
	app := kernel.NewApp(kernel.AppConfig{})

	return app, nil
}
