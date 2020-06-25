package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/log"
	"github.com/creamdog/gonfig"
)

const (
	JsonConfigFileName   = "space.json"
	SpaceServerPort      = "space/rpcPort"
	SpaceStorePath       = "space/storePath"
	TextileHubTarget     = "space/textileHubTarget"
	TextileThreadsTarget = "space/textileThreadsTarget"
	MountFuseDrive       = "space/mountFuseDrive"
	FuseMountPath        = "space/fuseMountPath"
	FuseDriveName        = "space/fuseDriveName"
	SpaceServicesAPIURL  = "space/servicesApiUrl"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

// Config used to fetch config information
type Config interface {
	GetString(key string, defaultValue interface{}) string
	GetInt(key string, defaultValue interface{}) int
}

// standardConfig implements Config
// It loads its config information from the space.json file
type standardConfig struct {
	cfg gonfig.Gonfig
}

func New(env env.SpaceEnv) Config {
	wd := env.WorkingFolder()
	f, err := os.Open(wd + "/" + JsonConfigFileName)
	if err != nil {
		// TODO: this may turn into a fatal panic error
		log.Info("could not find space.json file in " + wd + ", using defaults")
	}

	defer f.Close()
	config, err := gonfig.FromJson(f)
	if err != nil {
		log.Info("could not read space.json file, using defaults")
	}

	c := standardConfig{
		cfg: config,
	}

	return c
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c standardConfig) GetString(key string, defaultValue interface{}) string {
	if c.cfg == nil {
		return ""
	}
	v, err := c.cfg.GetString(key, defaultValue)
	if err != nil {
		log.Error(fmt.Sprintf("error getting key %s from config", key), err)
		return ""
	}
	log.Debug("Getting conf " + key + ": " + v)

	return v
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c standardConfig) GetInt(key string, defaultValue interface{}) int {
	if c.cfg == nil {
		return 0
	}
	v, err := c.cfg.GetInt(key, defaultValue)
	if err != nil {
		log.Error(fmt.Sprintf("error getting key %s from config", key), err)
		return 0
	}

	return v
}
