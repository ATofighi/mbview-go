# mbview-go

`mbview-go` is a Go reimplementation of [`mapbox/mbview`](https://github.com/mapbox/mbview), designed for modern toolchains and distributed as standalone binaries.

The project keeps the local MBTiles inspection workflow from `mbview`, but replaces the legacy Node.js runtime with Go.

## Repository

```bash
git clone git@github.com:ATofighi/mbview-go.git
cd mbview-go
```

## Why this project exists

The original `mbview` is discontinued and no longer practical on newer Node.js versions. `mbview-go` keeps the same developer experience while remaining easy to install and maintain.

## Features

- `mbview`-style CLI workflow for one or more `.mbtiles` files.
- MapLibre frontend runtime for vector and raster preview.
- Multiple sources served from a single local server.
- Per-source toggle controls in the frontend menu when multiple MBTiles are loaded.
- Optional automatic browser opening.
- Basemap modes:
  - custom style JSON URL/file (`--basemap-style-url`)
  - Mapbox default basemap style (`--basemap`) with access token.
- Tagged GitHub releases with prebuilt binaries via GoReleaser.

## Install

### From source

```bash
go install github.com/ATofighi/mbview-go/cmd/mbview@latest
```

### From releases

Download the archive for your platform from GitHub Releases and run `mbview`.

## Usage

```bash
mbview [options] FILE1.mbtiles [FILE2.mbtiles ...]
```

### Default Mapbox basemap mode

If you do not provide `--basemap-style-url`, set a Mapbox public token:

```bash
export MAPBOX_ACCESS_TOKEN='pk.XXXX'
mbview --port 9000 ./roads.mbtiles ./places.mbtiles
```

### Custom basemap style mode

A custom style JSON URL or local file skips Mapbox token requirements:

```bash
mbview --basemap-style-url https://demotiles.maplibre.org/style.json ./roads.mbtiles
```

or

```bash
mbview --basemap-style-url ./style.json ./roads.mbtiles
```

## CLI options

- `--port` server port (default `3000`)
- `--host` bind host (default `localhost`)
- `--quiet`, `-q` suppress logs except startup URL
- `--no-open`, `-n` do not auto-open browser
- `--basemap`, `--base`, `--map` mapbox style name/id (default `dark`)
- `--basemap-style-url` custom style JSON URL or file path
- `--mapbox-access-token` mapbox token (or `MAPBOX_ACCESS_TOKEN` env)
- `--center` explicit center as `lon,lat,zoom`
- `--version`, `-v` print version
- `--help` print usage

## Notes

- Mixed MBTiles formats in one run are not supported (all inputs must match).
- Tile endpoints are served as `/{source}/{z}/{x}/{y}.{format}`.
- In the map UI, open `Menu` and use `Sources` checkboxes to show/hide each MBTiles source.
- Style rendering is powered by [MapLibre GL JS](https://maplibre.org/maplibre-gl-js/docs/).

## Development

```bash
go test ./...
```

## Release process

1. Push a semantic tag like `v0.1.0`.
2. GitHub Actions runs GoReleaser.
3. Release archives and checksums are published automatically.

## License

MIT
