package config

import (
	"github.com/FleekHQ/space-poc/core/env"
)

type mapConfig struct {
	configStr map[string]string
	configInt map[string]int
}

func NewMap(env env.SpaceEnv) Config {
	configStr := make(map[string]string)
	configInt := make(map[string]int)

	// default values
	configStr[SpaceStorePath] = "~/.fleek-space"
	configStr[TextileHubTarget] = "textile-hub-dev.fleek.co:3006"
	configStr[TextileThreadsTarget] = "textile-hub-dev.fleek.co:3006"
	configStr[MountFuseDrive] = "false"
	configStr[FuseMountPath] = "~/"
	configStr[FuseDriveName] = "FleekSpace"
	configInt[SpaceServerPort] = 9999

	c := mapConfig{
		configStr: configStr,
		configInt: configInt,
	}

	return c
}

func (m mapConfig) GetString(key string, defaultValue interface{}) string {
	if val, exists := m.configStr[key]; exists {
		return val
	}

	if stringValue, ok := defaultValue.(string); ok {
		return stringValue
	}

	return ""
}

func (m mapConfig) GetInt(key string, defaultValue interface{}) int {
	if val, exists := m.configInt[key]; exists {
		return val
	}

	if intVal, ok := defaultValue.(int); ok {
		return intVal
	}

	return 0
}
