package main

import (
	"io/ioutil"

	"strings"

	"github.com/Eun/domwatch/fcgi/api1"
	hjson "github.com/hjson/hjson-go"
	"github.com/mitchellh/mapstructure"
)

type Database struct {
	Provider *string
	Host     *string
	User     *string
	Password *string
	Database *string
}

type Config struct {
	api1.Config `mapstructure:",squash"`
	Database    Database
}

// NewConfigFromMap creates a new config instance from a map
func NewConfigFromMap(dat map[string]interface{}) (*Config, error) {
	var config Config
	var err error
	if dat != nil {
		err = mapstructure.Decode(dat, &config)
		if err != nil {
			return nil, err
		}
	}
	err = config.SetDefaults()
	return &config, err
}

// NewConfigFromFile returns a Config instance for a file
func NewConfigFromFile(configFile string) (*Config, error) {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var dat map[string]interface{}
	if err := hjson.Unmarshal(bytes, &dat); err != nil {
		return nil, err
	}

	return NewConfigFromMap(dat)
}

func (config *Config) SetDefaults() error {
	if config.Database.Provider == nil {
		config.Database.Provider = new(string)
		*config.Database.Provider = "sqlite3"
	} else {
		*config.Database.Provider = strings.ToLower(*config.Database.Provider)
	}
	if config.Database.Database == nil {
		config.Database.Database = new(string)
		*config.Database.Database = "domwatch.db"
	}

	if config.Database.Host == nil {
		config.Database.Host = new(string)
		*config.Database.Host = ""
	}

	if config.Database.User == nil {
		config.Database.User = new(string)
		*config.Database.User = ""
	}
	if config.Database.Password == nil {
		config.Database.Password = new(string)
		*config.Database.Password = ""
	}

	return nil
}
