package pg

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestHealthCheckPingOK(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectPing()

	check := HealthCheck(mock)
	if err := check(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestHealthCheckPingError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectPing().WillReturnError(errBoom)

	check := HealthCheck(mock)
	if err := check(context.Background()); !errors.Is(err, errBoom) {
		t.Fatalf("HealthCheck() error = %v, want %v", err, errBoom)
	}
}
