package engine

import (
	"context"
	"testing"
)

func TestPriceTimePriority(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()

	sell1, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 2, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 100})
	sell2, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 3, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 100})
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 4, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 101})

	buy, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeLimit, Quantity: 8, Price: 100})
	if len(buy.Executions) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(buy.Executions))
	}
	if buy.Executions[0].MakerOrderID != sell1.Order.ID {
		t.Fatalf("expected first execution maker %d, got %d", sell1.Order.ID, buy.Executions[0].MakerOrderID)
	}
	if buy.Executions[1].MakerOrderID != sell2.Order.ID {
		t.Fatalf("expected second execution maker %d, got %d", sell2.Order.ID, buy.Executions[1].MakerOrderID)
	}
	if buy.Order.Status != OrderStatusFilled {
		t.Fatalf("expected buy order filled, got %s", buy.Order.Status)
	}
	if order, ok := eng.FindOrder(1, sell2.Order.ID); !ok || order.Remaining != 2 {
		t.Fatalf("expected remaining 2 on sell2, got %+v", order)
	}
}

func TestMarketOrderGuard(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 2, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 100})
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 3, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 120})

	buy, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeMarket, Quantity: 10})
	if len(buy.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(buy.Executions))
	}
	if buy.Executions[0].Price != 100 {
		t.Fatalf("expected execution price 100, got %d", buy.Executions[0].Price)
	}
	if buy.Order.Status != OrderStatusPartial {
		t.Fatalf("expected partial status, got %s", buy.Order.Status)
	}
	if buy.Order.Remaining != 0 {
		t.Fatalf("expected remaining cancelled, got %d", buy.Order.Remaining)
	}
}

func TestSelfTradePrevention(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()
	sell, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 100})

	buy, err := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeLimit, Quantity: 5, Price: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buy.Order.Status != OrderStatusOpen {
		t.Fatalf("expected open status, got %s", buy.Order.Status)
	}
	if len(buy.Executions) != 0 {
		t.Fatalf("expected no executions, got %d", len(buy.Executions))
	}
	cancelled, ok := eng.FindOrder(1, sell.Order.ID)
	if !ok || cancelled.Status != OrderStatusCancelled {
		t.Fatalf("expected sell order cancelled, got %+v", cancelled)
	}
}

func TestSelfTradePreventionMarketReduction(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()
	selfQty := int64(5)
	otherQty := int64(5)
	marketQty := int64(8)
	expectedExecQty := marketQty - selfQty
	selfSell, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideSell, Type: OrderTypeLimit, Quantity: selfQty, Price: 100})
	otherSell, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 2, Side: SideSell, Type: OrderTypeLimit, Quantity: otherQty, Price: 100})

	buy, err := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeMarket, Quantity: marketQty})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buy.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(buy.Executions))
	}
	if buy.Executions[0].Quantity != expectedExecQty {
		t.Fatalf("expected execution quantity %d, got %d", expectedExecQty, buy.Executions[0].Quantity)
	}
	if buy.Order.Status != OrderStatusPartial {
		t.Fatalf("expected partial status, got %s", buy.Order.Status)
	}
	if buy.Order.Remaining != 0 {
		t.Fatalf("expected remaining cancelled, got %d", buy.Order.Remaining)
	}
	selfOrder, ok := eng.FindOrder(1, selfSell.Order.ID)
	if !ok || selfOrder.Status != OrderStatusCancelled {
		t.Fatalf("expected self sell order cancelled, got %+v", selfOrder)
	}
	otherOrder, ok := eng.FindOrder(1, otherSell.Order.ID)
	if !ok || otherOrder.Remaining != 2 {
		t.Fatalf("expected remaining 2 on other sell, got %+v", otherOrder)
	}
}

func TestStopOrderTrigger(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()

	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 2, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 112})
	stop, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeStop, Quantity: 5, StopPrice: 110})
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 3, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 110})
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 4, Side: SideBuy, Type: OrderTypeMarket, Quantity: 5})

	order, ok := eng.FindOrder(1, stop.Order.ID)
	if !ok {
		t.Fatalf("stop order not found")
	}
	if order.Status != OrderStatusFilled {
		t.Fatalf("expected stop order filled, got %s", order.Status)
	}
}

func TestFindOrderRoutesByAssetID(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()

	first, _ := eng.SubmitOrder(ctx, &Order{ID: 101, AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeLimit, Quantity: 1, Price: 100})
	second, _ := eng.SubmitOrder(ctx, &Order{ID: 202, AssetID: 2, UserID: 1, Side: SideBuy, Type: OrderTypeLimit, Quantity: 1, Price: 100})

	order, ok := eng.FindOrder(1, first.Order.ID)
	if !ok || order.AssetID != 1 {
		t.Fatalf("expected order in asset 1, got %+v", order)
	}
	order, ok = eng.FindOrder(2, second.Order.ID)
	if !ok || order.AssetID != 2 {
		t.Fatalf("expected order in asset 2, got %+v", order)
	}
	if _, ok := eng.FindOrder(2, first.Order.ID); ok {
		t.Fatalf("expected asset-scoped lookup to reject wrong asset")
	}
}
