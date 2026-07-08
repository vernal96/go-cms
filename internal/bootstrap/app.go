package bootstrap

import "github.com/vernal96/go-cms/kernel"

func NewApp() (*kernel.App, error) {
	app := kernel.NewApp(kernel.AppConfig{
		AdapterDefaults: kernel.AdapterDefaults{
			RepositoryAdapter: "postgres",
		},
	})

	return app, nil
}
