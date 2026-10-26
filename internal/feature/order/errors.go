package order

import "example.com/acme/internal/platform/httpx"

var (
	ErrOrderNotFound    = httpx.New(httpx.KindNotFound, "order.not_found", "Order not found")
	ErrOrderClosed      = httpx.New(httpx.KindUnprocessable, "order.closed", "This order is cancelled and can no longer change")
	ErrCustomerNotFound = httpx.New(httpx.KindUnprocessable, "order.customer_not_found", "Customer does not exist")
	ErrOtherCustomer    = httpx.New(httpx.KindForbidden, "order.other_customer", "You can only create orders for yourself")
)
