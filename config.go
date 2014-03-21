package didyouseethis

import (
	"errors"
	"io/ioutil"

	"gopkg.in/v1/yaml"
)

type Config struct {
	OAuth struct {
		Key    string
		Secret string
	}
	Keywords []string
}

func ReadConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	// validate that required fields were set
	if config.OAuth.Key == "" {
		return nil, errors.New("missing field: oauth key")
	}
	if config.OAuth.Secret == "" {
		return nil, errors.New("missing field: oauth secret")
	}
	if len(config.Keywords) == 0 {
		return nil, errors.New("missing field: keywords")
	}

	return &config, nil
}
