package httpserver

import (
	"net/http"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/http/health"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/http/middleware"
)

func New(cfg Config, opts *Options) (*http.ServeMux, *http.Server, *health.Health) {
	if opts == nil {
		opts = Default()
	}

	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 10 * time.Second
	}

	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 10 * time.Second
	}

	mux := http.NewServeMux()
	h := health.New(5 * time.Second)

	if opts.health {
		mux.HandleFunc("GET /healthz", h.Liveness())
		mux.HandleFunc("GET /readyz", h.Readiness())
	}

	var hdl http.Handler = mux

	if opts.recovery {
		hdl = middleware.Recovery(hdl)
	}
	if opts.requestID {
		hdl = middleware.RequestID(hdl)
	}
	if opts.logger {
		hdl = middleware.Logger(hdl)
	}
	for _, mw := range opts.middleware {
		hdl = mw(hdl)
	}

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      hdl,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	return mux, srv, h
}
