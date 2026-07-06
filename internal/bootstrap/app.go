package bootstrap

import "github.com/vernal96/go-cms/kernel"

func NewApp() *kernel.App {
	return kernel.NewApp(kernel.AppConfig{})
}
