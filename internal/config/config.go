package config

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration for the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	AppEnv string `mapstructure:"APP_ENV"`

	HTTPAddr string `mapstructure:"HTTP_ADDR"`
	HTTPPort int    `mapstructure:"HTTP_PORT"`

	PGDSN             string        `mapstructure:"PG_DSN"`
	PGMaxOpenConns    int           `mapstructure:"PG_MAX_OPEN_CONNS"`
	PGMaxIdleConns    int           `mapstructure:"PG_MAX_IDLE_CONNS"`
	PGConnMaxLifetime time.Duration `mapstructure:"PG_CONN_MAX_LIFETIME"`

	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	JWTSecretFile string `mapstructure:"JWT_SECRET_FILE"`

	RateLimitRPMDefault int `mapstructure:"RATE_LIMIT_RPM_DEFAULT"`

	OtelExporterOTLPEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OtelServiceName          string `mapstructure:"OTEL_SERVICE_NAME"`

	CORSAllowedOrigins []string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	VendorBaseURL string `mapstructure:"VENDOR_BASE_URL"`
	VendorToken   string `mapstructure:"VENDOR_TOKEN"`

	Debug bool `mapstructure:"DEBUG"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (config Config, err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err = viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return
		}
	}

	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("HTTP_ADDR", "0.0.0.0")
	viper.SetDefault("HTTP_PORT", 8080)
	viper.SetDefault("PG_MAX_OPEN_CONNS", 25)
	viper.SetDefault("PG_MAX_IDLE_CONNS", 25)
	viper.SetDefault("PG_CONN_MAX_LIFETIME", "5m")
	viper.SetDefault("RATE_LIMIT_RPM_DEFAULT", 100)
	viper.SetDefault("OTEL_SERVICE_NAME", "go-api")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"*"})
	viper.SetDefault("DEBUG", false)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, key := range []string{"PG_DSN", "REDIS_PASSWORD", "VENDOR_TOKEN", "JWT_SECRET_FILE"} {
		_ = viper.BindEnv(key)
	}

	err = viper.Unmarshal(&config)
	return
}
