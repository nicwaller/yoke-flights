package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRender_ResourceCount(t *testing.T) {
	resources, err := render("ots", "ots-ns", defaults)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(resources) != 6 {
		t.Errorf("expected 6 resources, got %d", len(resources))
	}
}

func TestRender_ResourcesAreValidJSON(t *testing.T) {
	resources, err := render("ots", "ots-ns", defaults)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	for i, r := range resources {
		var obj map[string]any
		if err := json.Unmarshal(r, &obj); err != nil {
			t.Errorf("resource %d is not valid JSON: %v", i, err)
		}
	}
}

func TestRender_RejectsInvalidValues(t *testing.T) {
	v := defaults
	v.Domain = "bad:domain"
	if _, err := render("ots", "ots-ns", v); err == nil {
		t.Error("expected error for invalid domain")
	}
}

func TestRender_RedisURLPointsToSidecar(t *testing.T) {
	resources, err := render("myots", "myns", defaults)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	raw, err := json.Marshal(resources)
	if err != nil {
		t.Fatalf("failed to marshal resources: %v", err)
	}

	rawStr := string(raw)
	expected := "redis://myots-redis:6379/0"
	if !strings.Contains(rawStr, expected) {
		t.Errorf("expected REDIS_URL %q in rendered output", expected)
	}
}
