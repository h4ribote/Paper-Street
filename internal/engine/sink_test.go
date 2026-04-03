package engine

import (
	"context"
	"errors"
	"testing"
)

func TestDiscardSinkNoop(t *testing.T) {
	sink := NewDiscardSink()
	sink.EnqueueOrder(&Order{ID: 1, AssetID: 10})
	sink.EnqueueExecution(Execution{AssetID: 10, TakerOrderID: 1, MakerOrderID: 2})

	if err := sink.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown should succeed, got error: %v", err)
	}
}

func TestAsyncMemorySinkStoresClonedOrderAndExecution(t *testing.T) {
	sink := NewAsyncMemorySink(2)
	order := &Order{ID: 10, AssetID: 1, UserID: 99, Remaining: 5, Status: OrderStatusOpen}
	exec := Execution{AssetID: 1, TakerOrderID: 10, MakerOrderID: 11, Quantity: 3}

	sink.EnqueueOrder(order)
	sink.EnqueueExecution(exec)

	if err := sink.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown should succeed, got error: %v", err)
	}

	orders := sink.Orders()
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0] == order {
		t.Fatalf("expected stored order clone, got same pointer")
	}
	if orders[0].ID != order.ID || orders[0].AssetID != order.AssetID {
		t.Fatalf("unexpected stored order: %+v", orders[0])
	}

	order.Status = OrderStatusCancelled
	if sink.Orders()[0].Status != OrderStatusOpen {
		t.Fatalf("stored order should not change when original mutates")
	}

	executions := sink.Executions()
	if len(executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(executions))
	}
	if executions[0].TakerOrderID != exec.TakerOrderID || executions[0].Quantity != exec.Quantity {
		t.Fatalf("unexpected execution: %+v", executions[0])
	}
}

func TestAsyncMemorySinkIgnoresNilOrder(t *testing.T) {
	sink := NewAsyncMemorySink(1)
	sink.EnqueueOrder(nil)
	if err := sink.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown should succeed, got error: %v", err)
	}
	if len(sink.Orders()) != 0 {
		t.Fatalf("nil order should be ignored")
	}
}

func TestAsyncMemorySinkShutdownTimeout(t *testing.T) {
	sink := NewAsyncMemorySink(1)
	sink.mu.Lock()
	sink.execCh <- Execution{AssetID: 1}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()

	err := sink.Shutdown(ctx)
	sink.mu.Unlock()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
