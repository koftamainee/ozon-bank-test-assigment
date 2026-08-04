package httpserver

import "net/http"

type Options struct {
	health     bool
	logger     bool
	requestID  bool
	recovery   bool
	middleware []func(http.Handler) http.Handler
}

func Default() *Options {
	return &Options{
		health:    true,
		logger:    true,
		requestID: true,
		recovery:  true,
	}
}

func None() *Options {
	return &Options{}
}

func (o *Options) WithoutHealth() *Options {
	clone := *o
	clone.health = false
	return &clone
}

func (o *Options) WithoutLogger() *Options {
	clone := *o
	clone.logger = false
	return &clone
}

func (o *Options) WithoutRequestID() *Options {
	clone := *o
	clone.requestID = false
	return &clone
}

func (o *Options) WithoutRecovery() *Options {
	clone := *o
	clone.recovery = false
	return &clone
}

func (o *Options) WithMiddleware(mw func(http.Handler) http.Handler) *Options {
	clone := *o
	clone.middleware = append([]func(http.Handler) http.Handler(nil), clone.middleware...)
	clone.middleware = append(clone.middleware, mw)
	return &clone
}
