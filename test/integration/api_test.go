//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"example.com/acme/internal/app"
)

func TestAPIOrderFlow(t *testing.T) {
	srv := httptest.NewServer(app.NewAPIServer(newCore(t)))
	defer srv.Close()
	c := &apiClient{t: t, srv: srv}

	if code := c.do(http.MethodPost, "/api/v1/users", `{"email":"flow@example.com","name":"Flow","password":"12345678"}`, nil); code != http.StatusCreated {
		t.Fatalf("sign up: %d", code)
	}
	if code := c.do(http.MethodPost, "/api/v1/users", `{"email":"flow@example.com","name":"Flow","password":"12345678"}`, nil); code != http.StatusConflict {
		t.Fatalf("duplicate sign up: %d, want 409", code)
	}
	c.login("flow@example.com", "12345678")

	var order struct {
		ID         string
		TotalCents int64
		Status     string
	}
	code := c.do(http.MethodPost, "/api/v1/orders", `{"items":[{"sku":"A","name":"Apple","quantity":2,"unitPriceCents":150}]}`, &order)
	if code != http.StatusCreated || order.TotalCents != 300 {
		t.Fatalf("create: %d %+v", code, order)
	}
	if code := c.do(http.MethodPost, "/api/v1/orders/"+order.ID+"/items", `{"sku":"B","name":"Bread","quantity":1,"unitPriceCents":250}`, nil); code != http.StatusCreated {
		t.Fatalf("add item: %d", code)
	}
	c.do(http.MethodGet, "/api/v1/orders/"+order.ID, "", &order)
	if order.TotalCents != 550 {
		t.Fatalf("total after add = %d, want 550", order.TotalCents)
	}
	if code := c.do(http.MethodPost, "/api/v1/orders/"+order.ID+"/cancel", "", &order); code != http.StatusOK || order.Status != "cancelled" {
		t.Fatalf("cancel: %d %s", code, order.Status)
	}
	var problem struct{ Code string }
	if code := c.do(http.MethodPost, "/api/v1/orders/"+order.ID+"/cancel", "", &problem); code != http.StatusUnprocessableEntity || problem.Code != "order.closed" {
		t.Fatalf("second cancel: %d %s", code, problem.Code)
	}

	// Another customer cannot see the order: it is reported as not found.
	c.do(http.MethodPost, "/api/v1/users", `{"email":"other@example.com","name":"Other","password":"12345678"}`, nil)
	c.login("other@example.com", "12345678")
	if code := c.do(http.MethodGet, "/api/v1/orders/"+order.ID, "", nil); code != http.StatusNotFound {
		t.Fatalf("other customer's view: %d, want 404", code)
	}
	if code := c.do(http.MethodGet, "/api/v1/users", "", nil); code != http.StatusForbidden {
		t.Fatalf("customer listing users: %d, want 403", code)
	}
}

// publicRoutes are the only API routes reachable without a token.
var publicRoutes = []string{
	"POST /api/v1/auth/token",
	"POST /api/v1/users",
}

// TestEveryProtectedRouteRequiresAuth calls every registered API route
// without credentials. A new route that forgets its guard fails here.
func TestEveryProtectedRouteRequiresAuth(t *testing.T) {
	e := app.NewAPIServer(newCore(t))
	param := regexp.MustCompile(`:[A-Za-z]+`)

	checked := 0
	for _, r := range e.Router().Routes() {
		if !strings.HasPrefix(r.Path, "/api/v1") || !isHTTPMethod(r.Method) {
			continue
		}
		key := r.Method + " " + r.Path
		if slices.Contains(publicRoutes, key) {
			continue
		}
		path := param.ReplaceAllString(r.Path, "01a158d8-0000-7000-8000-000000000000")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(r.Method, path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s answered %d without a token, want 401", key, rec.Code)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no routes checked")
	}
}

func isHTTPMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
