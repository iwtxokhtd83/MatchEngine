# Roadmap

This page tracks planned features and areas where contributions are welcome.

## Short Term

### ~~Decimal Precision~~ ✅ Done
Replaced `float64` with [`shopspring/decimal`](https://github.com/shopspring/decimal) for all price and quantity fields. All comparisons use exact decimal arithmetic, eliminating floating-point rounding errors.

### Additional Order Types
- **Immediate-or-Cancel (IOC)**: Fill as much as possible immediately, cancel the rest.
- **Fill-or-Kill (FOK)**: Fill the entire order immediately or cancel it entirely.
- **Good-Till-Cancelled (GTC)**: Order stays in the book until explicitly cancelled or an expiry time is reached.

### Benchmarks
Add Go benchmark tests to measure:
- Orders per second (throughput)
- Time to match (latency)
- Memory usage under load

## Medium Term

### Event System
Publish events for downstream consumers:
- `OrderAccepted` — order added to book
- `OrderMatched` — trade executed
- `OrderCancelled` — order removed
- `OrderBookUpdated` — book state changed

This enables building real-time UIs, logging systems, and analytics pipelines.

### REST / gRPC API
Expose the engine over the network:
- `POST /orders` — submit an order
- `DELETE /orders/{id}` — cancel an order
- `GET /orderbook/{symbol}` — get current book state
- `GET /trades` — get trade history

### Persistence
- Write-ahead log (WAL) for crash recovery
- Trade history storage (SQLite, PostgreSQL, or append-only file)
- Order book snapshots for fast restart

## Long Term

### Optimized Data Structures
Replace sorted slices with:
- Red-black tree for O(log n) insert/remove
- Skip list as an alternative with simpler implementation
- Price-level grouping to reduce per-order overhead

### Per-Symbol Locking
Replace the single engine mutex with per-symbol locks to allow parallel matching across different trading pairs.

### Stop Orders
- **Stop-Market**: Becomes a market order when the stop price is reached.
- **Stop-Limit**: Becomes a limit order when the stop price is reached.

Requires a price trigger mechanism that monitors the last trade price.

### WebSocket Feed
Real-time streaming of:
- Order book updates (Level 2 data)
- Trade feed
- Ticker (last price, 24h volume, etc.)

## Contributing

Contributions are welcome. If you want to work on any of these items:

1. Open an issue to discuss the approach
2. Fork the repository
3. Create a feature branch
4. Submit a pull request

Please include tests for any new functionality.
