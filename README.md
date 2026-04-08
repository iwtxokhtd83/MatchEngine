# MatchEngine

A high-performance order matching engine written in Go.

## Features

- Limit order support (buy/sell)
- Market order support
- Price-time priority matching (FIFO)
- Exact decimal arithmetic via [shopspring/decimal](https://github.com/shopspring/decimal)
- Efficient price-level order book with O(log p) insertion (p = distinct price levels)
- Duplicate order ID detection

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Order Input │────▶│ Match Engine │────▶│   Trades    │
└─────────────┘     └──────┬───────┘     └─────────────┘
                           │
                    ┌──────┴───────┐
                    │  Order Book  │
                    ├──────────────┤
                    │  Bids (Buy)  │
                    │  Asks (Sell) │
                    └──────────────┘
```

## Getting Started

### Prerequisites

- Go 1.21+

### Build

```bash
go build ./...
```

### Test

```bash
go test ./... -v
```

### Run Example

```bash
go run cmd/example/main.go
```

## Project Structure

```
.
├── pkg/
│   ├── engine/       # Core matching engine
│   ├── orderbook/    # Order book implementation
│   └── model/        # Order, Trade, and other models
├── cmd/
│   └── example/      # Example usage
├── go.mod
└── README.md
```

## How It Works

The engine uses a price-time priority algorithm:

1. Buy orders are sorted by price descending, then by time ascending
2. Sell orders are sorted by price ascending, then by time ascending
3. A match occurs when the best bid price >= best ask price
4. Partial fills are supported — remaining quantity stays in the book

## License

MIT License
