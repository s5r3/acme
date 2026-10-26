//go:build integration

package integration

import (
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"example.com/acme/api/openapi"
	"example.com/acme/internal/app"
)

// TestOpenAPICoversRoutes fails when a route is missing from the spec, or the
// spec documents a route that does not exist.
func TestOpenAPICoversRoutes(t *testing.T) {
	spec := specOperations(t)

	e := app.NewAPIServer(newCore(t))
	param := regexp.MustCompile(`:([A-Za-z]+)`)
	var routes []string
	for _, r := range e.Router().Routes() {
		if strings.HasPrefix(r.Path, "/api/v1") && isHTTPMethod(r.Method) {
			routes = append(routes, r.Method+" "+param.ReplaceAllString(r.Path, "{$1}"))
		}
	}

	for _, r := range routes {
		if !slices.Contains(spec, r) {
			t.Errorf("route %s is not documented in api/openapi", r)
		}
	}
	for _, op := range spec {
		if !slices.Contains(routes, op) {
			t.Errorf("spec documents %s but no such route is registered", op)
		}
	}
}

// specOperations returns "METHOD /path" for every operation in the spec,
// following the per-feature $ref files.
func specOperations(t *testing.T) []string {
	t.Helper()
	var root struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	readYAML(t, "openapi.yaml", &root)

	var ops []string
	for p, item := range root.Paths {
		if ref, ok := item["$ref"].(string); ok {
			item = resolve(t, ref)
		}
		for method := range item {
			if isHTTPMethod(strings.ToUpper(method)) {
				ops = append(ops, strings.ToUpper(method)+" "+p)
			}
		}
	}
	return ops
}

// resolve loads "./paths/order.yaml#/orders".
func resolve(t *testing.T, ref string) map[string]any {
	t.Helper()
	file, pointer, _ := strings.Cut(ref, "#/")
	var doc map[string]map[string]any
	readYAML(t, path.Clean(file), &doc)
	item, ok := doc[pointer]
	if !ok {
		t.Fatalf("unresolved $ref %s", ref)
	}
	return item
}

func readYAML(t *testing.T, name string, out any) {
	t.Helper()
	data, err := fs.ReadFile(openapi.FS, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}
