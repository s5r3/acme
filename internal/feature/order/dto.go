package order

import (
	"time"

	"github.com/google/uuid"
)

// API request/response structs. Money is in integer cents to avoid float errors.

type CreateOrderRequest struct {
	CustomerID *uuid.UUID         `json:"customerId"` // admins only; defaults to the caller
	Note       string             `json:"note" validate:"max=500"`
	Items      []OrderItemRequest `json:"items" validate:"required,min=1,max=50,dive"`
}

type OrderItemRequest struct {
	SKU            string `json:"sku" validate:"required,max=64"`
	Name           string `json:"name" validate:"required,max=200"`
	Quantity       int    `json:"quantity" validate:"required,gt=0,lte=1000"`
	UnitPriceCents int64  `json:"unitPriceCents" validate:"gte=0"`
}

type ListOrdersQuery struct {
	Status     string     `query:"status" validate:"omitempty,oneof=open cancelled"`
	CustomerID *uuid.UUID `query:"customerId"` // honoured for admins only
	Page       int        `query:"page"`
	PageSize   int        `query:"pageSize" validate:"omitempty,max=100"`
}

type OrderResponse struct {
	ID         uuid.UUID           `json:"id"`
	CustomerID uuid.UUID           `json:"customerId"`
	Status     string              `json:"status"`
	Note       string              `json:"note"`
	TotalCents int64               `json:"totalCents"`
	Items      []OrderItemResponse `json:"items"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
}

type OrderSummaryResponse struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customerId"`
	Status     string    `json:"status"`
	TotalCents int64     `json:"totalCents"`
	CreatedAt  time.Time `json:"createdAt"`
}

type OrderItemResponse struct {
	ID             uuid.UUID `json:"id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	Quantity       int       `json:"quantity"`
	UnitPriceCents int64     `json:"unitPriceCents"`
	LineTotalCents int64     `json:"lineTotalCents"`
}
