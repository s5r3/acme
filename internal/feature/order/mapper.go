package order

import (
	"example.com/acme/internal/feature/order/views"
)

// API: DTO -> service input

func (r CreateOrderRequest) toInput() CreateOrderInput {
	in := CreateOrderInput{CustomerID: r.CustomerID, Note: r.Note}
	for _, it := range r.Items {
		in.Items = append(in.Items, it.toInput())
	}
	return in
}

func (r OrderItemRequest) toInput() ItemInput {
	return ItemInput{SKU: r.SKU, Name: r.Name, Quantity: r.Quantity, UnitPriceCents: r.UnitPriceCents}
}

// API: entity -> DTO

func toOrderResponse(o Order) OrderResponse {
	items := make([]OrderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, toItemResponse(it))
	}
	return OrderResponse{
		ID: o.ID, CustomerID: o.CustomerID, Status: o.Status, Note: o.Note,
		TotalCents: o.TotalCents, Items: items, CreatedAt: o.CreatedAt.UTC(), UpdatedAt: o.UpdatedAt.UTC(),
	}
}

func toOrderSummary(o Order) OrderSummaryResponse {
	return OrderSummaryResponse{ID: o.ID, CustomerID: o.CustomerID, Status: o.Status, TotalCents: o.TotalCents, CreatedAt: o.CreatedAt.UTC()}
}

func toItemResponse(it Item) OrderItemResponse {
	return OrderItemResponse{
		ID: it.ID, SKU: it.SKU, Name: it.Name, Quantity: it.Quantity,
		UnitPriceCents: it.UnitPriceCents, LineTotalCents: it.LineTotalCents(),
	}
}

// Web: entity -> view model

func toRow(o Order) views.OrderRow {
	return views.OrderRow{ID: o.ID.String(), Status: o.Status, TotalCents: o.TotalCents, CreatedAt: o.CreatedAt.UTC()}
}

func toDetail(o Order, canCancel bool) views.OrderDetail {
	d := views.OrderDetail{
		ID: o.ID.String(), CustomerID: o.CustomerID.String(), Status: o.Status, Note: o.Note,
		TotalCents: o.TotalCents, CreatedAt: o.CreatedAt.UTC(), CanCancel: canCancel && o.IsOpen(),
	}
	for _, it := range o.Items {
		d.Items = append(d.Items, views.ItemLine{
			SKU: it.SKU, Name: it.Name, Quantity: it.Quantity,
			UnitPriceCents: it.UnitPriceCents, LineTotalCents: it.LineTotalCents(),
		})
	}
	return d
}
