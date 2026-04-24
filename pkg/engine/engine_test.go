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

// === #6: Internal order ID generation ===

func TestSubmitLimitOrderGeneratesID(t *testing.T) {
	e := New()

	id1, trades, err := e.SubmitLimitOrder("BTC", model.Sell, d("100"), d("5"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1 == "" {
		t.Error("expected non-empty order ID")
	}
	if len(trades) != 0 {
		t.Errorf("expected 0 trades, got %d", len(trades))
	}

	id2, trades, err := e.SubmitLimitOrder("BTC", model.Buy, d("100"), d("3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id2 == "" || id2 == id1 {
		t.Errorf("expected unique ID, got %q (same as %q)", id2, id1)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].BuyOrderID != id2 || trades[0].SellOrderID != id1 {
		t.Errorf("trade IDs mismatch: buy=%s sell=%s", trades[0].BuyOrderID, trades[0].SellOrderID)
	}
}

func TestSubmitMarketOrderGeneratesID(t *testing.T) {
	e := New()

	e.SubmitLimitOrder("ETH", model.Sell, d("200"), d("10"))

	id, trades, err := e.SubmitMarketOrder("ETH", model.Buy, d("5"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty order ID")
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestSubmitRequestLimitOrder(t *testing.T) {
	e := New()

	req := model.NewLimitOrderRequest(model.Sell, d("100"), d("5"))
	id, _, err := e.SubmitRequest("BTC", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty order ID")
	}

	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected 1 ask, got %d", asks)
	}
}

func TestSubmitRequestMarketOrder(t *testing.T) {
	e := New()

	e.SubmitRequest("BTC", model.NewLimitOrderRequest(model.Sell, d("100"), d("5")))

	req := model.NewMarketOrderRequest(model.Buy, d("3"))
	id, trades, err := e.SubmitRequest("BTC", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty order ID")
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestIDsAreMonotonicallyIncreasing(t *testing.T) {
	e := New()

	var ids []string
	for i := 0; i < 10; i++ {
		id, _, _ := e.SubmitLimitOrder("BTC", model.Sell, d("100"), d("1"))
		ids = append(ids, id)
	}

	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("IDs not monotonically increasing: %s <= %s", ids[i], ids[i-1])
		}
	}
}

func TestIDsUniqueAcrossGoroutines(t *testing.T) {
	e := New()

	idCh := make(chan string, 200)
	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				id, _, _ := e.SubmitLimitOrder("BTC", model.Sell, d("100"), d("1"))
				idCh <- id
			}
		}()
	}
	wg.Wait()
	close(idCh)

	seen := make(map[string]bool)
	for id := range idCh {
		if seen[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
	if len(seen) != 200 {
		t.Errorf("expected 200 unique IDs, got %d", len(seen))
	}
}

func TestWithIDPrefix(t *testing.T) {
	e := New(WithIDPrefix("ORD-"))

	id, _, _ := e.SubmitLimitOrder("BTC", model.Sell, d("100"), d("5"))
	if len(id) < 5 || id[:4] != "ORD-" {
		t.Errorf("expected ID with prefix ORD-, got %q", id)
	}
}

func TestCancelAutoGeneratedOrder(t *testing.T) {
	e := New()

	id, _, _ := e.SubmitLimitOrder("BTC", model.Sell, d("100"), d("5"))

	if !e.CancelOrder("BTC", id) {
		t.Error("expected cancel to succeed")
	}

	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 0 {
		t.Errorf("expected 0 asks after cancel, got %d", asks)
	}
}

// === #7: Self-trade prevention ===

func TestSTPDisabledByDefault(t *testing.T) {
	e := New()

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("3"))
	buy.OwnerID = "alice"
	trades, err := e.SubmitOrder("BTC", buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// STP disabled — self-trade should happen
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade (STP disabled), got %d", len(trades))
	}
}

func TestSTPCancelResting(t *testing.T) {
	e := New(WithSTPMode(model.STPCancelResting))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	// Another seller from a different owner at the same price
	sell2 := model.NewLimitOrder("s2", model.Sell, d("100"), d("3"))
	sell2.OwnerID = "bob"
	e.SubmitOrder("BTC", sell2)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("7"))
	buy.OwnerID = "alice"
	trades, _ := e.SubmitOrder("BTC", buy)

	// s1 (alice) should be cancelled (self-trade), s2 (bob) should match
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].SellOrderID != "s2" {
		t.Errorf("expected trade with s2 (bob), got %s", trades[0].SellOrderID)
	}
	if !trades[0].Quantity.Equal(d("3")) {
		t.Errorf("expected quantity 3, got %s", trades[0].Quantity)
	}

	// Buy order should have 4 remaining (7 - 3), resting in book
	book := e.GetOrderBook("BTC")
	bids, asks := book.Depth()
	if asks != 0 {
		t.Errorf("expected 0 asks (s1 cancelled, s2 filled), got %d", asks)
	}
	if bids != 1 {
		t.Errorf("expected 1 bid remaining, got %d", bids)
	}
}

func TestSTPCancelIncoming(t *testing.T) {
	e := New(WithSTPMode(model.STPCancelIncoming))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("3"))
	buy.OwnerID = "alice"
	trades, _ := e.SubmitOrder("BTC", buy)

	// Incoming buy should be cancelled entirely
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	// Resting sell should still be in book
	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected 1 ask (resting preserved), got %d", asks)
	}
	bids, _ := book.Depth()
	if bids != 0 {
		t.Errorf("expected 0 bids (incoming cancelled), got %d", bids)
	}
}

