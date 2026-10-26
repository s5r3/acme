package __name__

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/shared/page"
)

type service interface {
	Create(ctx context.Context, in Create__Name__Input) (*__Name__, error)
	Get(ctx context.Context, id uuid.UUID) (*__Name__, error)
	List(ctx context.Context, q page.Query) (page.Result[__Name__], error)
}

type apiHandler struct {
	svc service
}

func (h *apiHandler) List(c *echo.Context) error {
	var q List__Name__sQuery
	if err := httpx.BindAndValidate(c, &q); err != nil {
		return err
	}
	res, err := h.svc.List(c.Request().Context(), page.New(q.Page, q.PageSize))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.NewList(res, to__Name__Response))
}

func (h *apiHandler) Create(c *echo.Context) error {
	var req Create__Name__Request
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	x, err := h.svc.Create(c.Request().Context(), req.toInput())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, to__Name__Response(*x))
}

func (h *apiHandler) Get(c *echo.Context) error {
	id, err := echo.PathParam[uuid.UUID](c, "__name__ID")
	if err != nil {
		return Err__Name__NotFound
	}
	x, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, to__Name__Response(*x))
}
