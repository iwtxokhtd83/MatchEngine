# Roadmap

This page tracks planned features and areas where contributions are welcome.

## Completed

### ~~Decimal Precision~~ ✅ (v0.1.0)
Replaced `float64` with [`shopspring/decimal`](https://github.com/shopspring/decimal) for all price and quantity fields.

### ~~Duplicate Order ID Detection~~ ✅ (v0.2.0)
Engine maintains an order index and rejects duplicate IDs. IDs are freed after fill or cancel.

### ~~Thread-Safe Order Book Reads~~ ✅ (v0.2.0)
`GetOrderBook()` returns a deep-copy snapshot instead of an internal pointer.

### ~~Bounded Trade Log~~ ✅ (v0.2.0)
Trade log has a configurable max size (default 10,000). Supports `WithTradeHandler` callback for real-time trade processing. Log can be disabled with `WithMaxTradeLog(0)`.

### ~~Symbol Validation~~ ✅ (v0.2.0)
Symbols are normalized (uppercased, trimmed). Empty/whitespace symbols are rejected. Optional `RegisterSymbol()` for strict validation.

## Short Term

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
Publish structured events for downstream consumers:
- `OrderAccepted` — order added to book
- `OrderMatched` — trade executed
- `OrderCancelled` — order removed
- `OrderBookUpdated` — book state changed

### REST / gRPC API
Expose the engine over the network.

### Persistence
- Write-ahead log (WAL) for crash recovery
- Trade history storage
- Order book snapshots for fast restart

## Long Term

### Optimized Data Structures
Replace sorted slices with red-black tree or skip list for O(log n) insert/remove.

### Per-Symbol Locking
Replace the single engine mutex with per-symbol locks.

### Stop Orders
Stop-Market and Stop-Limit orders with price trigger mechanism.

### WebSocket Feed
Real-time streaming of order book updates and trade feed.

## Contributing

Contributions are welcome. If you want to work on any of these items:

1. Open an issue to discuss the approach
2. Fork the repository
3. Create a feature branch
4. Submit a pull request

Please include tests for any new functionality.
