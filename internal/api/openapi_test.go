package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAPIContractIsValidAndCoversRegisteredRoutes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "openapi.json"))
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}

	var document struct {
		OpenAPI string                                `json:"openapi"`
		Paths   map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
	}

	wants := map[string][]string{
		"/health/live": {"get"}, "/health/ready": {"get"},
		"/payments": {"post"}, "/payments/{paymentId}": {"get"},
		"/payments/{paymentId}/authorize": {"post"},
		"/payments/{paymentId}/capture":   {"post"},
		"/payments/{paymentId}/cancel":    {"post"},
		"/payments/{paymentId}/refund":    {"post"},
		"/webhook-endpoints":              {"get", "post"},
		"/webhook-endpoints/{endpointId}": {"get"},
		"/admin/time":                     {"get"}, "/admin/time/advance": {"post"},
		"/admin/providers": {"get"}, "/admin/scenarios/{scenarioId}": {"get"},
		"/admin/payments/{paymentId}/timeline":    {"get"},
		"/admin/diagnostics/payments/{paymentId}": {"get"},
	}
	for route, methods := range wants {
		for _, method := range methods {
			if _, ok := document.Paths[route][method]; !ok {
				t.Errorf("OpenAPI contract is missing %s %s", method, route)
			}
		}
	}
}
