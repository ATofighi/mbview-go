package mbview

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

var sourceSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

type VectorLayer struct {
	ID string `json:"id"`
}

type Tileset struct {
	ID           string        `json:"id"`
	Path         string        `json:"path"`
	Name         string        `json:"name"`
	Format       string        `json:"format"`
	MinZoom      int           `json:"minzoom"`
	MaxZoom      int           `json:"maxzoom"`
	Center       Center        `json:"center"`
	Bounds       [4]float64    `json:"bounds"`
	VectorLayers []VectorLayer `json:"vector_layers,omitempty"`

	db *sql.DB
}

func LoadTilesets(paths []string) ([]*Tileset, error) {
	tilesets := make([]*Tileset, 0, len(paths))
	usedSourceIDs := map[string]int{}

	for _, path := range paths {
		tileset, err := openTileset(path)
		if err != nil {
			closeTilesets(tilesets)
			return nil, err
		}

		tileset.ID = dedupeSourceID(tileset.ID, usedSourceIDs)
		tilesets = append(tilesets, tileset)
	}

	if len(tilesets) == 0 {
		return nil, errors.New("no MBTiles loaded")
	}

	primaryFormat := normalizeFormat(tilesets[0].Format)
	for _, ts := range tilesets[1:] {
		if normalizeFormat(ts.Format) != primaryFormat {
			closeTilesets(tilesets)
			return nil, fmt.Errorf("mixed tile formats are not supported: %s is %s, expected %s", ts.Path, ts.Format, tilesets[0].Format)
		}
	}

	return tilesets, nil
}

func closeTilesets(tilesets []*Tileset) {
	for _, tileset := range tilesets {
		_ = tileset.Close()
	}
}

func (t *Tileset) Close() error {
	if t == nil || t.db == nil {
		return nil
	}
	return t.db.Close()
}

func (t *Tileset) GetTile(z, x, y int) ([]byte, error) {
	if z < 0 || x < 0 || y < 0 {
		return nil, sql.ErrNoRows
	}

	tmsY := (1 << z) - 1 - y
	if tmsY < 0 {
		return nil, sql.ErrNoRows
	}

	var tile []byte
	err := t.db.QueryRow(
		"SELECT tile_data FROM tiles WHERE zoom_level = ? AND tile_column = ? AND tile_row = ? LIMIT 1",
		z,
		x,
		tmsY,
	).Scan(&tile)
	if err != nil {
		return nil, err
	}
	return tile, nil
}

func openTileset(path string) (*Tileset, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, expected .mbtiles file", path)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}

	metadata, err := readMetadata(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to read metadata from %s: %w", path, err)
	}

	format := normalizeFormat(metadata["format"])
	if format == "" {
		format = "pbf"
	}

	minzoom := parseInt(metadata["minzoom"], 0)
	maxzoom := parseInt(metadata["maxzoom"], minzoom)
	if maxzoom < minzoom {
		maxzoom = minzoom
	}

	center, hasCenter := parseCenterMetadata(metadata["center"])
	bounds, hasBounds := parseBoundsMetadata(metadata["bounds"])
	if !hasBounds {
		bounds = [4]float64{-180, -85, 180, 85}
	}
	if !hasCenter {
		center = Center{
			Lon:  (bounds[0] + bounds[2]) / 2,
			Lat:  (bounds[1] + bounds[3]) / 2,
			Zoom: float64(maxzoom),
		}
	}

	if center.Zoom == 0 {
		center.Zoom = float64(maxzoom)
	}

	vectorLayers := parseVectorLayers(metadata["json"])

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	tileset := &Tileset{
		ID:           sanitizeSourceID(base),
		Path:         path,
		Name:         metadata["name"],
		Format:       format,
		MinZoom:      minzoom,
		MaxZoom:      maxzoom,
		Center:       center,
		Bounds:       bounds,
		VectorLayers: vectorLayers,
		db:           db,
	}
	if tileset.Name == "" {
		tileset.Name = base
	}
	return tileset, nil
}

func readMetadata(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT name, value FROM metadata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := map[string]string{}
	for rows.Next() {
		var name string
		var value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		metadata[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metadata, nil
}

func parseVectorLayers(raw string) []VectorLayer {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var payload struct {
		VectorLayers []VectorLayer `json:"vector_layers"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}

	layers := make([]VectorLayer, 0, len(payload.VectorLayers))
	seen := map[string]struct{}{}
	for _, layer := range payload.VectorLayers {
		if strings.TrimSpace(layer.ID) == "" {
			continue
		}
		if _, ok := seen[layer.ID]; ok {
			continue
		}
		seen[layer.ID] = struct{}{}
		layers = append(layers, layer)
	}
	return layers
}

func parseCenterMetadata(raw string) (Center, bool) {
	parts := strings.Split(raw, ",")
	if len(parts) != 3 {
		return Center{}, false
	}

	values := make([]float64, 3)
	for i, part := range parts {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return Center{}, false
		}
		values[i] = parsed
	}

	return Center{Lon: values[0], Lat: values[1], Zoom: values[2]}, true
}

func parseBoundsMetadata(raw string) ([4]float64, bool) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return [4]float64{}, false
	}

	var values [4]float64
	for i, part := range parts {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return [4]float64{}, false
		}
		values[i] = parsed
	}
	return values, true
}

func sanitizeSourceID(raw string) string {
	cleaned := sourceSanitizer.ReplaceAllString(raw, "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return "source"
	}
	return cleaned
}

func dedupeSourceID(candidate string, used map[string]int) string {
	count, exists := used[candidate]
	if !exists {
		used[candidate] = 1
		return candidate
	}
	count++
	used[candidate] = count
	return fmt.Sprintf("%s_%d", candidate, count)
}

func parseInt(raw string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return parsed
}

func normalizeFormat(raw string) string {
	format := strings.ToLower(strings.TrimSpace(raw))
	switch format {
	case "mvt":
		return "pbf"
	case "jpg":
		return "jpeg"
	default:
		return format
	}
}
