# mbview-go

`mbview-go` is a Go reimplementation of [`mapbox/mbview`](https://github.com/mapbox/mbview), built for modern runtimes and distributed as standalone binaries.

## Motivation

The original `mbview` is discontinued and tied to older Node.js tooling. This project keeps the same local MBTiles viewing workflow, while using Go for easier distribution and maintenance.

## Project Status

Bootstrap phase: repository and project conventions are being initialized.

## Goals

- Match `mbview` usage and UX as closely as possible.
- Use MapLibre in the browser runtime.
- Support custom basemap style JSON URLs.
- Provide Mapbox default basemap support when a token is supplied.
- Publish reproducible binaries via GitHub Actions releases.

## Planned CLI shape

```bash
mbview-go [options] FILE1.mbtiles [FILE2.mbtiles ...]
```

## License

MIT
