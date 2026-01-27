package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
	Addr     string `mapstructure:"addr"     yaml:"addr"`
}

type Config struct {
	Addr        string   `mapstructure:"addr"        yaml:"addr"`
	MetricsAddr string   `mapstructure:"metrics_addr" yaml:"metrics_addr"`
	DB          DBConfig `mapstructure:"db"          yaml:"db"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("../config")
		viper.AddConfigPath("../../config")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed read config: %w", err)
		}
		log.Println("Warning: config file not found, use default value")
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed parse: %w", err)
	}

	setDefaults(&cfg)

	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.MetricsAddr == "" {
		cfg.MetricsAddr = ":9090"
	}
	if cfg.DB.Addr == "" {
		cfg.DB.Addr = "neo4j://localhost:5432"
	}
	if cfg.DB.Username == "" {
		cfg.DB.Username = "neo4j"
	}
}
