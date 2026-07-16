package postgres

import (
	"context"

	"github.com/vernal96/go-cms/kernel"
)

type Connector struct {
	config Config
}

func New(config Config) *Connector {
	return &Connector{}
}

func (c *Connector) Code() kernel.DriverCode {
	return kernel.PostgresDriverCode
}

func (c *Connector) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

func (c *Connector) Ping(ctx context.Context) error {
	_ = ctx
	return nil
}

var _ kernel.DBConnector = (*Connector)(nil)
