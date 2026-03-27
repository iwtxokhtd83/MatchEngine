package engine

import (
	"testing"

	"github.com/matchengine/matchengine/pkg/model"
)

func TestLimitOrderMatching(t *testing.T) {
	e := New()

	// Place a sell limit order
	sell := model.NewLimitOrder("s1", model.Sell, 100.0, 10)
	trades, err := e.SubmitOrder("AAPL", sell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	// Place a buy limit order that crosses the spread
	buy := model.NewLimitOrder("b1", model.Buy, 100.0, 5)
	trades, err = e.SubmitOrder("AAPL", buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 5 {
		t.Errorf("expected trade quantity 5, got %f", trades[0].Quantity)
	}
	if trades[0].Price != 100.0 {
		t.Errorf("expected trade price 100, got %f", trades[0].Price)
	}
}

func TestPartialFill(t *testing.T) {
	e := New()

	sell := model.NewLimitOrder("s1", model.Sell, 50.0, 3)
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, 50.0, 10)
	trades, _ := e.SubmitOrder("BTC", buy)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 3 {
		t.Errorf("expected trade quantity 3, got %f", trades[0].Quantity)
	}

	// Buy order should remain in book with remaining=7
	book := e.GetOrderBook("BTC")
	if len(book.Bids) != 1 {
		t.Fatalf("expected 1 bid remaining, got %d", len(book.Bids))
	}
	if book.Bids[0].Remaining != 7 {
		t.Errorf("expected remaining 7, got %f", book.Bids[0].Remaining)
	}
}

func TestMarketOrder(t *testing.T) {
	e := New()

	// Place limit sells at different prices
	e.SubmitOrder("ETH", model.NewLimitOrder("s1", model.Sell, 100.0, 5))
	e.SubmitOrder("ETH", model.NewLimitOrder("s2", model.Sell, 101.0, 5))

	// Market buy should sweep through price levels
	buy := model.NewMarketOrder("b1", model.Buy, 8)
	trades, err := e.SubmitOrder("ETH", buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if trades[0].Quantity != 5 || trades[0].Price != 100.0 {
		t.Errorf("first trade: expected 5@100, got %f@%f", trades[0].Quantity, trades[0].Price)
	}
	if trades[1].Quantity != 3 || trades[1].Price != 101.0 {
		t.Errorf("second trade: expected 3@101, got %f@%f", trades[1].Quantity, trades[1].Price)
	}
}

func TestNoMatchWhenPricesDontCross(t *testing.T) {
	e := New()

	e.SubmitOrder("GOOG", model.NewLimitOrder("s1", model.Sell, 200.0, 10))
	trades, _ := e.SubmitOrder("GOOG", model.NewLimitOrder("b1", model.Buy, 199.0, 10))

	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	book := e.GetOrderBook("GOOG")
	bids, asks := book.Depth()
	if bids != 1 || asks != 1 {
		t.Errorf("expected depth 1/1, got %d/%d", bids, asks)
	}
}

func TestCancelOrder(t *testing.T) {
	e := New()

	e.SubmitOrder("MSFT", model.NewLimitOrder("s1", model.Sell, 300.0, 10))

	if !e.CancelOrder("MSFT", "s1") {
		t.Error("expected cancel to succeed")
	}
	if e.CancelOrder("MSFT", "s1") {
		t.Error("expected second cancel to fail")
	}

	book := e.GetOrderBook("MSFT")
	_, asks := book.Depth()
	if asks != 0 {
		t.Errorf("expected 0 asks after cancel, got %d", asks)
	}
}

func TestPriceTimePriority(t *testing.T) {
	e := New()

	// Two sells at same price — first one should match first (FIFO)
	e.SubmitOrder("TSLA", model.NewLimitOrder("s1", model.Sell, 100.0, 5))
	e.SubmitOrder("TSLA", model.NewLimitOrder("s2", model.Sell, 100.0, 5))

	buy := model.NewLimitOrder("b1", model.Buy, 100.0, 3)
	trades, _ := e.SubmitOrder("TSLA", buy)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].SellOrderID != "s1" {
		t.Errorf("expected match with s1 (FIFO), got %s", trades[0].SellOrderID)
	}
}

func TestMultipleSymbols(t *testing.T) {
	e := New()

	e.SubmitOrder("AAPL", model.NewLimitOrder("a-s1", model.Sell, 150.0, 10))
	e.SubmitOrder("GOOG", model.NewLimitOrder("g-s1", model.Sell, 2800.0, 5))

	trades, _ := e.SubmitOrder("AAPL", model.NewLimitOrder("a-b1", model.Buy, 150.0, 10))
	if len(trades) != 1 {
		t.Fatalf("expected 1 AAPL trade, got %d", len(trades))
	}

	// GOOG book should be unaffected
	book := e.GetOrderBook("GOOG")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected 1 GOOG ask, got %d", asks)
	}
}

func TestInvalidOrders(t *testing.T) {
	e := New()

	_, err := e.SubmitOrder("X", nil)
	if err == nil {
		t.Error("expected error for nil order")
	}

	_, err = e.SubmitOrder("X", model.NewLimitOrder("bad", model.Buy, -1, 10))
	if err == nil {
		t.Error("expected error for negative price")
	}

	_, err = e.SubmitOrder("X", model.NewLimitOrder("bad", model.Buy, 100, 0))
	if err == nil {
		t.Error("expected error for zero quantity")
	}
}
