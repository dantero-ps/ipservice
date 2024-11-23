package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	PostgresURL string `mapstructure:"POSTGRES_URL"`
	RedisURL    string `mapstructure:"REDIS_URL"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
	RIRs        []RIR  `mapstructure:"rirs"`
}

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Database string
}

type RIR struct {
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
}

func buildPostgresURL(cfg PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)
}

func buildRedisURL(cfg RedisConfig) string {
	return fmt.Sprintf("redis://%s:%s/%s",
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
}

func Load() (*Config, error) {
	// PostgreSQL defaults
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_NAME", "ipservice")
	viper.SetDefault("DB_SSLMODE", "disable")

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_DB", "0")

	// Server default
	viper.SetDefault("SERVER_PORT", ":8080")

	viper.AutomaticEnv()

	// Build PostgreSQL URL
	postgresConfig := PostgresConfig{
		User:     viper.GetString("DB_USER"),
		Password: viper.GetString("DB_PASSWORD"),
		Host:     viper.GetString("DB_HOST"),
		Port:     viper.GetString("DB_PORT"),
		Database: viper.GetString("DB_NAME"),
		SSLMode:  viper.GetString("DB_SSLMODE"),
	}

	// Build Redis URL
	redisConfig := RedisConfig{
		Host:     viper.GetString("REDIS_HOST"),
		Port:     viper.GetString("REDIS_PORT"),
		Database: viper.GetString("REDIS_DB"),
	}

	var config Config
	config.PostgresURL = buildPostgresURL(postgresConfig)
	config.RedisURL = buildRedisURL(redisConfig)
	config.ServerPort = viper.GetString("SERVER_PORT")

	// Default RIR configurations
	config.RIRs = []RIR{
		{Name: "ARIN", URL: "https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"},
		{Name: "RIPE", URL: "https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-latest"},
		{Name: "APNIC", URL: "https://ftp.apnic.net/stats/apnic/delegated-apnic-latest"},
		{Name: "LACNIC", URL: "https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-latest"},
		{Name: "AFRINIC", URL: "https://ftp.afrinic.net/stats/afrinic/delegated-afrinic-latest"},
	}

	return &config, nil
}
