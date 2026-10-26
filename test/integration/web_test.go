//go:build integration

package integration

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"example.com/acme/internal/app"
)

var csrfField = regexp.MustCompile(`name="_csrf" value="([^"]+)"`)

type browser struct {
	t      *testing.T
	srv    *httptest.Server
	client *http.Client
}

func newBrowser(t *testing.T, srv *httptest.Server) *browser {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &browser{t: t, srv: srv, client: client}
}

func (b *browser) get(path string) (int, string) {
	b.t.Helper()
	resp, err := b.client.Get(b.srv.URL + path)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func (b *browser) post(path string, form url.Values) (int, string) {
	b.t.Helper()
	resp, err := b.client.PostForm(b.srv.URL+path, form)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location")
}

// TestWebAndAPIShareOrders signs up and places an order through the HTML
// front-end, then reads it through the JSON API: both binaries go through
// the same service, so the order is identical.
func TestWebAndAPIShareOrders(t *testing.T) {
	core := newCore(t)
	webSrv := httptest.NewServer(app.NewWebServer(core))
	defer webSrv.Close()
	apiSrv := httptest.NewServer(app.NewAPIServer(core))
	defer apiSrv.Close()

	b := newBrowser(t, webSrv)
	if code, _ := b.get("/orders"); code != http.StatusSeeOther {
		t.Fatalf("anonymous /orders: %d, want redirect to login", code)
	}

	_, page := b.get("/signup")
	token := csrfField.FindStringSubmatch(page)[1]

	if code, _ := b.post("/signup", url.Values{"email": {"web@example.com"}, "name": {"Web"}, "password": {"12345678"}}); code != http.StatusForbidden {
		t.Fatalf("sign up without CSRF token: %d, want 403", code)
	}
	code, loc := b.post("/signup", url.Values{"_csrf": {token}, "email": {"web@example.com"}, "name": {"Web"}, "password": {"12345678"}})
	if code != http.StatusSeeOther || loc != "/orders" {
		t.Fatalf("sign up: %d %s", code, loc)
	}

	code, loc = b.post("/orders", url.Values{
		"_csrf": {token},
		"sku":   {"W1", "", ""}, "name": {"Widget", "", ""}, "quantity": {"4", "", ""}, "price": {"2.50", "", ""},
	})
	if code != http.StatusSeeOther || !strings.HasPrefix(loc, "/orders/") {
		t.Fatalf("create via web: %d %s", code, loc)
	}
	orderID := strings.TrimPrefix(loc, "/orders/")
	if code, page := b.get(loc); code != http.StatusOK || !strings.Contains(page, "10.00") {
		t.Fatalf("order page: %d", code)
	}

	api := &apiClient{t: t, srv: apiSrv}
	api.login("web@example.com", "12345678")
	var order struct {
		TotalCents int64
		Items      []struct{ SKU string }
	}
	if code := api.do(http.MethodGet, "/api/v1/orders/"+orderID, "", &order); code != http.StatusOK {
		t.Fatalf("read via API: %d", code)
	}
	if order.TotalCents != 1000 || len(order.Items) != 1 || order.Items[0].SKU != "W1" {
		t.Fatalf("API sees %+v", order)
	}

	if code, loc := b.post("/logout", url.Values{"_csrf": {token}}); code != http.StatusSeeOther || loc != "/login" {
		t.Fatalf("logout: %d %s", code, loc)
	}
	if code, _ := b.get("/orders"); code != http.StatusSeeOther {
		t.Fatalf("after logout /orders: %d, want redirect", code)
	}
}
