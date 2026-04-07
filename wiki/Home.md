# MatchEngine Wiki

Welcome to the MatchEngine wiki — an open-source order matching engine written in Go.

## Quick Navigation

- [Getting Started](Getting-Started) — Installation, build, and first run
- [Architecture Overview](Architecture-Overview) — System design and package structure
- [Order Types](Order-Types) — Limit orders, market orders, and their behavior
- [Matching Algorithm](Matching-Algorithm) — Price-time priority and the matching loop
- [Order Book](Order-Book) — Data structure, sorting, and operations
- [API Reference](API-Reference) — Public methods and usage examples
- [Testing](Testing) — Test suite and how to run it
- [Roadmap](Roadmap) — Planned features and contribution ideas

## What Is MatchEngine?

MatchEngine is a lightweight, in-memory trade matching engine that implements price-time priority (FIFO) order matching. It supports:

- Limit and market orders
- Partial fills
- Order cancellation
- Multi-symbol order books
- Thread-safe concurrent access

It is designed to be educational, readable, and extensible. No external dependencies — just the Go standard library.

## Repository

[https://github.com/iwtxokhtd83/MatchEngine](https://github.com/iwtxokhtd83/MatchEngine)
