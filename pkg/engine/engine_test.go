package engine

import (
	"fmt"
	"sync"
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
	if len(book.Bids()) != 1 {
		t.Fatalf("expected 1 bid remaining, got %d", len(book.Bids()))
	}
	if !book.Bids()[0].Remaining.Equal(d("7")) {
		t.Errorf("expected remaining 7, got %s", book.Bids()[0].Remaining)
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

	_, err = e.SubmitOrder("X", model.NewLimitOrder("bad2", model.Buy, d("100"), d("0")))
	if err == nil {
		t.Error("expected error for zero quantity")
	}
}

func TestDecimalPrecision(t *testing.T) {
	e := New()

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
	if !buy.IsFilled() {
		t.Errorf("expected buy order to be fully filled, remaining: %s", buy.Remaining)
	}

	book := e.GetOrderBook("PREC")
	bids, asks := book.Depth()
	if bids != 0 || asks != 0 {
		t.Errorf("expected empty book, got %d bids / %d asks", bids, asks)
	}
}

// === Bug fix tests ===

// #2: Duplicate order ID detection
func TestDuplicateOrderID(t *testing.T) {
	e := New()

	_, err := e.SubmitOrder("BTC", model.NewLimitOrder("dup", model.Sell, d("100"), d("5")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = e.SubmitOrder("BTC", model.NewLimitOrder("dup", model.Sell, d("200"), d("10")))
	if err == nil {
		t.Error("expected error for duplicate order ID")
	}

	// Only one order should exist
	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected 1 ask, got %d", asks)
	}
}

func TestDuplicateOrderIDAfterFill(t *testing.T) {
	e := New()

	// Place and fully fill an order
	e.SubmitOrder("BTC", model.NewLimitOrder("reuse", model.Sell, d("100"), d("5")))
	e.SubmitOrder("BTC", model.NewLimitOrder("buyer", model.Buy, d("100"), d("5")))

	// The filled order ID should be available for reuse
	_, err := e.SubmitOrder("BTC", model.NewLimitOrder("reuse", model.Sell, d("100"), d("3")))
	if err != nil {
		t.Errorf("expected reuse of filled order ID to succeed, got: %v", err)
	}
}

func TestDuplicateOrderIDAfterCancel(t *testing.T) {
	e := New()

	e.SubmitOrder("BTC", model.NewLimitOrder("cancel-me", model.Sell, d("100"), d("5")))
	e.CancelOrder("BTC", "cancel-me")

	// Cancelled order ID should be available for reuse
	_, err := e.SubmitOrder("BTC", model.NewLimitOrder("cancel-me", model.Sell, d("200"), d("3")))
	if err != nil {
		t.Errorf("expected reuse of cancelled order ID to succeed, got: %v", err)
	}
}

// #3: GetOrderBook returns snapshot (thread safety)
func TestGetOrderBookReturnsSnapshot(t *testing.T) {
	e := New()

	e.SubmitOrder("SNAP", model.NewLimitOrder("s1", model.Sell, d("100"), d("10")))

	// Get a snapshot
	snap := e.GetOrderBook("SNAP")
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}

	// Modify the snapshot — should NOT affect the engine's internal book
	snap.Asks()[0].Remaining = d("999")

	// Get another snapshot — should still show original value
	snap2 := e.GetOrderBook("SNAP")
	if snap2.Asks()[0].Remaining.Equal(d("999")) {
		t.Error("snapshot mutation affected internal order book — not a true deep copy")
	}
	if !snap2.Asks()[0].Remaining.Equal(d("10")) {
		t.Errorf("expected remaining 10, got %s", snap2.Asks()[0].Remaining)
	}
}

func TestGetOrderBookConcurrentAccess(t *testing.T) {
	e := New()

	e.SubmitOrder("CONC", model.NewLimitOrder("s1", model.Sell, d("100"), d("10")))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			book := e.GetOrderBook("CONC")
			if book == nil {
				t.Error("expected non-nil book")
			}
			_ = book.Depth()
			_ = book.Spread()
		}()
	}
	wg.Wait()
}

// #5: Trade log bounded
func TestTradeLogBounded(t *testing.T) {
	e := New(WithMaxTradeLog(10))

	// Generate 20 trades
	for i := 0; i < 20; i++ {
		sid := fmt.Sprintf("s%d", i)
		bid := fmt.Sprintf("b%d", i)
		e.SubmitOrder("BND", model.NewLimitOrder(sid, model.Sell, d("100"), d("1")))
		e.SubmitOrder("BND", model.NewLimitOrder(bid, model.Buy, d("100"), d("1")))
	}

	trades := e.GetTrades()
	if len(trades) > 10 {
		t.Errorf("expected at most 10 trades in log, got %d", len(trades))
	}
}

func TestTradeLogDisabled(t *testing.T) {
	e := New(WithMaxTradeLog(0))

	e.SubmitOrder("DIS", model.NewLimitOrder("s1", model.Sell, d("100"), d("1")))
	e.SubmitOrder("DIS", model.NewLimitOrder("b1", model.Buy, d("100"), d("1")))

	trades := e.GetTrades()
	if len(trades) != 0 {
		t.Errorf("expected 0 trades when log disabled, got %d", len(trades))
	}
}

func TestTradeHandler(t *testing.T) {
	var received []model.Trade
	handler := func(symbol string, trade model.Trade) {
		received = append(received, trade)
	}

	e := New(WithTradeHandler(handler))

	e.SubmitOrder("CB", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	e.SubmitOrder("CB", model.NewLimitOrder("b1", model.Buy, d("100"), d("3")))

	if len(received) != 1 {
		t.Fatalf("expected 1 trade via handler, got %d", len(received))
	}
	if !received[0].Quantity.Equal(d("3")) {
		t.Errorf("expected quantity 3, got %s", received[0].Quantity)
	}
}

// #10: Symbol validation
func TestEmptySymbolRejected(t *testing.T) {
	e := New()

	_, err := e.SubmitOrder("", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	if err == nil {
		t.Error("expected error for empty symbol")
	}

	_, err = e.SubmitOrder("   ", model.NewLimitOrder("s2", model.Sell, d("100"), d("5")))
	if err == nil {
		t.Error("expected error for whitespace-only symbol")
	}
}

func TestSymbolNormalization(t *testing.T) {
	e := New()

	// "btc/usd" and "BTC/USD" should be the same book
	e.SubmitOrder("btc/usd", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	trades, _ := e.SubmitOrder("BTC/USD", model.NewLimitOrder("b1", model.Buy, d("100"), d("3")))

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade (normalized symbols), got %d", len(trades))
	}
}

func TestRegisteredSymbolsOnly(t *testing.T) {
	e := New()
	e.RegisterSymbol("BTC/USD")
	e.RegisterSymbol("ETH/USD")

	_, err := e.SubmitOrder("BTC/USD", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	if err != nil {
		t.Errorf("expected registered symbol to be accepted, got: %v", err)
	}

	_, err = e.SubmitOrder("DOGE/USD", model.NewLimitOrder("s2", model.Sell, d("1"), d("100")))
	if err == nil {
		t.Error("expected error for unregistered symbol")
	}
}
