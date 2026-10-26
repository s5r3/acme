package order

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/httpx"
)

// Handlers for the /orders/:orderID/items sub-resource. Large features split
// handler files by sub-resource like this.

func (h *apiHandler) ListItems(c *echo.Context) error {
	orderID, err := orderIDParam(c)
	if err != nil {
		return err
	}
	o, err := h.svc.Get(c.Request().Context(), orderID)
	if err != nil {
		return err
	}
	items := make([]OrderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, toItemResponse(it))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *apiHandler) AddItem(c *echo.Context) error {
	orderID, err := orderIDParam(c)
	if err != nil {
		return err
	}
	var req OrderItemRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	item, err := h.svc.AddItem(c.Request().Context(), orderID, req.toInput())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toItemResponse(*item))
}

// orderIDParam reads :orderID; a malformed ID is simply not found.
func orderIDParam(c *echo.Context) (uuid.UUID, error) {
	id, err := echo.PathParam[uuid.UUID](c, "orderID")
	if err != nil {
		return uuid.Nil, ErrOrderNotFound
	}
	return id, nil
}
