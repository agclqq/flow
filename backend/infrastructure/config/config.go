package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config 汇总应用全部配置。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Cron     CronConfig     `mapstructure:"cron"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig PostgreSQL 连接配置。
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// CronConfig 定时任务配置。
type CronConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Schedule string `mapstructure:"schedule"`
}

// Load 从指定配置文件加载配置，并允许环境变量覆盖。
// 环境变量形如 FLOW_SERVER_PORT、FLOW_DATABASE_HOST 等。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("FLOW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Default 返回带默认值的配置，便于无配置文件时启动与测试。
func Default() *Config {
	return &Config{
		Server: ServerConfig{Port: 8080},
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			User:    "postgres",
			DBName:  "flow",
			SSLMode: "disable",
		},
		Cron: CronConfig{Enabled: false},
	}
}
