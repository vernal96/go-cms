package kernal

type ProfileCode string

type Profile interface {
	Code() ProfileCode
	Modules() []Module
}
