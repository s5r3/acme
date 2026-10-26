package order

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/web"
)

func newWebTestEcho(svc service) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = web.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			p := &auth.Principal{UserID: customerID, Name: "Carl"}
			c.SetRequest(c.Request().WithContext(auth.WithPrincipal(c.Request().Context(), p)))
			return next(c)
		}
	}, web.PageData)
	h := &webHandler{svc: svc}
	e.POST("/orders", h.Create)
	e.POST("/orders/:orderID/cancel", h.Cancel)
	return e
}

func postForm(e *echo.Echo, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestWebCreateRedirectsAndUsesSameService(t *testing.T) {
	svc := &fakeService{}
	form := url.Values{
		"sku":      {"", "A1", ""},
		"name":     {"", "Apple", ""},
		"quantity": {"", "3", ""},
		"price":    {"", "1.2", ""},
		"note":     {"from the web"},
	}
	rec := postForm(newWebTestEcho(svc), "/orders", form)
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get(echo.HeaderLocation), "/orders/") {
		t.Fatalf("status %d location %q", rec.Code, rec.Header().Get(echo.HeaderLocation))
	}
	got := svc.gotCreate
	if got.Note != "from the web" || len(got.Items) != 1 || got.Items[0].UnitPriceCents != 120 || got.Items[0].Quantity != 3 {
		t.Fatalf("service input = %+v", got)
	}
}

func TestWebCreateRerendersFormOnErrors(t *testing.T) {
	form := url.Values{"sku": {"A1"}, "name": {"Apple"}, "quantity": {"two"}, "price": {"1.20"}}
	rec := postForm(newWebTestEcho(&fakeService{}), "/orders", form)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "must be a whole number above 0") || !strings.Contains(body, `value="two"`) {
		t.Fatalf("form not re-rendered with the error and typed value:\n%s", body)
	}
}

func TestWebCancelClosedOrderShowsFlash(t *testing.T) {
	rec := postForm(newWebTestEcho(&fakeService{cancelErr: ErrOrderClosed}), "/orders/"+customerID.String()+"/cancel", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "acme_flash=") {
		t.Fatalf("expected a flash cookie")
	}
}

func TestParseMoney(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"12", 1200, true}, {"12.5", 1250, true}, {"12.50", 1250, true}, {"0.05", 5, true},
		{"", 0, false}, {"1.234", 0, false}, {"-1", 0, false}, {"abc", 0, false}, {".5", 0, false},
	}
	for _, tt := range tests {
		got, err := parseMoney(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("parseMoney(%q) = %d, %v; want %d ok=%v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}

func TestRemapItemErrors(t *testing.T) {
	got := remapItemErrors(map[string]string{"items[0].sku": "is required", "items": "x"}, []int{2})
	if got["items[2].sku"] != "is required" || got["items"] != "x" {
		t.Fatalf("got %v", got)
	}
}
