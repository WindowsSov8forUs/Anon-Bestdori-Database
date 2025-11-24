package config

import (
	"bytes"
	"fmt"
	"os"

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

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 创建默认配置文件
			viper.ReadConfig(bytes.NewBuffer([]byte(DEFAULT_CONFIG)))
			if err := viper.WriteConfigAs("config.yaml"); err != nil {
				return nil, err
			}
			fmt.Print("Default configuration file config.yaml has been created. Please edit it and restart the program.\n")
			os.Exit(0)
		}
		return nil, err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Reload() error {
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config file: %w", err)
	}
	return viper.Unmarshal(c)
}