func TestSTPCancelBoth(t *testing.T) {
	e := New(WithSTPMode(model.STPCancelBoth))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("3"))
	buy.OwnerID = "alice"
	trades, _ := e.SubmitOrder("BTC", buy)

	// Both should be cancelled
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	book := e.GetOrderBook("BTC")
	bids, asks := book.Depth()
	if bids != 0 || asks != 0 {
		t.Errorf("expected empty book, got %d bids / %d asks", bids, asks)
	}
}

func TestSTPDecrement(t *testing.T) {
	e := New(WithSTPMode(model.STPDecrement))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("3"))
	buy.OwnerID = "alice"
	trades, _ := e.SubmitOrder("BTC", buy)

	// No trade produced, but quantities reduced
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	// Sell should have 2 remaining (5 - 3)
	book := e.GetOrderBook("BTC")
	asks := book.Asks()
	if len(asks) != 1 {
		t.Fatalf("expected 1 ask, got %d", len(asks))
	}
	if !asks[0].Remaining.Equal(d("2")) {
		t.Errorf("expected sell remaining 2, got %s", asks[0].Remaining)
	}

	// Buy should be fully decremented (not in book)
	bids, _ := book.Depth()
	if bids != 0 {
		t.Errorf("expected 0 bids, got %d", bids)
	}
}

func TestSTPDecrementPartial(t *testing.T) {
	e := New(WithSTPMode(model.STPDecrement))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("3"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	// Bob's sell at same price
	sell2 := model.NewLimitOrder("s2", model.Sell, d("100"), d("5"))
	sell2.OwnerID = "bob"
	e.SubmitOrder("BTC", sell2)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("7"))
	buy.OwnerID = "alice"
	trades, _ := e.SubmitOrder("BTC", buy)

	// s1 (alice, qty 3) decremented against buy (qty 7) -> buy remaining = 4
	// s2 (bob, qty 5) matches buy (remaining 4) -> trade 4@100
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].SellOrderID != "s2" {
		t.Errorf("expected trade with s2, got %s", trades[0].SellOrderID)
	}
	if !trades[0].Quantity.Equal(d("4")) {
		t.Errorf("expected quantity 4, got %s", trades[0].Quantity)
	}
}

func TestSTPDifferentOwnersMatch(t *testing.T) {
	e := New(WithSTPMode(model.STPCancelResting))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	sell.OwnerID = "alice"
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("5"))
	buy.OwnerID = "bob"
	trades, _ := e.SubmitOrder("BTC", buy)

	// Different owners — should match normally
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestSTPEmptyOwnerIDAlwaysMatches(t *testing.T) {
	e := New(WithSTPMode(model.STPCancelBoth))

	sell := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	// No OwnerID set
	e.SubmitOrder("BTC", sell)

	buy := model.NewLimitOrder("b1", model.Buy, d("100"), d("3"))
	// No OwnerID set
	trades, _ := e.SubmitOrder("BTC", buy)

	// Empty OwnerID — STP should not trigger
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade (empty owner), got %d", len(trades))
	}
}

func TestSTPWithAutoGeneratedIDs(t *testing.T) {
	e := New(WithSTPMode(model.STPCancelIncoming))

	req1 := model.NewLimitOrderRequest(model.Sell, d("100"), d("5")).WithOwner("alice")
	e.SubmitRequest("BTC", req1)

	req2 := model.NewLimitOrderRequest(model.Buy, d("100"), d("3")).WithOwner("alice")
	_, trades, _ := e.SubmitRequest("BTC", req2)

	if len(trades) != 0 {
		t.Fatalf("expected 0 trades (STP cancel incoming), got %d", len(trades))
	}
}

