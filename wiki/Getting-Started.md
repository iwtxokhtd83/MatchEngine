# Getting Started

## Prerequisites

- Go 1.21 or later

## Clone the Repository

```bash
git clone https://github.com/iwtxokhtd83/MatchEngine.git
cd MatchEngine
```

## Build

```bash
go build ./...
```

## Run the Example

```bash
go run cmd/example/main.go
```

Expected output:

```
=== Order Matching Engine Demo ===

Placing sell orders...
  SELL 1.0 @ 50000
  SELL 2.0 @ 50100
  SELL 1.5 @ 50200

Placing buy limit order: BUY 0.5 @ 50000...
  TRADE: b1 <-> s1 | 0.5000 @ 50000.00

Placing market buy order: BUY 2.0 (market)...
  TRADE: b2 <-> s1 | 0.5000 @ 50000.00
  TRADE: b2 <-> s2 | 1.5000 @ 50100.00

--- Order Book ---
Bids:
Asks:
  s2: 0.5000 @ 50100.00
  s3: 1.5000 @ 50200.00

Total trades executed: 3
```

## Run Tests

```bash
go test ./... -v
```

## Use as a Library

Add MatchEngine to your Go project:

```bash
go get github.com/iwtxokhtd83/MatchEngine
```

Basic usage:

```go
package main

import (
    "fmt"
    "github.com/shopspring/decimal"
    "github.com/iwtxokhtd83/MatchEngine/pkg/engine"
    "github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

func main() {
    e := engine.New()

    price := decimal.NewFromInt(50000)
    e.SubmitOrder("BTC/USD", model.NewLimitOrder("s1", model.Sell, price, decimal.NewFromInt(1)))

    trades, _ := e.SubmitOrder("BTC/USD", model.NewLimitOrder("b1", model.Buy, price, decimal.RequireFromString("0.5")))

    for _, t := range trades {
        fmt.Printf("Trade: %s <-> %s | %s @ %s\n",
            t.BuyOrderID, t.SellOrderID, t.Quantity.StringFixed(4), t.Price.StringFixed(2))
    }
}
```
