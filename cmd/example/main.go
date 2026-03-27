package main

import (
	"fmt"

	"github.com/matchengine/matchengine/pkg/engine"
	"github.com/matchengine/matchengine/pkg/model"
)

func main() {
	e := engine.New()
	symbol := "BTC/USD"

	fmt.Println("=== Order Matching Engine Demo ===")
	fmt.Println()

	// Place some sell limit orders
	fmt.Println("Placing sell orders...")
	e.SubmitOrder(symbol, model.NewLimitOrder("s1", model.Sell, 50000.0, 1.0))
	e.SubmitOrder(symbol, model.NewLimitOrder("s2", model.Sell, 50100.0, 2.0))
	e.SubmitOrder(symbol, model.NewLimitOrder("s3", model.Sell, 50200.0, 1.5))
	fmt.Println("  SELL 1.0 @ 50000")
	fmt.Println("  SELL 2.0 @ 50100")
	fmt.Println("  SELL 1.5 @ 50200")
	fmt.Println()

	// Place a buy limit order that matches
	fmt.Println("Placing buy limit order: BUY 0.5 @ 50000...")
	trades, _ := e.SubmitOrder(symbol, model.NewLimitOrder("b1", model.Buy, 50000.0, 0.5))
	printTrades(trades)

	// Place a market buy order that sweeps multiple levels
	fmt.Println("Placing market buy order: BUY 2.0 (market)...")
	trades, _ = e.SubmitOrder(symbol, model.NewMarketOrder("b2", model.Buy, 2.0))
	printTrades(trades)

	// Show remaining order book
	fmt.Println("--- Order Book ---")
	book := e.GetOrderBook(symbol)
	if book != nil {
		fmt.Println("Bids:")
		for _, b := range book.Bids {
			fmt.Printf("  %s: %.4f @ %.2f\n", b.ID, b.Remaining, b.Price)
		}
		fmt.Println("Asks:")
		for _, a := range book.Asks {
			fmt.Printf("  %s: %.4f @ %.2f\n", a.ID, a.Remaining, a.Price)
		}
	}

	fmt.Printf("\nTotal trades executed: %d\n", len(e.GetTrades()))
}

func printTrades(trades []model.Trade) {
	if len(trades) == 0 {
		fmt.Println("  No trades executed.")
	}
	for _, t := range trades {
		fmt.Printf("  TRADE: %s <-> %s | %.4f @ %.2f\n",
			t.BuyOrderID, t.SellOrderID, t.Quantity, t.Price)
	}
	fmt.Println()
}
