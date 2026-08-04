package shutdown

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Func func(ctx context.Context) error

type service struct {
	name string
	fn   Func
}

type Manager struct {
	services []service
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(name string, fn Func) {
	m.services = append(m.services, service{name: name, fn: fn})
}

func (m *Manager) Run(ctx context.Context) {
	m.run(ctx, 0, m.signalCh())
}

func (m *Manager) RunWithTimeout(ctx context.Context, timeout time.Duration) {
	m.run(ctx, timeout, m.signalCh())
}

func (m *Manager) signalCh() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	return quit
}

func (m *Manager) run(ctx context.Context, timeout time.Duration, quit <-chan os.Signal) {
	sig := <-quit
	slog.Info("received signal, shutting down", "signal", sig)

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	for i := len(m.services) - 1; i >= 0; i-- {
		svc := m.services[i]
		slog.Info("stopping service", "name", svc.name)

		if err := svc.fn(ctx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				slog.Error("service shutdown timed out", "name", svc.name)
			} else {
				slog.Error("service shutdown failed", "name", svc.name, "error", err)
			}
		} else {
			slog.Info("service stopped", "name", svc.name)
		}
	}

	slog.Info("all services stopped")
}
