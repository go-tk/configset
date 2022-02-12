package configstore

var (
	FsFactory          = &fsFactory
	EnvironmentFactory = &environmentFactory
)

type ConfigStore = configStore

var OpenConfigStore = openConfigStore
