package httpapi

import (
	"encoding/json"
	"testing"
)

func TestOpenAPISpecIncludesRequiredContract(t *testing.T) {
	var spec map[string]any
	if err := json.Unmarshal(OpenAPISpecJSON(), &spec); err != nil {
		t.Fatalf("OpenAPI JSON should decode: %v", err)
	}
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %#v", spec["openapi"])
	}
	paths := spec["paths"].(map[string]any)
	for _, path := range []string{"/api/auth/qr/exchange", "/api/keys", "/api/courses", "/api/openapi.json"} {
		if paths[path] == nil {
			t.Fatalf("missing OpenAPI path %s", path)
		}
	}
}
