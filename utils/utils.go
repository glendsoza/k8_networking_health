package utils

import (
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

func GetLogger() *zerolog.Logger {
	level, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = zerolog.InfoLevel

	}
	l := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Level(level)
	return &l
}

func ParseEnvElseDefault(envName string, v int) int {
	value := v
	envString := os.Getenv(envName)
	if envString != "" {
		envValue, err := strconv.Atoi(envString)
		if err != nil {
			value = envValue
		}
	}
	return value
}
