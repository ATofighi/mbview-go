package mbview

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestLoadTilesetAndGetTile(t *testing.T) {
	mbtilesPath := createTestMBTiles(t)

	tilesets, err := LoadTilesets([]string{mbtilesPath})
	if err != nil {
		t.Fatalf("failed loading tileset: %v", err)
	}
	t.Cleanup(func() { closeTilesets(tilesets) })

	tileset := tilesets[0]
	if tileset.Format != "pbf" {
		t.Fatalf("expected pbf format, got %s", tileset.Format)
	}
	if tileset.MaxZoom != 14 {
		t.Fatalf("expected maxzoom 14, got %d", tileset.MaxZoom)
	}
	if tileset.Center.Lon != -122.42 || tileset.Center.Lat != 37.75 || tileset.Center.Zoom != 12 {
		t.Fatalf("unexpected center: %#v", tileset.Center)
	}
	if len(tileset.VectorLayers) != 2 {
		t.Fatalf("expected 2 vector layers, got %d", len(tileset.VectorLayers))
	}

	// XYZ y=0 maps to TMS row=1 at z=1.
	tile, err := tileset.GetTile(1, 1, 0)
	if err != nil {
		t.Fatalf("failed to get tile: %v", err)
	}
	if string(tile) != "hello-vector-tile" {
		t.Fatalf("unexpected tile contents: %q", string(tile))
	}

	_, err = tileset.GetTile(1, 1, 1)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestLoadTilesetsDedupeSourceIDs(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a", "roads.mbtiles")
	second := filepath.Join(root, "b", "roads.mbtiles")

	if err := os.MkdirAll(filepath.Dir(first), 0o755); err != nil {
		t.Fatalf("failed creating directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(second), 0o755); err != nil {
		t.Fatalf("failed creating directory: %v", err)
	}

	createTestMBTilesAt(t, first)
	createTestMBTilesAt(t, second)

	tilesets, err := LoadTilesets([]string{first, second})
	if err != nil {
		t.Fatalf("failed loading tilesets: %v", err)
	}
	t.Cleanup(func() { closeTilesets(tilesets) })

	if tilesets[0].ID != "roads" {
		t.Fatalf("unexpected first source id: %s", tilesets[0].ID)
	}
	if tilesets[1].ID != "roads_2" {
		t.Fatalf("unexpected second source id: %s", tilesets[1].ID)
	}
}

func createTestMBTiles(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.mbtiles")
	createTestMBTilesAt(t, path)
	return path
}

func createTestMBTilesAt(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("failed opening sqlite database: %v", err)
	}
	defer db.Close()

	statements := []string{
		"CREATE TABLE metadata (name TEXT, value TEXT)",
		"CREATE TABLE tiles (zoom_level INTEGER, tile_column INTEGER, tile_row INTEGER, tile_data BLOB)",
		"INSERT INTO metadata(name, value) VALUES ('name', 'Sample')",
		"INSERT INTO metadata(name, value) VALUES ('format', 'pbf')",
		"INSERT INTO metadata(name, value) VALUES ('minzoom', '0')",
		"INSERT INTO metadata(name, value) VALUES ('maxzoom', '14')",
		"INSERT INTO metadata(name, value) VALUES ('center', '-122.42,37.75,12')",
		"INSERT INTO metadata(name, value) VALUES ('bounds', '-123,37,-122,38')",
		"INSERT INTO metadata(name, value) VALUES ('json', '{\"vector_layers\":[{\"id\":\"roads\"},{\"id\":\"places\"}]}')",
		"INSERT INTO tiles(zoom_level, tile_column, tile_row, tile_data) VALUES (1, 1, 1, x'68656c6c6f2d766563746f722d74696c65')",
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("failed executing statement %q: %v", statement, err)
		}
	}
}
