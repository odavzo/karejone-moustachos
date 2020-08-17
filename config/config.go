package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Token string `json:"bot_token"`
}

func GetConf(file string) (err error, conf Config) {
	var configFile *os.File
	if configFile, err = os.Open(file); err == nil {
		defer configFile.Close()
		jsonParser := json.NewDecoder(configFile)
		err = jsonParser.Decode(&conf)

	}
	return err, conf

}
