package mbview

import (
	"strings"
	"testing"
)

func TestParseOptionsRequiresTokenWhenUsingDefaultBasemap(t *testing.T) {
	_, err := ParseOptions([]string{"sample.mbtiles"}, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if !strings.Contains(err.Error(), "missing mapbox access token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOptionsAllowsCustomStyleWithoutToken(t *testing.T) {
	opts, err := ParseOptions([]string{"--basemap-style-url", "https://example.com/style.json", "sample.mbtiles"}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.BasemapStyleURL != "https://example.com/style.json" {
		t.Fatalf("unexpected style url: %s", opts.BasemapStyleURL)
	}
	if opts.MapboxAccessToken != "" {
		t.Fatalf("expected empty token, got %q", opts.MapboxAccessToken)
	}
}

func TestParseOptionsUsesAliasAndCenter(t *testing.T) {
	opts, err := ParseOptions(
		[]string{"--map", "light", "--center", "-122.42,37.75,12", "--mapbox-access-token", "token", "sample.mbtiles"},
		func(string) string { return "" },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Basemap != "light" {
		t.Fatalf("expected basemap light, got %s", opts.Basemap)
	}
	if opts.CenterOverride == nil {
		t.Fatal("expected center override")
	}
	if opts.CenterOverride.Lon != -122.42 || opts.CenterOverride.Lat != 37.75 || opts.CenterOverride.Zoom != 12 {
		t.Fatalf("unexpected center override: %#v", *opts.CenterOverride)
	}
}
