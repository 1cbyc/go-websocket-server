package config

import (
	"os"
)

type Config struct {
	Addr     string
	LogLevel string
	DBDSN    string
}

func Load() *Config {
	addr := os.Getenv("WS_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	logLevel := os.Getenv("WS_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	dbDsn := os.Getenv("WS_DB_DSN")
	if dbDsn == "" {
		dbDsn = "file:messages.db?_foreign_keys=on"
	}
	return &Config{
		Addr:     addr,
		LogLevel: logLevel,
		DBDSN:    dbDsn,
	}
}
