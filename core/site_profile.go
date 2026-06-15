package core

type SiteProfile interface {
	Code() string
	Modules() []Module
}
