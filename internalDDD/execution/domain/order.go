package domain

import (
	exchangeDomain "traiderBot/internalDDD/exchange/domain"
)

type OrderStatus = exchangeDomain.OrderStatus

const (
	OrderStatusPending         = exchangeDomain.OrderStatusPending
	OrderStatusPartiallyFilled = exchangeDomain.OrderStatusPartiallyFilled
	OrderStatusFilled          = exchangeDomain.OrderStatusFilled
	OrderStatusRejected        = exchangeDomain.OrderStatusRejected
	OrderStatusCancelled       = exchangeDomain.OrderStatusCancelled
	OrderStatusUnknown         = exchangeDomain.OrderStatusUnknown
)

type OrderRequest = exchangeDomain.OrderRequest
type OrderResult = exchangeDomain.OrderResult

var NewOrderRequest = exchangeDomain.NewOrderRequest