package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	REST        RESTConf
	DB          DBConf
	RateLimiter RateLimiterConf
	Auth        AuthConf
}

type RESTConf struct {
	Host         string
	Port         string
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DBConf struct {
	User         string
	Password     string
	Name         string
	Host         string
	Port         string
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxIdleTime  time.Duration `mapstructure:"max_idle_time"`
}

type RateLimiterConf struct {
	Limit   int
	Enabled bool
}

type AuthConf struct {
	Host string
	Port string
}

func LoadConfig(path string) (Config, error) {
	viper.SetConfigFile(path)

	err := viper.ReadInConfig()
	if err != nil {
		return Config{}, fmt.Errorf("failed reading config: %w", err)
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
