package configset

var (
	FsFactory          = &fsFactory
	EnvironmentFactory = &environmentFactory
)

type ConfigSet = configSet

var OpenConfigSet = openConfigSet
