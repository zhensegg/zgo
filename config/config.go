package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Postgres PostgresConfig `yaml:"postgres"`
}

type AppConfig struct {
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type PostgresConfig struct {
	URL      string `yaml:"url"`
	MaxConns int32  `yaml:"max_conns"`
	MinConns int32  `yaml:"min_conns"`
}

func Default() *Config {
	return &Config{
		App: AppConfig{
			Name: "zgo-app",
			Env:  "development",
			Host: "0.0.0.0",
			Port: 8080,
		},
		Postgres: PostgresConfig{
			URL:      "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
			MaxConns: 10,
			MinConns: 1,
		},
	}
}

func Load() (*Config, error) {
	cfg := Default()

	for _, path := range candidatePaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
		break
	}

	applyEnv(cfg)
	return cfg, nil
}

func candidatePaths() []string {
	paths := []string{"config.yaml", "config.yml", "zgo.yaml", "zgo.yml"}
	if dir := os.Getenv("ZGO_CONFIG_DIR"); dir != "" {
		return []string{
			dir + "/config.yaml", dir + "/config.yml",
			dir + "/zgo.yaml", dir + "/zgo.yml",
		}
	}
	return paths
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("ZGO_APP_NAME"); v != "" {
		cfg.App.Name = v
	}
	if v := os.Getenv("ZGO_APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("ZGO_APP_HOST"); v != "" {
		cfg.App.Host = v
	}
	if v := os.Getenv("ZGO_APP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.Port = n
		}
	}
	if v := os.Getenv("ZGO_POSTGRES_URL"); v != "" {
		cfg.Postgres.URL = strings.TrimSpace(v)
	}
	if v := os.Getenv("ZGO_POSTGRES_MAX_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			cfg.Postgres.MaxConns = int32(n)
		}
	}
	if v := os.Getenv("ZGO_POSTGRES_MIN_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			cfg.Postgres.MinConns = int32(n)
		}
	}
}

func (c *Config) Addr() string {
	host := c.App.Host
	if host == "" {
		host = "0.0.0.0"
	}
	return host + ":" + strconv.Itoa(c.App.Port)
}
