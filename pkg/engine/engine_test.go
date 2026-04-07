package engine

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

func d(val string) decimal.Decimal {
	v, _ := decimal.NewFromString(val)
	return v
}

func TestLimitOrderMatching(t *testing.T) {
	e := New()

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("10"))
	trades, err := e.SubmitOrder("AAPL", sell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("5"))
	trades, err = e.SubmitOrder("AAPL", buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if !trades[0].Quantity.Equal(d("5")) {
		t.Errorf("expected trade quantity 5, got %s", trades[0].Quantity)
	}
	if !trades[0].Price.Equal(d("100")) {
		t.Errorf("expected trade price 100, got %s", trades[0].Price)
	}
}

func TestPartialFill(t *testing.T) {
	e := New()

	sell := model.NewLimitOrder("s1", model.Sell, d("50"), d("3"))
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("50"), d("10"))
	trades, _ := e.SubmitOrder("BTC", buy)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if !trades[0].Quantity.Equal(d("3")) {
		t.Errorf("expected trade quantity 3, got %s", trades[0].Quantity)
	}

	book := e.GetOrderBook("BTC")
	if len(book.Bids) != 1 {
		t.Fatalf("expected 1 bid remaining, got %d", len(book.Bids))
	}
	if !book.Bids[0].Remaining.Equal(d("7")) {
		t.Errorf("expected remaining 7, got %s", book.Bids[0].Remaining)
	}
}

func TestMarketOrder(t *testing.T) {
	e := New()

	e.SubmitOrder("ETH", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	e.SubmitOrder("ETH", model.NewLimitOrder("s2", model.Sell, d("101"), d("5")))

	buy := model.NewMarketOrder("b1", model.Buy, d("8"))
	trades, err := e.SubmitOrder("ETH", buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if !trades[0].Quantity.Equal(d("5")) || !trades[0].Price.Equal(d("100")) {
		t.Errorf("first trade: expected 5@100, got %s@%s", trades[0].Quantity, trades[0].Price)
	}
	if !trades[1].Quantity.Equal(d("3")) || !trades[1].Price.Equal(d("101")) {
		t.Errorf("second trade: expected 3@101, got %s@%s", trades[1].Quantity, trades[1].Price)
	}
}

func TestNoMatchWhenPricesDontCross(t *testing.T) {
	e := New()

	e.SubmitOrder("GOOG", model.NewLimitOrder("s1", model.Sell, d("200"), d("10")))
	trades, _ := e.SubmitOrder("GOOG", model.NewLimitOrder("b1", model.Buy, d("199"), d("10")))

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

	e.SubmitOrder("MSFT", model.NewLimitOrder("s1", model.Sell, d("300"), d("10")))

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

	e.SubmitOrder("TSLA", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	e.SubmitOrder("TSLA", model.NewLimitOrder("s2", model.Sell, d("100"), d("5")))

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("3"))
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

	e.SubmitOrder("AAPL", model.NewLimitOrder("a-s1", model.Sell, d("150"), d("10")))
	e.SubmitOrder("GOOG", model.NewLimitOrder("g-s1", model.Sell, d("2800"), d("5")))

	trades, _ := e.SubmitOrder("AAPL", model.NewLimitOrder("a-b1", model.Buy, d("150"), d("10")))
	if len(trades) != 1 {
		t.Fatalf("expected 1 AAPL trade, got %d", len(trades))
	}

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

	_, err = e.SubmitOrder("X", model.NewLimitOrder("bad", model.Buy, d("-1"), d("10")))
	if err == nil {
		t.Error("expected error for negative price")
	}

	_, err = e.SubmitOrder("X", model.NewLimitOrder("bad", model.Buy, d("100"), d("0")))
	if err == nil {
		t.Error("expected error for zero quantity")
	}
}

// TestDecimalPrecision verifies that decimal arithmetic avoids float64 rounding errors.
func TestDecimalPrecision(t *testing.T) {
	e := New()

	// 0.1 + 0.2 != 0.3 in float64, but should be exact with decimal
	e.SubmitOrder("PREC", model.NewLimitOrder("s1", model.Sell, d("0.3"), d("0.1")))
	e.SubmitOrder("PREC", model.NewLimitOrder("s2", model.Sell, d("0.3"), d("0.2")))

	buy := model.NewLimitOrder("b1", model.Buy, d("0.3"), d("0.3"))
	trades, err := e.SubmitOrder("PREC", buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	totalQty := trades[0].Quantity.Add(trades[1].Quantity)
	if !totalQty.Equal(d("0.3")) {
		t.Errorf("expected total quantity 0.3, got %s", totalQty)
	}

	// The buy order should be exactly filled — no leftover dust
	if !buy.IsFilled() {
		t.Errorf("expected buy order to be fully filled, remaining: %s", buy.Remaining)
	}

	book := e.GetOrderBook("PREC")
	bids, asks := book.Depth()
	if bids != 0 {
		t.Errorf("expected 0 bids, got %d", bids)
	}
	if asks != 0 {
		t.Errorf("expected 0 asks, got %d", asks)
	}
}
