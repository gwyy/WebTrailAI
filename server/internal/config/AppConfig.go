package config

type AppConfig struct {
	Mode    string  `yaml:"mode" mapstructure:"mode"`
	Version string  `yaml:"version" mapstructure:"version"`
	Name    string  `yaml:"name" mapstructure:"name"`
	Secret  string  `yaml:"secret" mapstructure:"secret"`
	Gin     Gin     `yaml:"gin" mapstructure:"gin"`
	Log     Log     `yaml:"log" mapstructure:"log"`
	AI      AI      `yaml:"ai" mapstructure:"ai"`
	Summary Summary `yaml:"summary" mapstructure:"summary"`
}

type Db struct {
	Filedir string `yaml:"filedir" mapstructure:"filedir"`
}

type Gin struct {
	Host         string `yaml:"host" mapstructure:"host"`
	Prefix       string `yaml:"prefix" mapstructure:"prefix"`
	IP           string `yaml:"ip" mapstructure:"ip"`
	Port         int    `yaml:"port" mapstructure:"port"`
	Timeout      int    `yaml:"timeout" mapstructure:"timeout"`
	Pprof        bool   `yaml:"pprof" mapstructure:"pprof"`
	ReadTimeout  string `yaml:"readtimeout" mapstructure:"readtimeout"`
	WriteTimeout string `yaml:"writetimeout" mapstructure:"writetimeout"`
}

type Log struct {
	Level      int    `yaml:"level" mapstructure:"level"`
	Path       string `yaml:"path" mapstructure:"path"`
	MaxSize    int    `yaml:"max-size" mapstructure:"max-size"`
	MaxAge     int    `yaml:"max-age" mapstructure:"max-age"`
	MaxBackups int    `yaml:"max-backups" mapstructure:"max-backups"`
	Compress   bool   `yaml:"compress" mapstructure:"compress"`
}

type AI struct {
	DashScopeAPIKey string `yaml:"dashscope_api_key" mapstructure:"dashscope_api_key"`
	BaseURL         string `yaml:"base-url" mapstructure:"base-url"`
	Model           string `yaml:"model" mapstructure:"model"`
	TimeoutSeconds  int    `yaml:"timeout-seconds" mapstructure:"timeout-seconds"`
}

type Summary struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	Cron         string `yaml:"cron" mapstructure:"cron"`
	DebugEnabled bool   `yaml:"debug-enabled" mapstructure:"debug-enabled"`
	DebugToken   string `yaml:"debug-token" mapstructure:"debug-token"`
}
