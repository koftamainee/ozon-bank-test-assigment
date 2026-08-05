package main

import "testing"

func TestExpandEnvRefs(t *testing.T) {
	t.Setenv("APP_PASSWORD", "s3cr#t")

	cases := map[string]string{
		"postgres://app_user:${APP_PASSWORD}@db:5432/forum":                        "postgres://app_user:s3cr#t@db:5432/forum",
		"postgres://app_user:${APP_PASSWORD}@localhost:5432/forum?sslmode=disable": "postgres://app_user:s3cr#t@localhost:5432/forum?sslmode=disable",
		"postgres://app_user:s3cr#t@db:5432/forum":                                 "postgres://app_user:s3cr#t@db:5432/forum",
		"${MISSING_VAR}":               "${MISSING_VAR}",
		"$APP_PASSWORD":                "$APP_PASSWORD",
		"postgres://u:p$w$d@db:5432/f": "postgres://u:p$w$d@db:5432/f",
		"plain":                        "plain",
	}

	for in, want := range cases {
		if got := expandEnvRefs(in); got != want {
			t.Errorf("expandEnvRefs(%q) = %q, want %q", in, got, want)
		}
	}
}
