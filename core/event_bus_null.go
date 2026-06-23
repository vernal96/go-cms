package core

import "context"

type NullEventBus struct{}

func (b NullEventBus) Publish(ctx context.Context, event Event) error {
	return nil
}

func (b NullEventBus) Subscribe(name EventName, handler EventHandler) error {
	return nil
}

var _ EventBus = NullEventBus{}
