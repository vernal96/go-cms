package background

import "context"

type Task struct {
	Name string
	Run  func(context.Context) error
}

// Provider is an optional module-runtime capability for application-scoped
// lifecycle work. App starts one provider per module and cancels it before
// closing databases, filesystems, or the event bus.
type Provider interface {
	BackgroundTasks() []Task
}

type NamesProvider interface {
	BackgroundTaskNames() []string
}
