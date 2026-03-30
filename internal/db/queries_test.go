package db

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func newMockQueries(t *testing.T) (*Queries, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	conn := &Connection{DB: db}
	return NewQueries(conn), mock, func() {
		_ = db.Close()
	}
}

func TestUpsertOrder(t *testing.T) {
	queries, mock, cleanup := newMockQueries(t)
	defer cleanup()

	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Minute)
	order := &engine.Order{
		ID:        42,
		UserID:    9,
		AssetID:   101,
		Side:      engine.SideBuy,
		Type:      engine.OrderTypeLimit,
		Quantity:  10,
		Remaining: 4,
		Price:     1200,
		Status:    engine.OrderStatusPartial,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(order.ID, order.UserID, order.AssetID, order.Side, order.Type, order.Quantity, order.Price, order.StopPrice, int64(6), order.Status, createdAt.UnixMilli(), updatedAt.UnixMilli()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := queries.UpsertOrder(context.Background(), order); err != nil {
		t.Fatalf("UpsertOrder error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestInsertExecutionTakerBuy(t *testing.T) {
	queries, mock, cleanup := newMockQueries(t)
	defer cleanup()

	executedAt := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	exec := engine.Execution{
		ID:            77,
		AssetID:       55,
		Price:         8900,
		Quantity:      3,
		TakerOrderID:  500,
		MakerOrderID:  501,
		OccurredAtUTC: executedAt,
	}

	mock.ExpectExec("INSERT INTO executions").
		WithArgs(exec.ID, exec.TakerOrderID, exec.MakerOrderID, exec.AssetID, exec.Price, exec.Quantity, executedAt.UnixMilli(), true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := queries.InsertExecution(context.Background(), exec, engine.SideBuy); err != nil {
		t.Fatalf("InsertExecution error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
