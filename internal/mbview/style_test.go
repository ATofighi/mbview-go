package mbview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeMapboxStyle(t *testing.T) {
	owner, style := normalizeMapboxStyle("dark")
	if owner != "mapbox" || style != "dark-v11" {
		t.Fatalf("unexpected normalize output: %s/%s", owner, style)
	}

	owner, style = normalizeMapboxStyle("customuser/customstyle")
	if owner != "customuser" || style != "customstyle" {
		t.Fatalf("unexpected custom style output: %s/%s", owner, style)
	}
}

func TestRewriteStyleReference(t *testing.T) {
	token := "pk.test"

	styleURL := rewriteStyleReference("mapbox://styles/mapbox/streets-v12", token)
	if !strings.Contains(styleURL, "https://api.mapbox.com/styles/v1/mapbox/streets-v12") {
		t.Fatalf("unexpected style URL: %s", styleURL)
	}
	if !strings.Contains(styleURL, "access_token=pk.test") {
		t.Fatalf("missing token in style URL: %s", styleURL)
	}

	tileJSONURL := rewriteStyleReference("mapbox://mapbox.mapbox-streets-v8", token)
	if !strings.Contains(tileJSONURL, "https://api.mapbox.com/v4/mapbox.mapbox-streets-v8.json") {
		t.Fatalf("unexpected tilejson URL: %s", tileJSONURL)
	}

	httpURL := rewriteStyleReference("https://api.mapbox.com/fonts/v1/mapbox/{fontstack}/{range}.pbf", token)
	if !strings.Contains(httpURL, "access_token=pk.test") {
		t.Fatalf("missing token in http mapbox URL: %s", httpURL)
	}
	if !strings.Contains(httpURL, "{fontstack}") || !strings.Contains(httpURL, "{range}") {
		t.Fatalf("font tokens must be preserved: %s", httpURL)
	}

	fontURL := rewriteStyleReference("mapbox://fonts/mapbox/{fontstack}/{range}.pbf", token)
	if !strings.Contains(fontURL, "{fontstack}") || !strings.Contains(fontURL, "{range}") {
		t.Fatalf("mapbox font tokens must be preserved: %s", fontURL)
	}
	if !strings.Contains(fontURL, "access_token=pk.test") {
		t.Fatalf("missing token in mapbox font URL: %s", fontURL)
	}
}

func TestResolveCustomBasemapFile(t *testing.T) {
	tmp := t.TempDir()
	styleFile := filepath.Join(tmp, "style.json")
	if err := os.WriteFile(styleFile, []byte(`{"version":8,"sources":{},"layers":[]}`), 0o600); err != nil {
		t.Fatalf("failed to write style file: %v", err)
	}

	basemap, err := resolveCustomBasemap(styleFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if basemap.StyleURL != "/basemap/style.json" {
		t.Fatalf("unexpected style url: %s", basemap.StyleURL)
	}
	if len(basemap.StyleJSON) == 0 {
		t.Fatal("expected style JSON bytes")
	}
}

func TestSanitizeMapboxStyleForMapLibre(t *testing.T) {
	input := map[string]any{
		"version": 8,
		"name":    "Mapbox Dark",
		"owner":   "mapbox",
		"id":      "dark-v11",
		"projection": map[string]any{
			"name": "globe",
		},
		"sources": map[string]any{
			"composite": map[string]any{
				"type": "vector",
				"url":  "mapbox://mapbox.mapbox-streets-v8",
				"name": "Mapbox Streets",
			},
		},
	}

	outputAny := sanitizeMapboxStyleForMapLibre(input)
	output, ok := outputAny.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got %T", outputAny)
	}

	if _, exists := output["name"]; exists {
		t.Fatal("name should be removed")
	}
	if _, exists := output["owner"]; exists {
		t.Fatal("owner should be removed")
	}
	if _, exists := output["id"]; exists {
		t.Fatal("id should be removed")
	}

	projection := output["projection"].(map[string]any)
	if projection["type"] != "globe" {
		t.Fatalf("expected projection.type=globe, got %#v", projection["type"])
	}
	if _, exists := projection["name"]; exists {
		t.Fatal("projection.name should be removed")
	}

	sources := output["sources"].(map[string]any)
	composite := sources["composite"].(map[string]any)
	if _, exists := composite["name"]; exists {
		t.Fatal("source.name should be removed")
	}
}
