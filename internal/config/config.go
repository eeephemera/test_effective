package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Address string `mapstructure:"address"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.DBName)
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Timeout  time.Duration  `mapstructure:"timeout"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// continue if not found
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &cfg, nil
}
