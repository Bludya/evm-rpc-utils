package utils

import (
	"os"

	"gopkg.in/yaml.v2"
)

type RpcConfig struct {
	Url1  string `yaml:"url1"`
	Url2  string `yaml:"url2"`
	Block int64  `yaml:"block"`
}

func GetConf() (RpcConfig, error) {
	yamlFile, err := os.ReadFile("debugToolsConfig.yaml")
	if err != nil {
		return RpcConfig{}, err
	}

	c := RpcConfig{}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return RpcConfig{}, err
	}

	return c, nil
}
