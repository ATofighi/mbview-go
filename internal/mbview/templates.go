package mbview

import (
	"embed"
	"encoding/json"
	"html/template"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var pageTemplates = template.Must(template.ParseFS(templateFiles,
	"templates/vector.tmpl",
	"templates/raster.tmpl",
))

type frontendSource struct {
	ID           string        `json:"id"`
	Format       string        `json:"format"`
	MaxZoom      int           `json:"maxzoom"`
	VectorLayers []VectorLayer `json:"vectorLayers,omitempty"`
}

type frontendConfig struct {
	Center          [2]float64       `json:"center"`
	Zoom            float64          `json:"zoom"`
	BasemapStyleURL string           `json:"basemapStyleURL"`
	Sources         []frontendSource `json:"sources"`
}

type pageData struct {
	ConfigJSON template.JS
}

func marshalFrontendConfig(config frontendConfig) (template.JS, error) {
	encoded, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return template.JS(encoded), nil
}
