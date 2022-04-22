package gopatch

var defaultConfig  PatcherConfig

var defaultPatcher Patcher

func init() {

  defaultPatcher = *New(defaultConfig)
}

// Default returns an instance of the default patcher.
func Default() Patcher {

  return defaultPatcher
}