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

func TestUpsertNewsFeed(t *testing.T) {
	queries, mock, cleanup := newMockQueries(t)
	defer cleanup()

	record := NewsRecord{
		ID:             55,
		Headline:       "Arcadia central bank raises rates by 25bps.",
		Body:           "Policy tightening aims to curb inflation.",
		PublishedAt:    time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).UnixMilli(),
		Source:         "Paper Street Wire",
		SentimentScore: -25,
		AssetID:        101,
		Category:       "CENTRAL_BANK",
		Impact:         "NEGATIVE",
		ImpactScope:    `["ARC","ALL_STOCKS"]`,
	}

	mock.ExpectExec("INSERT INTO news_feed").
		WithArgs(record.ID, record.Headline, record.Body, record.PublishedAt, record.Source, record.SentimentScore, record.AssetID, record.Category, record.Impact, record.ImpactScope).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := queries.UpsertNewsFeed(context.Background(), record); err != nil {
		t.Fatalf("UpsertNewsFeed error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListNewsFeed(t *testing.T) {
	queries, mock, cleanup := newMockQueries(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"news_id", "headline", "body", "published_at", "source", "sentiment_score", "related_asset_id", "category", "impact", "impact_scope"}).
		AddRow(10, "Tech Bubble Burst", "Arcadia audits reveal irregularities.", int64(123456), "Paper Street Wire", int64(-80), int64(101), "MARKET", "NEGATIVE", `["OMNI"]`)

	mock.ExpectQuery("SELECT news_id, headline, body, published_at, source, sentiment_score, related_asset_id, category, impact, impact_scope").
		WillReturnRows(rows)

	items, err := queries.ListNewsFeed(context.Background())
	if err != nil {
		t.Fatalf("ListNewsFeed error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 news item, got %d", len(items))
	}
	if items[0].Headline != "Tech Bubble Burst" {
		t.Fatalf("unexpected headline: %s", items[0].Headline)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
