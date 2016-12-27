package config

import (
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

// Config is ...
type Config struct {
	ConsumerKey       string `toml:"consumer_key"`
	ConsumerSecret    string `toml:"consumer_secret"`
	AccessToken       string `toml:"access_token"`
	AccessTokenSecret string `toml:"access_token_secret"`
}

// LoadConfig is ...
func LoadConfig(path string) (*Config, error) {
	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config *Config
	_, err = toml.Decode(string(configFile), &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
