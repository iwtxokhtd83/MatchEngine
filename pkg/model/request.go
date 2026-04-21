package model

import "github.com/shopspring/decimal"

// OrderRequest represents a request to place an order.
// The engine assigns a unique ID internally.
type OrderRequest struct {
	OwnerID  string          // identifies the trader (used for self-trade prevention)
	Side     Side
	Type     OrderType
	Price    decimal.Decimal // required for Limit orders, ignored for Market
	Quantity decimal.Decimal
}

// NewLimitOrderRequest creates a limit order request.
func NewLimitOrderRequest(side Side, price, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:     side,
		Type:     Limit,
		Price:    price,
		Quantity: quantity,
	}
}

// NewMarketOrderRequest creates a market order request.
func NewMarketOrderRequest(side Side, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:     side,
		Type:     Market,
		Quantity: quantity,
	}
}

// WithOwner returns a copy of the request with the OwnerID set.
func (r OrderRequest) WithOwner(ownerID string) OrderRequest {
	r.OwnerID = ownerID
	return r
}
