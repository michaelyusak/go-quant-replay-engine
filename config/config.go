package config

import (
	"fmt"
	"os"

	hConfig "github.com/michaelyusak/go-helper/config"
	hEntity "github.com/michaelyusak/go-helper/entity"
)

type ServiceConfig struct {
	Port           string           `json:"port"`
	GracefulPeriod hEntity.Duration `json:"graceful_period"`
	Db             hEntity.DBConfig `json:"db"`
}

type LogConfig struct {
	Level string `json:"level"`
	Dir   string `json:"dir"`
}

type CorsConfig struct {
	AllowedOrigins []string `json:"allowed_origins"`
}

type BinanceHttpConfig struct {
	FapiBaseUrl string `json:"fapi_base_url"`
}

type AdapterConfig struct {
	BinanceHttp BinanceHttpConfig `json:"binance_http"`
}

type AppConfig struct {
	Service ServiceConfig `json:"service"`
	Log     LogConfig     `json:"log"`
	Cors    CorsConfig    `json:"cors"`
	Adapter AdapterConfig `json:"adapter"`
}

func Init() (AppConfig, error) {
	configFilePath := os.Getenv("GO_QUANT_REPLAY_ENGINE_CONFIG")

	var conf AppConfig

	conf, err := hConfig.InitFromJson[AppConfig](configFilePath)
	if err != nil {
		return conf, fmt.Errorf("[config][Init][hConfig.InitFromJson] Failed to init config from json: %w", err)
	}

	return conf, nil
}
