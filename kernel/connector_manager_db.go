package kernel

type DBConnectorManager struct {
	connectors map[ConnectorCode]DBConnector
}

func NewDBConnectorManager() *DBConnectorManager {
	return &DBConnectorManager{
		connectors: make(map[ConnectorCode]DBConnector),
	}
}

func (m *DBConnectorManager) Register(code DriverCode, conn DBConnector, config any) {}
