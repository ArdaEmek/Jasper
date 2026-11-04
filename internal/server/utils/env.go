package utils

import (
	"os"
	"slices"
	"strconv"
)

var (
	IsDev          = loadEnv()
	IsHTTP3Enabled = loadBoolEnv("HTTP3_ENABLED", true)
)

func loadEnv() bool {
	env := os.Getenv("APP_ENV")
	devStrings := []string{
		"dev",
		"development",
		"local",
		"debug",
		"test",
		"testing",
		"sandbox",
		"staging",
		"devel",
	}

	if slices.Contains(devStrings, env) {
		return true
	}

	return false
}

func loadBoolEnv(name string, defaultValue bool) bool {
	value, ok := os.LookupEnv(name)
	if !ok {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}
