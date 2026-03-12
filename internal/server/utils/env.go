package utils

import (
	"os"
	"slices"
)

var IsDev = loadEnv()

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