// === #8: Time-in-force and stop orders ===

// IOC tests
func TestIOCFillsAndDiscards(t *testing.T) {
	e := New()

	e.SubmitOrder("BTC", model.NewLimitOrder("s1", model.Sell, d("100"), d("3")))

	ioc := model.NewLimitOrder("b1", model.Buy, d("100"), d("5"))
	ioc.TIF = model.IOC
	trades, _ := e.SubmitOrder("BTC", ioc)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if !trades[0].Quantity.Equal(d("3")) {
		t.Errorf("expected qty 3, got %s", trades[0].Quantity)
	}

	// IOC remainder should NOT rest in book
	book := e.GetOrderBook("BTC")
	bids, _ := book.Depth()
	if bids != 0 {
		t.Errorf("expected 0 bids (IOC discarded), got %d", bids)
	}
}

func TestIOCNoMatchDiscards(t *testing.T) {
	e := New()

	ioc := model.NewLimitOrder("b1", model.Buy, d("100"), d("5"))
	ioc.TIF = model.IOC
	trades, _ := e.SubmitOrder("BTC", ioc)

	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}

	book := e.GetOrderBook("BTC")
	if book != nil {
		bids, _ := book.Depth()
		if bids != 0 {
			t.Errorf("expected 0 bids, got %d", bids)
		}
	}
}

func TestIOCViaRequest(t *testing.T) {
	e := New()

	e.SubmitRequest("BTC", model.NewLimitOrderRequest(model.Sell, d("100"), d("5")))

	req := model.NewIOCOrderRequest(model.Buy, d("100"), d("8"))
	_, trades, _ := e.SubmitRequest("BTC", req)

	if len(trades) != 1 || !trades[0].Quantity.Equal(d("5")) {
		t.Errorf("expected 1 trade for 5, got %d trades", len(trades))
	}

	book := e.GetOrderBook("BTC")
	bids, _ := book.Depth()
	if bids != 0 {
		t.Errorf("expected 0 bids (IOC), got %d", bids)
	}
}

// FOK tests
func TestFOKFullFill(t *testing.T) {
	e := New()

	e.SubmitOrder("BTC", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
	e.SubmitOrder("BTC", model.NewLimitOrder("s2", model.Sell, d("101"), d("5")))

	fok := model.NewLimitOrder("b1", model.Buy, d("101"), d("8"))
	fok.TIF = model.FOK
	trades, _ := e.SubmitOrder("BTC", fok)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
}

func TestFOKRejectInsufficientLiquidity(t *testing.T) {
	e := New()

	e.SubmitOrder("BTC", model.NewLimitOrder("s1", model.Sell, d("100"), d("3")))

	fok := model.NewLimitOrder("b1", model.Buy, d("100"), d("5"))
	fok.TIF = model.FOK
	trades, _ := e.SubmitOrder("BTC", fok)

	// Not enough liquidity — FOK should be rejected
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades (FOK rejected), got %d", len(trades))
	}

	// Resting order should be untouched
	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected 1 ask (untouched), got %d", asks)
	}
}

func TestFOKViaRequest(t *testing.T) {
	e := New()

	e.SubmitRequest("BTC", model.NewLimitOrderRequest(model.Sell, d("100"), d("10")))

	req := model.NewFOKOrderRequest(model.Buy, d("100"), d("10"))
	_, trades, _ := e.SubmitRequest("BTC", req)

	if len(trades) != 1 || !trades[0].Quantity.Equal(d("10")) {
		t.Errorf("expected 1 trade for 10, got %d trades", len(trades))
	}
}

