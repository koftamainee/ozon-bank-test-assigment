package config

import (
	"os"
	"strings"
)

type Source interface {
	Name() string
	Read(key string) (string, bool)
	FormatKey(key string) string
}

var defaultSource Source = &envSource{}

func SetSource(s Source) {
	defaultSource = s
}

func SourceName() string {
	return defaultSource.Name()
}

func readSource(key string) (string, bool) {
	return defaultSource.Read(key)
}

func formatKey(key string) string {
	return defaultSource.FormatKey(key)
}

type envSource struct{}

func (e *envSource) Name() string {
	return "env"
}

func (e *envSource) Read(key string) (string, bool) {
	envKey := strings.ReplaceAll(strings.ToUpper(key), ".", "_")
	val, ok := os.LookupEnv(envKey)
	return val, ok
}

func (e *envSource) FormatKey(key string) string {
	return strings.ReplaceAll(strings.ToUpper(key), ".", "_")
}
