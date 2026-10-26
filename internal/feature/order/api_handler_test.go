package order

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/shared/page"
)

// fakeService records the input it got and returns canned results.
type fakeService struct {
	gotCreate CreateOrderInput
	createErr error
	cancelErr error
}

func (f *fakeService) Create(_ context.Context, in CreateOrderInput) (*Order, error) {
	f.gotCreate = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	o := &Order{ID: uuid.New(), CustomerID: customerID, Status: StatusOpen}
	for _, it := range in.Items {
		o.Items = append(o.Items, Item{SKU: it.SKU, Name: it.Name, Quantity: it.Quantity, UnitPriceCents: it.UnitPriceCents})
		o.TotalCents += int64(it.Quantity) * it.UnitPriceCents
	}
	return o, nil
}

func (f *fakeService) Get(context.Context, uuid.UUID) (*Order, error) { return nil, ErrOrderNotFound }

func (f *fakeService) List(context.Context, ListFilter, page.Query) (page.Result[Order], error) {
	return page.Result[Order]{Items: []Order{{ID: uuid.New(), Status: StatusOpen}}, Page: 1, PageSize: 20, Total: 1}, nil
}

func (f *fakeService) Cancel(context.Context, uuid.UUID) (*Order, error) { return nil, f.cancelErr }

func (f *fakeService) AddItem(context.Context, uuid.UUID, ItemInput) (*Item, error) {
	return &Item{}, nil
}

// newTestEcho builds a real Echo with the API error handler and validator,
// and a middleware that authenticates every request as a customer.
func newTestEcho(svc service) *echo.Echo {
	e := echo.New()
	e.Validator = httpx.NewValidator()
	e.HTTPErrorHandler = httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			p := &auth.Principal{UserID: customerID}
			c.SetRequest(c.Request().WithContext(auth.WithPrincipal(c.Request().Context(), p)))
			return next(c)
		}
	})
	h := &apiHandler{svc: svc}
	e.POST("/orders", h.Create)
	e.GET("/orders", h.List)
	e.GET("/orders/:orderID", h.Get)
	e.POST("/orders/:orderID/cancel", h.Cancel)
	return e
}

func do(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAPICreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svcErr     error
		wantStatus int
		wantCode   string
	}{
		{"valid", `{"note":"hi","items":[{"sku":"A","name":"Apple","quantity":2,"unitPriceCents":150}]}`, nil, http.StatusCreated, ""},
		{"malformed json", `{"items":`, nil, http.StatusBadRequest, "request.malformed"},
		{"no items", `{"items":[]}`, nil, http.StatusBadRequest, "validation.failed"},
		{"bad item", `{"items":[{"sku":"","name":"x","quantity":0}]}`, nil, http.StatusBadRequest, "validation.failed"},
		{"business rule", `{"items":[{"sku":"A","name":"Apple","quantity":1}]}`, ErrCustomerNotFound, http.StatusUnprocessableEntity, "order.customer_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{createErr: tt.svcErr}
			rec := do(newTestEcho(svc), http.MethodPost, "/orders", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantCode != "" {
				var p httpx.Problem
				_ = json.Unmarshal(rec.Body.Bytes(), &p)
				if p.Code != tt.wantCode {
					t.Fatalf("code = %q, want %q", p.Code, tt.wantCode)
				}
				if ct := rec.Header().Get(echo.HeaderContentType); ct != "application/problem+json" {
					t.Fatalf("content-type = %q", ct)
				}
			}
		})
	}
}

func TestAPICreateMapsDTOToInput(t *testing.T) {
	svc := &fakeService{}
	rec := do(newTestEcho(svc), http.MethodPost, "/orders", `{"note":"n","items":[{"sku":"A","name":"Apple","quantity":2,"unitPriceCents":150}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d", rec.Code)
	}
	in := svc.gotCreate
	if in.Note != "n" || len(in.Items) != 1 || in.Items[0].UnitPriceCents != 150 || in.CustomerID != nil {
		t.Fatalf("service input = %+v", in)
	}
	var resp OrderResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.TotalCents != 300 || rec.Header().Get(echo.HeaderLocation) == "" {
		t.Fatalf("response total=%d location=%q", resp.TotalCents, rec.Header().Get(echo.HeaderLocation))
	}
}

func TestAPIGetMalformedIDIsNotFound(t *testing.T) {
	rec := do(newTestEcho(&fakeService{}), http.MethodGet, "/orders/not-a-uuid", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAPIListEnvelope(t *testing.T) {
	rec := do(newTestEcho(&fakeService{}), http.MethodGet, "/orders?status=open&page=1", "")
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	for _, k := range []string{"items", "page", "pageSize", "total"} {
		if _, ok := body[k]; !ok {
			t.Fatalf("list response missing %q: %s", k, rec.Body)
		}
	}
	if rec := do(newTestEcho(&fakeService{}), http.MethodGet, "/orders?status=bogus", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad status filter: %d, want 400", rec.Code)
	}
}
