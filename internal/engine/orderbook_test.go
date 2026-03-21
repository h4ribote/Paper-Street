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
	if order, ok := eng.FindOrder(sell2.Order.ID); !ok || order.Remaining != 2 {
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
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 100})

	buy, err := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeLimit, Quantity: 5, Price: 100})
	if err == nil {
		t.Fatalf("expected rejection error")
	}
	if buy.Order.Status != OrderStatusRejected {
		t.Fatalf("expected rejected status, got %s", buy.Order.Status)
	}
}

func TestStopOrderTrigger(t *testing.T) {
	eng := NewEngine(NewDiscardSink())
	ctx := context.Background()

	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 2, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 112})
	stop, _ := eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 1, Side: SideBuy, Type: OrderTypeStop, Quantity: 5, StopPrice: 110})
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 3, Side: SideSell, Type: OrderTypeLimit, Quantity: 5, Price: 110})
	_, _ = eng.SubmitOrder(ctx, &Order{AssetID: 1, UserID: 4, Side: SideBuy, Type: OrderTypeMarket, Quantity: 5})

	order, ok := eng.FindOrder(stop.Order.ID)
	if !ok {
		t.Fatalf("stop order not found")
	}
	if order.Status != OrderStatusFilled {
		t.Fatalf("expected stop order filled, got %s", order.Status)
	}
}
