package kernel

type ConnectorManager struct {
	DB *DBConnectorManager
}

func NewConnectorManager() *ConnectorManager {
	return &ConnectorManager{
		DB: NewDBConnectorManager(),
	}
}
