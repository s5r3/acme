package order

import (
	"time"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/database"
)

// Order statuses.
const (
	StatusOpen      = "open"
	StatusCancelled = "cancelled"
)

// Order is the orders table.
type Order struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	CustomerID uuid.UUID `gorm:"type:uuid"`
	Status     string
	Note       string
	TotalCents int64
	Items      []Item `gorm:"foreignKey:OrderID"`
	database.Timestamps
}

func (Order) TableName() string { return "orders" }

// Item is one line of an order (order_items table).
type Item struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID        uuid.UUID `gorm:"type:uuid"`
	SKU            string
	Name           string
	Quantity       int
	UnitPriceCents int64
	CreatedAt      time.Time
}

func (Item) TableName() string { return "order_items" }

// LineTotalCents is quantity x unit price.
func (i Item) LineTotalCents() int64 { return int64(i.Quantity) * i.UnitPriceCents }

// IsOpen reports whether the order can still change.
func (o *Order) IsOpen() bool { return o.Status == StatusOpen }
