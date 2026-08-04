package pg

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

var errBoom = errors.New("boom")

func TestWithTxCommits(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = WithTx(context.Background(), mock, func(tx pgx.Tx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestWithTxRollsBackOnFnError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = WithTx(context.Background(), mock, func(tx pgx.Tx) error {
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("WithTx() error = %v, want %v", err, errBoom)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestWithTxPropagatesBeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(errBoom)

	err = WithTx(context.Background(), mock, func(tx pgx.Tx) error {
		t.Error("fn must not be called when Begin fails")
		return nil
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("WithTx() error = %v, want %v", err, errBoom)
	}
}

func TestWithTxPropagatesCommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errBoom)

	err = WithTx(context.Background(), mock, func(tx pgx.Tx) error {
		return nil
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("WithTx() error = %v, want %v", err, errBoom)
	}
}
