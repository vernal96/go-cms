package kernel

type AdapterCode string

const AdapterDefault AdapterCode = "default"

type AdapterDefaults struct {
	RepositoryAdapter AdapterCode
}

func (d AdapterDefaults) Merge(override AdapterDefaults) AdapterDefaults {
	result := d

	if override.RepositoryAdapter != "" {
		result.RepositoryAdapter = override.RepositoryAdapter
	}

	return result
}

func (d AdapterDefaults) WithBuiltInFallbacks() AdapterDefaults {
	if d.RepositoryAdapter == "" {
		d.RepositoryAdapter = AdapterDefault
	}

	return d
}

func ResolveAdapterDefaults(defaults ...AdapterDefaults) AdapterDefaults {
	result := AdapterDefaults{}

	for _, current := range defaults {
		result = result.Merge(current)
	}

	return result.WithBuiltInFallbacks()
}
