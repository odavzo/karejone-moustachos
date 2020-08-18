package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Token               string `json:"bot_token"`
	MoustachosChannelId string `json:"moustachos_channel_id"`
	MoustachosGuildId   string `json:"moustachos_guild_id"`
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
