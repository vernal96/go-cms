package kernel

import "context"

type ConnectionCode string

type DBConnector interface {
	Code() ConnectionCode
	Ping(ctx context.Context) error
	Close() error
}

type ModuleDatabase interface {
	ModuleCode() ModuleCode
}

type DatabaseResolver interface {
	MainModuleDatabase(ModuleCode) (ModuleDatabase, bool)
	ModuleDatabase(ConnectionCode, ModuleCode) (ModuleDatabase, bool)
}
