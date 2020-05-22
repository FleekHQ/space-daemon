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
	JsonConfigFileName = "space.json"
	SpaceFolderPath    = "space/folderPath"
	SpaceServerPort    = "space/rpcPort"
	SpaceStorePath     = "space/storePath"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

type Config struct {
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

	c := Config{
		cfg: config,
	}

	return c
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c Config) GetString(key string, defaultValue interface{}) string {
	if c.cfg == nil {
		return ""
	}
	v, err := c.cfg.GetString(key, defaultValue)
	if err != nil {
		log.Error(fmt.Sprintf("error getting key %s from config", key), err)
		return ""
	}

	return v
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c Config) GetInt(key string, defaultValue interface{}) int {
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
