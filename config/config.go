package config

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type AppConfig struct {
	App struct {
		Name string `mapstructure:"NAME"`
		Port string `mapstructure:"PORT"`
	}

	DATABASE struct {
		Postgres struct {
			DSN string `mapstructure:"URL"`
		}
		Redis struct {
			Addr     string `mapstructure:"ADDR"`
			Password string `mapstructure:"PASSWORD"`
		}
		Mongo struct {
			Url string `mapstructure:"URL"`
		}
	}

	MAILTRAP struct {
		SMTPHost string `mapstructure:"SMTP_HOST"`
		SMTPPort int    `mapstructure:"SMTP_PORT"`
		Username string `mapstructure:"USERNAME"`
		Password string `mapstructure:"PASSWORD"`
		From     string `mapstructure:"FROM"`
		TO       string `mapstructure:"TO"`
	}
}

var Conf *AppConfig

func LoadConfig() error {
	viper.SetConfigName("application")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("CHATAPP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	var config AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("error unmarshalling config: %w", err)
	}

	Conf = &config
	log.Info().Msg("configuration loaded...")
	return nil
}
