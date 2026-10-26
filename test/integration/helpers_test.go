//go:build integration

// Package integration runs both binaries' Echo instances in-process against
// a real PostgreSQL (ACME_TEST_DATABASE_DSN) and drives them over HTTP.
package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/acme/internal/app"
	"example.com/acme/internal/platform/config"
	"example.com/acme/internal/platform/database/dbtest"
)

// newCore builds the same Core the binaries use, pointed at the test DB.
func newCore(t *testing.T) *app.Core {
	t.Helper()
	dbtest.Open(t) // migrate + truncate

	cfg, err := config.Load("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Database.DSN = dbtest.DSN(t)
	cfg.Log.Stdout = false
	cfg.Log.File.Path = ""

	core, err := app.NewCore(context.Background(), cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(core.Close)
	return core
}

type apiClient struct {
	t     *testing.T
	srv   *httptest.Server
	token string
}

func (c *apiClient) do(method, path, body string, out any) int {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.srv.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.srv.Client().Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			c.t.Fatalf("decode %s %s: %v\n%s", method, path, err, data)
		}
	}
	return resp.StatusCode
}

// login signs up (if needed) and stores a bearer token.
func (c *apiClient) login(email, password string) {
	c.t.Helper()
	var tok struct{ AccessToken string }
	if code := c.do(http.MethodPost, "/api/v1/auth/token", `{"email":"`+email+`","password":"`+password+`"}`, &tok); code != http.StatusOK {
		c.t.Fatalf("login %s: status %d", email, code)
	}
	c.token = tok.AccessToken
}
