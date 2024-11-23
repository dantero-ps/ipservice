package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	PostgresURL string `mapstructure:"POSTGRES_URL"`
	RedisURL    string `mapstructure:"REDIS_URL"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
	RIRs        []RIR  `mapstructure:"rirs"`
}

type RIR struct {
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", ":8080")
	viper.SetDefault("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/ipservice?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")

	viper.AutomaticEnv()

	var config Config

	// Default RIR configurations
	config.RIRs = []RIR{
		{Name: "ARIN", URL: "https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"},
		{Name: "RIPE", URL: "https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-latest"},
		{Name: "APNIC", URL: "https://ftp.apnic.net/stats/apnic/delegated-apnic-latest"},
		{Name: "LACNIC", URL: "https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-latest"},
		{Name: "AFRINIC", URL: "https://ftp.afrinic.net/stats/afrinic/delegated-afrinic-latest"},
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
