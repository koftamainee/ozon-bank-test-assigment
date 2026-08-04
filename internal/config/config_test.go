package config

import (
	"reflect"
	"testing"
	"time"
)

func TestCamelToSnake(t *testing.T) {
	cases := map[string]string{
		"FooBar":          "foo_bar",
		"HTTPServer":      "http_server",
		"XMLParser":       "xml_parser",
		"ABC":             "abc",
		"already_snake":   "already_snake",
		"single":          "single",
		"ConnMaxLifetime": "conn_max_lifetime",
	}

	for in, want := range cases {
		if got := camelToSnake(in); got != want {
			t.Errorf("camelToSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

type nestedCfg struct {
	Server struct {
		Host string `default:"localhost"`
		Port int    `default:"8080"`
	}
	Debug bool    `default:"false"`
	Rate  float64 `default:"0.5"`
}

type durationCfg struct {
	Timeout time.Duration `default:"30s"`
}

type requiredCfg struct {
	Token string
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SERVER_HOST", "example.com")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DEBUG", "true")
	t.Setenv("RATE", "1.5")

	var cfg nestedCfg
	if err := Load(&cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "example.com" {
		t.Errorf("Host = %q, want example.com", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Server.Port)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
	if cfg.Rate != 1.5 {
		t.Errorf("Rate = %v, want 1.5", cfg.Rate)
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	var cfg nestedCfg
	if err := Load(&cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("Host = %q, want default localhost", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Port = %d, want default 8080", cfg.Server.Port)
	}
	if cfg.Debug {
		t.Error("Debug = true, want default false")
	}
	if cfg.Rate != 0.5 {
		t.Errorf("Rate = %v, want default 0.5", cfg.Rate)
	}
}

func TestLoadMissingFieldReturnsError(t *testing.T) {
	var cfg requiredCfg
	err := Load(&cfg)
	if err == nil {
		t.Fatal("Load() = nil, want error for missing field without default")
	}
}

func TestLoadDuration(t *testing.T) {
	t.Setenv("TIMEOUT", "1m30s")

	var cfg durationCfg
	if err := Load(&cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Timeout != 90*time.Second {
		t.Errorf("Timeout = %v, want 1m30s", cfg.Timeout)
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Setenv("TIMEOUT", "not-a-duration")

	var cfg durationCfg
	if err := Load(&cfg); err == nil {
		t.Fatal("Load() = nil, want error for invalid duration")
	}
}

func TestLoadInvalidInt(t *testing.T) {
	t.Setenv("SERVER_PORT", "abc")

	var cfg nestedCfg
	if err := Load(&cfg); err == nil {
		t.Fatal("Load() = nil, want error for invalid int")
	}
}

func TestLoadInvalidBool(t *testing.T) {
	t.Setenv("DEBUG", "maybe")

	var cfg nestedCfg
	if err := Load(&cfg); err == nil {
		t.Fatal("Load() = nil, want error for invalid bool")
	}
}

func TestLoadInvalidFloat(t *testing.T) {
	t.Setenv("RATE", "one point five")

	var cfg nestedCfg
	if err := Load(&cfg); err == nil {
		t.Fatal("Load() = nil, want error for invalid float")
	}
}

func TestLoadValidatesEnv(t *testing.T) {
	t.Setenv("ENV", "staging")

	var cfg DefaultConfig
	if err := Load(&cfg); err == nil {
		t.Fatal("Load() = nil, want error for unsupported env")
	}
}

func TestLoadAcceptsDevAndProd(t *testing.T) {
	for _, env := range []string{"dev", "prod"} {
		t.Setenv("ENV", env)
		var cfg DefaultConfig
		if err := Load(&cfg); err != nil {
			t.Errorf("Load() with env %q: unexpected error %v", env, err)
		}
		if cfg.GetEnv() != env {
			t.Errorf("GetEnv() = %q, want %q", cfg.GetEnv(), env)
		}
	}
}

func TestMustLoadPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustLoad() did not panic")
		}
	}()

	var cfg requiredCfg
	MustLoad(&cfg)
}

func TestMustLoadSucceeds(t *testing.T) {
	t.Setenv("ENV", "dev")

	defer func() {
		if recover() != nil {
			t.Fatal("MustLoad() panicked unexpectedly")
		}
	}()

	var cfg DefaultConfig
	MustLoad(&cfg)
}

type fakeSource struct {
	values map[string]string
}

func (f *fakeSource) Name() string {
	return "fake"
}

func (f *fakeSource) Read(key string) (string, bool) {
	v, ok := f.values[key]
	return v, ok
}

func (f *fakeSource) FormatKey(key string) string {
	return key
}

func TestSetSource(t *testing.T) {
	prev := defaultSource
	defer SetSource(prev)

	SetSource(&fakeSource{values: map[string]string{
		"server.host": "fake.example",
		"server.port": "1234",
		"debug":       "true",
		"rate":        "2.5",
	}})

	var cfg nestedCfg
	if err := Load(&cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "fake.example" {
		t.Errorf("Host = %q, want fake.example", cfg.Server.Host)
	}
	if cfg.Server.Port != 1234 {
		t.Errorf("Port = %d, want 1234", cfg.Server.Port)
	}
}

func TestSourceName(t *testing.T) {
	prev := defaultSource
	defer SetSource(prev)

	if got := SourceName(); got != "env" {
		t.Errorf("SourceName() = %q, want env", got)
	}

	SetSource(&fakeSource{})
	if got := SourceName(); got != "fake" {
		t.Errorf("SourceName() = %q, want fake", got)
	}
}

func TestFields(t *testing.T) {
	var cfg nestedCfg
	fields := Fields(&cfg)

	want := map[string]FieldInfo{
		"SERVER_HOST": {
			Key:        "server.host",
			HasDefault: true,
			Default:    "localhost",
			SourceKey:  "SERVER_HOST",
		},
		"SERVER_PORT": {
			Key:        "server.port",
			HasDefault: true,
			Default:    "8080",
			SourceKey:  "SERVER_PORT",
		},
		"DEBUG": {
			Key:        "debug",
			HasDefault: true,
			Default:    "false",
			SourceKey:  "DEBUG",
		},
		"RATE": {
			Key:        "rate",
			HasDefault: true,
			Default:    "0.5",
			SourceKey:  "RATE",
		},
	}

	if len(fields) != len(want) {
		t.Fatalf("Fields() returned %d fields, want %d: %+v", len(fields), len(want), fields)
	}

	for _, f := range fields {
		w, ok := want[f.SourceKey]
		if !ok {
			t.Errorf("unexpected field %+v", f)
			continue
		}
		if f.Key != w.Key || f.HasDefault != w.HasDefault || f.Default != w.Default {
			t.Errorf("field %s = %+v, want %+v", f.SourceKey, f, w)
		}
	}
}

func TestSetFieldUnsupportedType(t *testing.T) {
	var v complex64
	if err := setField(reflect.ValueOf(&v).Elem(), "x"); err == nil {
		t.Fatal("setField() = nil, want error for unsupported type")
	}
}
