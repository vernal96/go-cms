package core

type Site struct {
	ID       int64
	Code     string
	Domain   string
	Settings map[string]any
}
