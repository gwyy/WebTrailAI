package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct{}

func NewConfig(file string) *viper.Viper {
	v := viper.New()
	v.SetConfigFile(file)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		if err = v.Unmarshal(&GlobalApp.Conf); err != nil {
			fmt.Println(err)
		}
	})
	if err = v.Unmarshal(&global.Conf); err != nil {
		panic(err)
	}
	return &Config{}
}
