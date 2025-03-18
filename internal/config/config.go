package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DatabaseConfig `yaml:"db"`
	HostConfig     `yaml:"host"`
}
type HostConfig struct {
	Port   string `yaml:"port"` //`env:"PORT" env-default:"9000"`
	Domain string `yaml:"domain"`
}
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
}

func New(path string) (*Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
