package configset

var (
	FsFactory          = &fsFactory
	EnvironmentFactory = &environmentFactory
)

type ConfigSet = configSet

func (cs *ConfigSet) IsLoaded() bool { return cs.raw != nil }
