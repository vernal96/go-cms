package kernel

import "context"

type DriverCode string

type ConnectionCode string

type DBConnector interface {
	Ping(ctx context.Context) error
	Close() error
}
