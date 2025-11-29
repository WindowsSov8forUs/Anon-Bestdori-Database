package config

import "github.com/spf13/viper"

func setDefaults() {
	for _, path := range configPaths {
		if viper.IsSet(path) {
			continue
		}
		var defVal any
		switch path {
		case "mongo.uri":
			defVal = "mongodb://localhost:27017/"
		case "log.level":
			defVal = "info"
		case "api.timeout":
			defVal = 5
		case "api.proxy":
			defVal = ""
		case "api.retry":
			defVal = 5
		case "api.gap":
			defVal = 10
		case "server.host":
			defVal = "0.0.0.0"
		case "server.port":
			defVal = "8080"
		}
		viper.Set(path, defVal)
	}
}
