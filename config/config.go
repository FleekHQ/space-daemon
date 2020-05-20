package config

import (
	"errors"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/log"
	"github.com/creamdog/gonfig"
	"os"
)

const (
	JsonConfigFileName = "space.json"
	SpaceFolderPath = "space/folderPath"
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
		log.Info("could not find space.json.example file, using defaults")
	}

	defer f.Close()
	config, err := gonfig.FromJson(f)
	if err != nil {
		log.Info("could not read space.json.example file, using defaults")
	}

	c := Config{
		cfg: config,
	}

	return c
}
// Gets the configuration value given a path in the json config file
func (c Config) GetString(key string, defaultValue interface{}) (string, error) {
	if c.cfg == nil {
		return "", ErrConfigNotLoaded
	}
	return c.cfg.GetString(key, defaultValue)
}


