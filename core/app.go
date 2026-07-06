package core

// App is the root CMS application container.
//
// At this stage it is intentionally empty. Later it will hold application-level
// dependencies such as logger, cache manager, storage manager, event bus,
// repositories, and configuration.
type App struct{}

// NewApp creates a new CMS application container.
func NewApp() *App {
	return &App{}
}
