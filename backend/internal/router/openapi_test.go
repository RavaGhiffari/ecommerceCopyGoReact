package router

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// specPath is the hand-written OpenAPI document (PLAN.md Phase 7, Q5).
const specPath = "../../api/openapi.yaml"

var httpMethods = []string{"get", "post", "put", "patch", "delete"}

// loadSpec reads openapi.yaml into a generic tree, so the checks below work
// on exactly what is written (no generator, no extra dependency).
func loadSpec(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(specPath)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc), "openapi.yaml is not valid YAML")
	return doc
}

// ginToOpenAPI turns "/stores/:slug" into "/stores/{slug}".
var ginParam = regexp.MustCompile(`:([A-Za-z]+)`)

func ginToOpenAPI(path string) string { return ginParam.ReplaceAllString(path, "{$1}") }

// The spec and router.go must describe the same set of routes, in both
// directions: a new route without docs fails, and so does documentation of a
// route that was removed or renamed.
func TestOpenAPI_CoversExactlyTheRoutes(t *testing.T) {
	doc := loadSpec(t)
	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok, "spec has no paths")

	documented := map[string]bool{}
	for path, item := range paths {
		for method := range item.(map[string]any) {
			if slices.Contains(httpMethods, method) {
				documented[strings.ToUpper(method)+" "+path] = true
			}
		}
	}

	served := map[string]bool{}
	for _, r := range testRouter(t).Routes() {
		served[r.Method+" "+ginToOpenAPI(r.Path)] = true
	}

	for route := range served {
		assert.True(t, documented[route], "route %s is not in api/openapi.yaml", route)
	}
	for route := range documented {
		assert.True(t, served[route], "api/openapi.yaml documents %s, which router.go does not serve", route)
	}
	assert.Len(t, documented, len(served))
}

// Basic well-formedness that a reader (or a generator) relies on: unique
// operationIds, a success response everywhere, every {param} declared, and
// every $ref pointing at something that exists.
func TestOpenAPI_IsWellFormed(t *testing.T) {
	doc := loadSpec(t)
	assert.Equal(t, "3.1.0", doc["openapi"])

	ids := map[string]string{}
	for path, item := range doc["paths"].(map[string]any) {
		for method, rawOp := range item.(map[string]any) {
			if !slices.Contains(httpMethods, method) {
				continue
			}
			where := strings.ToUpper(method) + " " + path
			op := rawOp.(map[string]any)

			id, _ := op["operationId"].(string)
			if assert.NotEmpty(t, id, "%s has no operationId", where) {
				assert.Empty(t, ids[id], "operationId %q used by %s and %s", id, ids[id], where)
				ids[id] = where
			}

			responses, _ := op["responses"].(map[string]any)
			hasSuccess := false
			for code := range responses {
				hasSuccess = hasSuccess || strings.HasPrefix(code, "2")
			}
			assert.True(t, hasSuccess, "%s has no 2xx response", where)

			declared := map[string]bool{}
			params, _ := op["parameters"].([]any) // absent on operations without parameters
			for _, p := range params {
				param := resolve(t, doc, p.(map[string]any))
				if param["in"] == "path" {
					declared[param["name"].(string)] = true
				}
			}
			for _, m := range regexp.MustCompile(`\{([A-Za-z]+)\}`).FindAllStringSubmatch(path, -1) {
				assert.True(t, declared[m[1]], "%s does not declare path parameter %q", where, m[1])
			}
		}
	}

	walkRefs(doc, func(ref string) {
		assert.NotNil(t, lookup(doc, ref), "$ref %s does not resolve", ref)
	})
}

// resolve follows a {"$ref": "#/components/..."} once.
func resolve(t *testing.T, doc, node map[string]any) map[string]any {
	ref, ok := node["$ref"].(string)
	if !ok {
		return node
	}
	target, ok := lookup(doc, ref).(map[string]any)
	require.True(t, ok, "$ref %s does not resolve", ref)
	return target
}

func lookup(doc map[string]any, ref string) any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var cur any = doc
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	return cur
}

func walkRefs(node any, visit func(string)) {
	switch n := node.(type) {
	case map[string]any:
		for k, v := range n {
			if k == "$ref" {
				visit(fmt.Sprint(v))
				continue
			}
			walkRefs(v, visit)
		}
	case []any:
		for _, v := range n {
			walkRefs(v, visit)
		}
	}
}
