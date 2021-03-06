package config

import (
	"encoding/json"
	"io/ioutil"
)

type PriceProviderType string

const (
	Binance PriceProviderType = "Binance"
	Huobi   PriceProviderType = "Huobi"
)

type Config struct {
	PriceProvider   PriceProviderType
	NodeURL         string
	ChainId         string
	ControlContract string
	Timeout         int
	Ips             []string
}

func Load(filename string) (Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	config := Config{}
	if err := json.Unmarshal(file, &config); err != nil {
		return Config{}, err
	}
	return config, err
}
