package engine

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

func BenchmarkInsertOnly(b *testing.B) {
	e := New(WithMaxTradeLog(0))
	price := decimal.NewFromInt(100)
	qty := decimal.NewFromInt(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("s%d", i)
		e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Sell, price, qty))
	}
}

func BenchmarkInsertAndMatch(b *testing.B) {
	e := New(WithMaxTradeLog(0))
	price := decimal.NewFromInt(100)
	qty := decimal.NewFromInt(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("o%d", i)
		if i%2 == 0 {
			e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Sell, price, qty))
		} else {
			e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Buy, price, qty))
		}
	}
}

func BenchmarkInsertMultiplePriceLevels(b *testing.B) {
	e := New(WithMaxTradeLog(0))
	qty := decimal.NewFromInt(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("s%d", i)
		price := decimal.NewFromInt(int64(1000 + i%100))
		e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Sell, price, qty))
	}
}
