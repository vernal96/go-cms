package kernel

import "context"

type DriverCode string

type ConnectorCode string

const (
	MemoryDriverCode   DriverCode = "memory"
	PostgresDriverCode DriverCode = "postgres"
)

type Connector interface {
	Code() DriverCode
	Close(ctx context.Context) error
}

type DBConnector interface {
	Connector
	Ping(ctx context.Context) error
}
