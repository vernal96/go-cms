package core

type ModuleRequirements struct {
	CacheStores []CacheStoreName
	FileDisks   []FileDisk
}

type ModuleWithRequirements interface {
	Requirements() ModuleRequirements
}
