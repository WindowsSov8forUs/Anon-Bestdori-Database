package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type MongoConfig struct {
	URI string `mapstructure:"uri"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type APIConfig struct {
	Timeout int    `mapstructure:"timeout"`
	Proxy   string `mapstructure:"proxy"`
}

type Config struct {
	Mongo  MongoConfig  `mapstructure:"mongo"`
	Log    LogConfig    `mapstructure:"log"`
	API    APIConfig    `mapstructure:"api"`
	Server ServerConfig `mapstructure:"server"`
}

var configPaths = []string{
	"mongo.uri",
	"log.level",
	"api.timeout",
	"api.proxy",
	"server.host",
	"server.port",
}

func applyEnvOverrides() {
	for _, path := range configPaths {
		if !viper.IsSet(path) {
			envKey := "ANON_DATABASE_" + strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
			if val := os.Getenv(envKey); val != "" {
				viper.Set(path, val)
			}
		}
	}
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 先加载默认配置
	viper.ReadConfig(bytes.NewBuffer([]byte(DEFAULT_CONFIG)))

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("No config.yaml found, using defaults and environment variables. You can create config.yaml to override.")
		} else {
			return nil, err
		}
	}

	applyEnvOverrides()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
func (c *Config) Reload() error {
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to reload config file: %w", err)
		}
	}

	applyEnvOverrides()

	return viper.Unmarshal(c)
}
