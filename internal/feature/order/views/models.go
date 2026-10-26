// Package views renders the order feature's pages from view models. It never
// imports the order package.
package views

import (
	"fmt"
	"strconv"
	"time"
)

type OrderRow struct {
	ID         string
	Status     string
	TotalCents int64
	CreatedAt  time.Time
}

type OrderList struct {
	Rows         []OrderRow
	Status       string
	Page         int
	TotalPages   int
	Total        int64
	ShowCustomer bool
}

type ItemLine struct {
	SKU            string
	Name           string
	Quantity       int
	UnitPriceCents int64
	LineTotalCents int64
}

type OrderDetail struct {
	ID         string
	CustomerID string
	Status     string
	Note       string
	TotalCents int64
	CreatedAt  time.Time
	Items      []ItemLine
	CanCancel  bool
}

// ItemRow is one editable row of the new-order form (raw strings, so a bad
// value can be shown back to the user as typed).
type ItemRow struct {
	SKU      string
	Name     string
	Quantity string
	Price    string
}

// Blank reports whether the row was left empty.
func (r ItemRow) Blank() bool {
	return r.SKU == "" && r.Name == "" && r.Quantity == "" && r.Price == ""
}

type NewOrderForm struct {
	Note string
	Rows []ItemRow
}

// Money formats cents as "1,234.50".
func Money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return fmt.Sprintf("%s%s.%02d", sign, whole, cents%100)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[len(id)-8:]
	}
	return id
}

func fieldKey(row int, field string) string {
	return fmt.Sprintf("items[%d].%s", row, field)
}

func pageURL(status string, page int) string {
	u := "/orders?page=" + strconv.Itoa(page)
	if status != "" {
		u += "&status=" + status
	}
	return u
}
