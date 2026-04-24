package model

import "github.com/shopspring/decimal"

// OrderRequest represents a request to place an order.
// The engine assigns a unique ID internally.
type OrderRequest struct {
	OwnerID   string
	Side      Side
	Type      OrderType
	TIF       TimeInForce     // default: GTC
	Price     decimal.Decimal // required for Limit/StopLimit orders
	StopPrice decimal.Decimal // required for StopMarket/StopLimit orders
	Quantity  decimal.Decimal
}

// NewLimitOrderRequest creates a limit order request with GTC time-in-force.
func NewLimitOrderRequest(side Side, price, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:     side,
		Type:     Limit,
		TIF:      GTC,
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

// NewIOCOrderRequest creates an Immediate-or-Cancel limit order request.
func NewIOCOrderRequest(side Side, price, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:     side,
		Type:     Limit,
		TIF:      IOC,
		Price:    price,
		Quantity: quantity,
	}
}

// NewFOKOrderRequest creates a Fill-or-Kill limit order request.
func NewFOKOrderRequest(side Side, price, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:     side,
		Type:     Limit,
		TIF:      FOK,
		Price:    price,
		Quantity: quantity,
	}
}

// NewStopMarketRequest creates a stop-market order request.
func NewStopMarketRequest(side Side, stopPrice, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:      side,
		Type:      StopMarket,
		StopPrice: stopPrice,
		Quantity:  quantity,
	}
}

// NewStopLimitRequest creates a stop-limit order request.
func NewStopLimitRequest(side Side, stopPrice, price, quantity decimal.Decimal) OrderRequest {
	return OrderRequest{
		Side:      side,
		Type:      StopLimit,
		StopPrice: stopPrice,
		Price:     price,
		Quantity:  quantity,
	}
}

// WithOwner returns a copy of the request with the OwnerID set.
func (r OrderRequest) WithOwner(ownerID string) OrderRequest {
	r.OwnerID = ownerID
	return r
}

// WithTIF returns a copy of the request with the TimeInForce set.
func (r OrderRequest) WithTIF(tif TimeInForce) OrderRequest {
	r.TIF = tif
	return r
}
