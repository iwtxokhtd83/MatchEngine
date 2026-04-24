# MatchEngine

A high-performance order matching engine written in Go.

## Features

- Limit and market order support
- Time-in-force: GTC (default), IOC (Immediate-or-Cancel), FOK (Fill-or-Kill)
- Stop orders: Stop-Market and Stop-Limit with automatic trigger on last trade price
- Price-time priority matching (FIFO)
- Exact decimal arithmetic via [shopspring/decimal](https://github.com/shopspring/decimal)
- Efficient price-level order book with O(log p) insertion (p = distinct price levels)
- Duplicate order ID detection
- Internal order ID generation (atomic counter, monotonically increasing)
- Self-trade prevention (4 modes: CancelResting, CancelIncoming, CancelBoth, Decrement)
- Thread-safe design with snapshot-based order book reads
- Bounded trade log with configurable size and trade callback
- Symbol validation and normalization

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
