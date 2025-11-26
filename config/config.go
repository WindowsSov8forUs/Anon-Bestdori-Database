package config

import (
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
		if viper.IsSet(path) {
			continue
		}
		envKey := "ANON_DATABASE_" + strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
		if val := os.Getenv(envKey); val != "" {
			viper.Set(path, val)
		}
	}
}

func Load() (*Config, error) {
	viper.Reset()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 优先 yaml
	err := viper.ReadInConfig()

	var hasYaml bool
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		fmt.Println("config.yaml not found, using ENV and defaults.")
		hasYaml = false
	} else {
		hasYaml = true
	}

	applyEnvOverrides()

	setDefaults()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if !hasYaml {
		if err := viper.WriteConfigAs("config.yaml"); err != nil {
			fmt.Printf("failed to create config.yaml: %v\n", err)
		} else {
			fmt.Println("Created config.yaml with current configuration.")
		}
	}

	return cfg, nil
}
func (c *Config) Reload() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	*c = *cfg
	return nil
}
