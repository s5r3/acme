package order

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/shared/page"
)

// service is the subset of Service the handlers use (both front-ends).
// Declared here so handler tests can use a fake.
type service interface {
	Create(ctx context.Context, in CreateOrderInput) (*Order, error)
	Get(ctx context.Context, orderID uuid.UUID) (*Order, error)
	List(ctx context.Context, f ListFilter, q page.Query) (page.Result[Order], error)
	Cancel(ctx context.Context, orderID uuid.UUID) (*Order, error)
	AddItem(ctx context.Context, orderID uuid.UUID, in ItemInput) (*Item, error)
}

type apiHandler struct {
	svc service
}

func (h *apiHandler) List(c *echo.Context) error {
	var q ListOrdersQuery
	if err := httpx.BindAndValidate(c, &q); err != nil {
		return err
	}
	res, err := h.svc.List(c.Request().Context(), ListFilter{Status: q.Status, CustomerID: q.CustomerID}, page.New(q.Page, q.PageSize))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.NewList(res, toOrderSummary))
}

func (h *apiHandler) Create(c *echo.Context) error {
	var req CreateOrderRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	o, err := h.svc.Create(c.Request().Context(), req.toInput())
	if err != nil {
		return err
	}
	c.Response().Header().Set(echo.HeaderLocation, "/api/v1/orders/"+o.ID.String())
	return c.JSON(http.StatusCreated, toOrderResponse(*o))
}

func (h *apiHandler) Get(c *echo.Context) error {
	orderID, err := orderIDParam(c)
	if err != nil {
		return err
	}
	o, err := h.svc.Get(c.Request().Context(), orderID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toOrderResponse(*o))
}

func (h *apiHandler) Cancel(c *echo.Context) error {
	orderID, err := orderIDParam(c)
	if err != nil {
		return err
	}
	o, err := h.svc.Cancel(c.Request().Context(), orderID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toOrderResponse(*o))
}
