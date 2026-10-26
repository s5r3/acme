package order

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"example.com/acme/internal/feature/order/views"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/web"
	"example.com/acme/internal/shared/page"
)

type webHandler struct {
	svc service
}

// Index lists the caller's orders (all orders for admins).
func (h *webHandler) Index(c *echo.Context) error {
	status := c.QueryParam("status")
	if status != StatusOpen && status != StatusCancelled {
		status = ""
	}
	pg, _ := echo.QueryParamOr[int](c, "page", 1)
	res, err := h.svc.List(c.Request().Context(), ListFilter{Status: status}, page.New(pg, 10))
	if err != nil {
		return err
	}
	vm := views.OrderList{Status: status, Page: res.Page, TotalPages: res.TotalPages(), Total: res.Total, ShowCustomer: CanAdminister(c.Request().Context())}
	for _, o := range res.Items {
		vm.Rows = append(vm.Rows, toRow(o))
	}
	return web.Render(c, http.StatusOK, views.List(vm))
}

// New shows an empty order form.
func (h *webHandler) New(c *echo.Context) error {
	f := CreateOrderForm{Rows: make([]views.ItemRow, FormRows)}
	return web.Render(c, http.StatusOK, views.Form(views.NewOrderForm{Note: f.Note, Rows: f.Rows}, nil, ""))
}

// Create handles the form post: parse -> map to CreateOrderInput -> the same
// Service.Create the API uses -> redirect (post/redirect/get).
func (h *webHandler) Create(c *echo.Context) error {
	f, err := parseCreateOrderForm(c.Request())
	if err != nil {
		return httpx.ErrMalformedRequest.Wrap(err)
	}
	render := func(fields map[string]string, msg string) error {
		return web.Render(c, http.StatusUnprocessableEntity, views.Form(views.NewOrderForm{Note: f.Note, Rows: f.Rows}, fields, msg))
	}

	in, rowOf, fieldErrs := f.toInput()
	if fieldErrs != nil {
		return render(fieldErrs, "")
	}
	o, err := h.svc.Create(c.Request().Context(), in)
	if err != nil {
		if appErr, ok := httpx.AsError(err); ok && len(appErr.Fields) > 0 {
			return render(remapItemErrors(appErr.Fields, rowOf), "")
		}
		return web.HandleFormError(err, render)
	}
	web.SetFlash(c, "success", "Order placed.")
	return web.RedirectSeeOther(c, "/orders/"+o.ID.String())
}

func (h *webHandler) Show(c *echo.Context) error {
	orderID, err := orderIDParam(c)
	if err != nil {
		return err
	}
	o, err := h.svc.Get(c.Request().Context(), orderID)
	if err != nil {
		return err
	}
	return web.Render(c, http.StatusOK, views.Show(toDetail(*o, true)))
}

// Cancel shows the business error as a flash message instead of an error page.
func (h *webHandler) Cancel(c *echo.Context) error {
	orderID, err := orderIDParam(c)
	if err != nil {
		return err
	}
	back := "/orders/" + orderID.String()
	if _, err := h.svc.Cancel(c.Request().Context(), orderID); err != nil {
		if errors.Is(err, ErrOrderClosed) {
			web.SetFlash(c, "error", ErrOrderClosed.Message)
			return web.RedirectSeeOther(c, back)
		}
		return err
	}
	web.SetFlash(c, "success", "Order cancelled.")
	return web.RedirectSeeOther(c, back)
}