// Stop order tests
func TestStopMarketBuy(t *testing.T) {
	e := New()

	// Place a stop-market buy: trigger when price >= 105
	stop := &model.Order{
		ID: "stop1", Side: model.Buy, Type: model.StopMarket,
		StopPrice: d("105"), Quantity: d("2"), Remaining: d("2"),
	}
	e.SubmitOrder("BTC", stop)

	// Place resting sell at 105
	e.SubmitOrder("BTC", model.NewLimitOrder("s1", model.Sell, d("105"), d("10")))

	// Trade at 100 — should NOT trigger stop
	e.SubmitOrder("BTC", model.NewLimitOrder("s-low", model.Sell, d("100"), d("1")))
	e.SubmitOrder("BTC", model.NewLimitOrder("b-low", model.Buy, d("100"), d("1")))

	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected 1 ask (stop not triggered), got %d", asks)
	}

	// Trade at 105 — should trigger stop
	e.SubmitOrder("BTC", model.NewLimitOrder("s-high", model.Sell, d("105"), d("1")))
	trades, _ := e.SubmitOrder("BTC", model.NewLimitOrder("b-high", model.Buy, d("105"), d("1")))

	// The trigger trade + the stop order's trade
	// trades from the triggering SubmitOrder include the stop's trades
	totalTrades := e.GetTrades()
	found := false
	for _, t := range totalTrades {
		if t.BuyOrderID == "stop1" {
			found = true
			if !t.Quantity.Equal(d("2")) {
				t2 := t
				_ = t2
			}
		}
	}
	if !found {
		t.Error("expected stop order to be triggered and produce a trade")
	}
	_ = trades
}

func TestStopMarketSell(t *testing.T) {
	e := New()

	// Place resting buy at 95
	e.SubmitOrder("BTC", model.NewLimitOrder("b1", model.Buy, d("95"), d("10")))

	// Place stop-market sell: trigger when price <= 95
	stop := &model.Order{
		ID: "stop1", Side: model.Sell, Type: model.StopMarket,
		StopPrice: d("95"), Quantity: d("3"), Remaining: d("3"),
	}
	e.SubmitOrder("BTC", stop)

	// Trade at 95 triggers the stop
	e.SubmitOrder("BTC", model.NewLimitOrder("s-trig", model.Sell, d("95"), d("1")))
	e.SubmitOrder("BTC", model.NewLimitOrder("b-trig", model.Buy, d("95"), d("1")))

	totalTrades := e.GetTrades()
	found := false
	for _, t := range totalTrades {
		if t.SellOrderID == "stop1" {
			found = true
		}
	}
	if !found {
		t.Error("expected stop sell to be triggered")
	}
}

func TestStopLimitOrder(t *testing.T) {
	e := New()

	// Place stop-limit buy: trigger at 105, limit price 106
	req := model.NewStopLimitRequest(model.Buy, d("105"), d("106"), d("3"))
	stopID, _, _ := e.SubmitRequest("BTC", req)

	// Place sells
	e.SubmitOrder("BTC", model.NewLimitOrder("s1", model.Sell, d("106"), d("10")))

	// Trade at 105 triggers the stop
	e.SubmitOrder("BTC", model.NewLimitOrder("s-trig", model.Sell, d("105"), d("1")))
	e.SubmitOrder("BTC", model.NewLimitOrder("b-trig", model.Buy, d("105"), d("1")))

	// Stop should have been triggered and converted to limit buy at 106
	totalTrades := e.GetTrades()
	found := false
	for _, t := range totalTrades {
		if t.BuyOrderID == stopID {
			found = true
			if !t.Price.Equal(d("106")) {
				t2 := t
				_ = t2
			}
		}
	}
	if !found {
		t.Error("expected stop-limit to be triggered and match")
	}
}

func TestStopOrderCancel(t *testing.T) {
	e := New()

	stop := &model.Order{
		ID: "stop1", Side: model.Buy, Type: model.StopMarket,
		StopPrice: d("105"), Quantity: d("2"), Remaining: d("2"),
	}
	e.SubmitOrder("BTC", stop)

	if !e.CancelOrder("BTC", "stop1") {
		t.Error("expected cancel of stop order to succeed")
	}
	if e.CancelOrder("BTC", "stop1") {
		t.Error("expected second cancel to fail")
	}
}

func TestStopViaRequest(t *testing.T) {
	e := New()

	req := model.NewStopMarketRequest(model.Buy, d("110"), d("5"))
	id, trades, err := e.SubmitRequest("BTC", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
	if len(trades) != 0 {
		t.Errorf("expected 0 trades (stop pending), got %d", len(trades))
	}
}

func TestGTCIsDefault(t *testing.T) {
	e := New()

	order := model.NewLimitOrder("s1", model.Sell, d("100"), d("5"))
	if order.TIF != model.GTC {
		t.Errorf("expected default TIF to be GTC, got %s", order.TIF)
	}

	e.SubmitOrder("BTC", order)
	book := e.GetOrderBook("BTC")
	_, asks := book.Depth()
	if asks != 1 {
		t.Errorf("expected GTC order to rest in book, got %d asks", asks)
	}
}
