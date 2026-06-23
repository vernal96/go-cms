package core

import (
	"errors"
	"fmt"
	"strings"
)

type runtimeControllerRegistry struct {
	controllers      *[]Controller
	controllerRoutes map[string]struct{}
}

func (r *runtimeControllerRegistry) Register(controller Controller) error {
	if controller == nil {
		return errors.New("controller is nil")
	}

	routes := controller.Routes()
	if len(routes) == 0 {
		return errors.New("controller routes are empty")
	}

	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		key, err := r.routeKey(route)
		if err != nil {
			return err
		}

		if _, exists := r.controllerRoutes[key]; exists {
			return fmt.Errorf("route %q is already registered", key)
		}

		keys = append(keys, key)
	}

	for _, key := range keys {
		r.controllerRoutes[key] = struct{}{}
	}

	*r.controllers = append(*r.controllers, controller)

	return nil
}

func (r *runtimeControllerRegistry) All() []Controller {
	controllers := make([]Controller, len(*r.controllers))
	copy(controllers, *r.controllers)

	return controllers
}

func (r *runtimeControllerRegistry) routeKey(route Route) (string, error) {
	if route.Method == "" {
		return "", errors.New("route method is empty")
	}

	if route.Path == "" {
		return "", errors.New("route path is empty")
	}

	if !strings.HasPrefix(route.Path, "/") {
		return "", fmt.Errorf("route path %q must start with /", route.Path)
	}

	if route.Handler == nil {
		return "", fmt.Errorf("route %q handler is nil", route.Path)
	}

	return string(route.Method) + " " + route.Path, nil
}

var _ ControllerRegistry = (*runtimeControllerRegistry)(nil)
