package main

import (
	"fmt"

	"github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/internal/profiles/main"
	"github.com/vernal96/go-cms/kernel"
)

func main() {

	connectorManager := kernel.NewConnectorManager()

	connectorManager.DB.Register(postgres.Connector)

	app := kernel.NewApp(
		[]kernel.Profile{
			main_profile.Profile{},
		},
		connectorManager,
	)

	fmt.Println(app)

}
