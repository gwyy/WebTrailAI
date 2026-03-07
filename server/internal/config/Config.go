package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 定义接口
type Config interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	// 添加更多方法，如 GetFloat64、AllSettings 等，根据需要
	Unmarshal(target interface{}) error          // 用于反序列化到结构体
	WatchConfig(onChange func(e fsnotify.Event)) // 如果需要暴露热重载
}

type viperConfig struct {
	v *viper.Viper
}

func NewConfig(file string) Config {
	v := viper.New()
	v.SetConfigFile(file)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	//config 实体结构体
	appConfig := AppConfig{}
	cfg := &viperConfig{v: v}
	cfg.WatchConfig(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		if err = cfg.Unmarshal(&appConfig); err != nil {
			fmt.Println(err)
		}
	})
	// 初始 Unmarshal
	if err = cfg.Unmarshal(&appConfig); err != nil {
		panic(err)
	}
	return cfg
}

// 实现接口方法
func (c *viperConfig) GetString(key string) string {
	return c.v.GetString(key)
}

func (c *viperConfig) GetInt(key string) int {
	return c.v.GetInt(key)
}

func (c *viperConfig) GetBool(key string) bool {
	return c.v.GetBool(key)
}

func (c *viperConfig) Unmarshal(target interface{}) error {
	return c.v.Unmarshal(target)
}

func (c *viperConfig) WatchConfig(onChange func(e fsnotify.Event)) {
	c.v.WatchConfig()
	c.v.OnConfigChange(onChange)
}
