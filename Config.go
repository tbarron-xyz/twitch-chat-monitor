package main

import (
	"encoding/json"
	"io/ioutil"
)

func loadConfig() *Config {
	var configString, err = ioutil.ReadFile("./config.json")
	if err != nil {
		panic("Failed to load config file - failed to read file")
	}
	var config = &Config{}
	err = json.Unmarshal(configString, config)
	if err != nil {
		panic("Failed to load config file - not valid JSON")
	}
	return config
}

type Config struct {
	TwitchOauthPassword string
	TwitchNick          string
	TwitchApiClientId   string
}
