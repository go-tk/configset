package configset

var (
	NewFs          = &newFs
	GetEnvironment = &getEnvironment
)

type ConfigSet = configSet

func (cs *ConfigSet) IsLoaded() bool { return cs.raw != nil }
