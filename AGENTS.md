# AGENTS

This file describes contribution conventions for human and automated contributors.

## Scope

- Build and maintain a Go replacement for `mapbox/mbview`.
- Keep behavior close to upstream unless explicitly documented otherwise.

## Engineering Rules

- Prefer small, reviewable commits.
- Keep changes portable across Linux, macOS, and Windows.
- Avoid CGO by default so release binaries are easy to ship.
- Add tests for parsing, routing, and metadata logic.

## Runtime Requirements

- Frontend map runtime must be MapLibre.
- Basemap behavior:
  - Accept custom style JSON URL input.
  - Otherwise use a Mapbox default style and require `MAPBOX_ACCESS_TOKEN`.

## Release Requirements

- Use GitHub Actions for CI.
- Build and publish tagged release binaries.

