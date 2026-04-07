# Architecture Overview

## Package Structure

```
.
├── pkg/
│   ├── model/        # Pure data types (Order, Trade)
│   ├── orderbook/    # Order book with price-time priority sorting
│   └── engine/       # Core matching logic and coordination
├── cmd/
│   └── example/      # Runnable demo program
├── go.mod
├── LICENSE
└── README.md
```

## Dependency Graph

```
model ← orderbook ← engine ← cmd/example
```

Each layer depends only on the layer to its left. This makes packages independently testable and replaceable.

## Component Responsibilities

### model

Pure data definitions. Depends only on `shopspring/decimal`. Contains:

- `Order` — represents a buy or sell order with ID, side, type, price, quantity, remaining amount, and timestamp
- `Trade` — represents an executed match between two orders
- `Side` — enum for Buy/Sell
- `OrderType` — enum for Limit/Market

### orderbook

Manages the sorted storage of orders. Responsibilities:

- Maintain bids (buy orders) sorted by price descending, then time ascending
- Maintain asks (sell orders) sorted by price ascending, then time ascending
- Provide O(1) access to best bid and best ask
- Insert, remove, and clean up filled orders

### engine

The orchestration layer. Responsibilities:

- Accept incoming orders and validate them
- Route orders to the correct symbol's order book
- Execute the matching algorithm
- Record executed trades
- Handle order cancellation
- Provide thread-safe access via mutex

## Data Flow

```
                    SubmitOrder("BTC/USD", order)
                              │
                              ▼
                    ┌─────────────────┐
                    │   Validation    │
                    │  (nil? price?)  │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Get/Create     │
                    │  Order Book     │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Match Loop     │──── trades[]
                    │  (price-time)   │
                    └────────┬────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
              order filled?     order unfilled?
                    │           (limit only)
                    ▼                 ▼
                  done         Add to order book
```

## Thread Safety Model

A single `sync.Mutex` protects all shared state within an `Engine` instance. Every public method acquires the lock before accessing order books or the trade log.

This is a deliberate simplicity trade-off. For higher throughput, the lock can be replaced with per-symbol mutexes or a sharded architecture without changing the matching logic.
