package engine

import (
	"context"
	"sync"
)

// DiscardSink ignores events but keeps the asynchronous contract.
type DiscardSink struct{}

func NewDiscardSink() *DiscardSink {
	return &DiscardSink{}
}

func (d *DiscardSink) EnqueueOrder(order *Order) {}

func (d *DiscardSink) EnqueueExecution(execution Execution) {}

func (d *DiscardSink) Shutdown(ctx context.Context) error {
	return nil
}

// AsyncMemorySink captures events in-memory for tests or diagnostics.
type AsyncMemorySink struct {
	orders     []*Order
	executions []Execution
	orderCh    chan *Order
	execCh     chan Execution
	done       chan struct{}
	mu         sync.Mutex
}

func NewAsyncMemorySink(buffer int) *AsyncMemorySink {
	if buffer < 1 {
		buffer = 1
	}
	sink := &AsyncMemorySink{
		orderCh: make(chan *Order, buffer),
		execCh:  make(chan Execution, buffer),
		done:    make(chan struct{}),
	}
	go sink.run()
	return sink
}

func (s *AsyncMemorySink) EnqueueOrder(order *Order) {
	if order == nil {
		return
	}
	s.orderCh <- order.clone()
}

func (s *AsyncMemorySink) EnqueueExecution(execution Execution) {
	s.execCh <- execution
}

func (s *AsyncMemorySink) Shutdown(ctx context.Context) error {
	close(s.orderCh)
	close(s.execCh)
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *AsyncMemorySink) Orders() []*Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	orders := make([]*Order, len(s.orders))
	copy(orders, s.orders)
	return orders
}

func (s *AsyncMemorySink) Executions() []Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	executions := make([]Execution, len(s.executions))
	copy(executions, s.executions)
	return executions
}

func (s *AsyncMemorySink) run() {
	defer close(s.done)
	for s.orderCh != nil || s.execCh != nil {
		select {
		case order, ok := <-s.orderCh:
			if !ok {
				s.orderCh = nil
				continue
			}
			s.mu.Lock()
			s.orders = append(s.orders, order)
			s.mu.Unlock()
		case execution, ok := <-s.execCh:
			if !ok {
				s.execCh = nil
				continue
			}
			s.mu.Lock()
			s.executions = append(s.executions, execution)
			s.mu.Unlock()
		}
	}
}
