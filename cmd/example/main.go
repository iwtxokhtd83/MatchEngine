package main

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/engine"
	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

func d(val string) decimal.Decimal {
	v, _ := decimal.NewFromString(val)
	return v
}

func main() {
	e := engine.New()
	symbol := "BTC/USD"

	fmt.Println("=== Order Matching Engine Demo ===")
	fmt.Println()

	// Place some sell limit orders
	fmt.Println("Placing sell orders...")
	e.SubmitOrder(symbol, model.NewLimitOrder("s1", model.Sell, d("50000"), d("1")))
	e.SubmitOrder(symbol, model.NewLimitOrder("s2", model.Sell, d("50100"), d("2")))
	e.SubmitOrder(symbol, model.NewLimitOrder("s3", model.Sell, d("50200"), d("1.5")))
	fmt.Println("  SELL 1.0 @ 50000")
	fmt.Println("  SELL 2.0 @ 50100")
	fmt.Println("  SELL 1.5 @ 50200")
	fmt.Println()

	// Place a buy limit order that matches
	fmt.Println("Placing buy limit order: BUY 0.5 @ 50000...")
	trades, _ := e.SubmitOrder(symbol, model.NewLimitOrder("b1", model.Buy, d("50000"), d("0.5")))
	printTrades(trades)

	// Place a market buy order that sweeps multiple levels
	fmt.Println("Placing market buy order: BUY 2.0 (market)...")
	trades, _ = e.SubmitOrder(symbol, model.NewMarketOrder("b2", model.Buy, d("2")))
	printTrades(trades)

	// Show remaining order book
	fmt.Println("--- Order Book ---")
	book := e.GetOrderBook(symbol)
	if book != nil {
		fmt.Println("Bids:")
		for _, b := range book.Bids() {
			fmt.Printf("  %s: %s @ %s\n", b.ID, b.Remaining.StringFixed(4), b.Price.StringFixed(2))
		}
		fmt.Println("Asks:")
		for _, a := range book.Asks() {
			fmt.Printf("  %s: %s @ %s\n", a.ID, a.Remaining.StringFixed(4), a.Price.StringFixed(2))
		}
	}

	fmt.Printf("\nTotal trades executed: %d\n", len(e.GetTrades()))
}

func printTrades(trades []model.Trade) {
	if len(trades) == 0 {
		fmt.Println("  No trades executed.")
	}
	for _, t := range trades {
		fmt.Printf("  TRADE: %s <-> %s | %s @ %s\n",
			t.BuyOrderID, t.SellOrderID, t.Quantity.StringFixed(4), t.Price.StringFixed(2))
	}
	fmt.Println()
}
