package httpserver

import (
	"net/http"
	"testing"
)

func TestDefaultEnablesBuiltinMiddleware(t *testing.T) {
	o := Default()
	if !o.health || !o.logger || !o.requestID || !o.recovery {
		t.Error("Default() should enable all built-in middleware")
	}
}

func TestNoneDisablesBuiltinMiddleware(t *testing.T) {
	o := None()
	if o.health || o.logger || o.requestID || o.recovery {
		t.Error("None() should disable all built-in middleware")
	}
}

func TestWithoutMethods(t *testing.T) {
	cases := []struct {
		name   string
		apply  func(*Options) *Options
		field  func(*Options) bool
		others func(*Options) bool
	}{
		{
			name:  "health",
			apply: func(o *Options) *Options { return o.WithoutHealth() },
			field: func(o *Options) bool { return o.health },
		},
		{
			name:  "logger",
			apply: func(o *Options) *Options { return o.WithoutLogger() },
			field: func(o *Options) bool { return o.logger },
		},
		{
			name:  "requestID",
			apply: func(o *Options) *Options { return o.WithoutRequestID() },
			field: func(o *Options) bool { return o.requestID },
		},
		{
			name:  "recovery",
			apply: func(o *Options) *Options { return o.WithoutRecovery() },
			field: func(o *Options) bool { return o.recovery },
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := Default()
			updated := c.apply(base)

			if c.field(updated) {
				t.Errorf("Without%s did not disable %s", c.name, c.name)
			}
			if !c.field(base) {
				t.Errorf("original options were mutated for %s", c.name)
			}
		})
	}
}

func TestWithMiddlewareAppendsAndClones(t *testing.T) {
	mw := func(h http.Handler) http.Handler { return h }

	base := None()
	updated := base.WithMiddleware(mw)

	if len(updated.middleware) != 1 {
		t.Errorf("updated middleware = %d, want 1", len(updated.middleware))
	}
	if len(base.middleware) != 0 {
		t.Errorf("original middleware = %d, want 0 (immutable)", len(base.middleware))
	}

	updated = updated.WithMiddleware(mw)
	if len(updated.middleware) != 2 {
		t.Errorf("second WithMiddleware = %d, want 2", len(updated.middleware))
	}
}
