package mbview

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Basemap struct {
	StyleURL  string
	StyleJSON []byte
}

func ResolveBasemap(opts Options) (Basemap, error) {
	if strings.TrimSpace(opts.BasemapStyleURL) != "" {
		return resolveCustomBasemap(opts.BasemapStyleURL)
	}

	owner, styleID := normalizeMapboxStyle(opts.Basemap)
	styleJSON, err := fetchAndRewriteMapboxStyle(owner, styleID, opts.MapboxAccessToken)
	if err != nil {
		return Basemap{}, err
	}

	return Basemap{
		StyleURL:  "/basemap/style.json",
		StyleJSON: styleJSON,
	}, nil
}

func resolveCustomBasemap(raw string) (Basemap, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Basemap{}, fmt.Errorf("basemap style URL is empty")
	}

	if isHTTPURL(trimmed) {
		return Basemap{StyleURL: trimmed}, nil
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return Basemap{}, fmt.Errorf("invalid basemap style file path %q: %w", trimmed, err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Basemap{}, fmt.Errorf("failed to read basemap style file %s: %w", absPath, err)
	}
	if !json.Valid(data) {
		return Basemap{}, fmt.Errorf("basemap style file %s is not valid JSON", absPath)
	}
	return Basemap{StyleURL: "/basemap/style.json", StyleJSON: data}, nil
}

func fetchAndRewriteMapboxStyle(owner, styleID, token string) ([]byte, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("missing mapbox access token")
	}

	styleEndpoint := fmt.Sprintf(
		"https://api.mapbox.com/styles/v1/%s/%s?access_token=%s&secure=true",
		url.PathEscape(owner),
		url.PathEscape(styleID),
		url.QueryEscape(token),
	)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(styleEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mapbox style %s/%s: %w", owner, styleID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapbox style response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mapbox style request failed: %s", strings.TrimSpace(string(body)))
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid style JSON returned by mapbox: %w", err)
	}

	rewritten := rewriteMapboxReferences(payload, token)
	encoded, err := json.Marshal(rewritten)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rewritten style JSON: %w", err)
	}
	return encoded, nil
}

func rewriteMapboxReferences(value any, token string) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			typed[key] = rewriteMapboxReferences(nested, token)
		}
		return typed
	case []any:
		for i, nested := range typed {
			typed[i] = rewriteMapboxReferences(nested, token)
		}
		return typed
	case string:
		return rewriteStyleReference(typed, token)
	default:
		return value
	}
}

func rewriteStyleReference(raw, token string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}

	switch {
	case strings.HasPrefix(raw, "mapbox://styles/"):
		path := strings.TrimPrefix(raw, "mapbox://styles/")
		return withToken("https://api.mapbox.com/styles/v1/"+path, token)
	case strings.HasPrefix(raw, "mapbox://sprites/"):
		path := strings.TrimPrefix(raw, "mapbox://sprites/")
		return withToken("https://api.mapbox.com/styles/v1/"+path+"/sprite", token)
	case strings.HasPrefix(raw, "mapbox://fonts/"):
		path := strings.TrimPrefix(raw, "mapbox://fonts/")
		return withToken("https://api.mapbox.com/fonts/v1/"+path, token)
	case strings.HasPrefix(raw, "mapbox://"):
		path := strings.TrimPrefix(raw, "mapbox://")
		return withToken("https://api.mapbox.com/v4/"+path+".json?secure=true", token)
	default:
		if strings.Contains(raw, "api.mapbox.com") {
			return withToken(raw, token)
		}
		return raw
	}
}

func withToken(rawURL, token string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsed.Query()
	if query.Get("access_token") == "" {
		query.Set("access_token", token)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func normalizeMapboxStyle(raw string) (owner string, styleID string) {
	style := strings.TrimSpace(raw)
	if style == "" {
		style = "dark"
	}

	aliases := map[string]string{
		"dark":              "dark-v11",
		"light":             "light-v11",
		"streets":           "streets-v12",
		"outdoors":          "outdoors-v12",
		"satellite":         "satellite-v9",
		"satellite-streets": "satellite-streets-v12",
		"navigation-day":    "navigation-day-v1",
		"navigation-night":  "navigation-night-v1",
	}

	if strings.HasPrefix(style, "mapbox://styles/") {
		style = strings.TrimPrefix(style, "mapbox://styles/")
	}

	if mapped, ok := aliases[style]; ok {
		return "mapbox", mapped
	}

	if strings.Contains(style, "/") {
		parts := strings.SplitN(style, "/", 2)
		return parts[0], parts[1]
	}

	return "mapbox", style
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
