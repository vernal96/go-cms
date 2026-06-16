package core

type Site struct {
	ID          int64
	ProfileCode string
	Domain      string
	Locale      string
	Settings    map[string]any
}
