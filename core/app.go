package core

import "errors"

type App struct {
	cache               CacheManager
	storage             FileStorageManager
	events              EventBus
	logger              Logger
	resources           ResourceRepository
	resourceFieldValues ResourceFieldValueRepository
	widgetInstances     WidgetInstanceRepository
}

func NewApp(
	cache CacheManager,
	storage FileStorageManager,
	events EventBus,
	logger Logger,
	resources ResourceRepository,
	resourceFieldValues ResourceFieldValueRepository,
	widgetInstances WidgetInstanceRepository,
) (*App, error) {
	if cache == nil {
		return nil, errors.New("cache manager is nil")
	}

	if storage == nil {
		return nil, errors.New("file storage manager is nil")
	}

	if events == nil {
		return nil, errors.New("event bus is nil")
	}

	if logger == nil {
		return nil, errors.New("logger is nil")
	}

	if resources == nil {
		return nil, errors.New("resource repository is nil")
	}

	if resourceFieldValues == nil {
		return nil, errors.New("resource field value repository is nil")
	}

	if widgetInstances == nil {
		return nil, errors.New("widget instance repository is nil")
	}

	return &App{
		cache:               cache,
		storage:             storage,
		events:              events,
		logger:              logger,
		resources:           resources,
		resourceFieldValues: resourceFieldValues,
		widgetInstances:     widgetInstances,
	}, nil
}

func (a *App) CacheManager() CacheManager {
	return a.cache
}

func (a *App) Storage() FileStorageManager {
	return a.storage
}

func (a *App) EventBus() EventBus {
	return a.events
}

func (a *App) Logger() Logger {
	return a.logger
}

func (a *App) Resources() ResourceRepository {
	return a.resources
}

func (a *App) ResourceFieldValues() ResourceFieldValueRepository {
	return a.resourceFieldValues
}

func (a *App) WidgetInstances() WidgetInstanceRepository {
	return a.widgetInstances
}
