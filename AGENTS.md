# AGENTS

This file defines contribution guidelines for human and automated contributors.

## Scope

- Maintain `mbview-go` as a Go replacement for `mapbox/mbview`.
- Keep behavior close to upstream unless a divergence is documented in `README.md`.

## Core requirements

- Browser map runtime must be MapLibre.
- MBTiles must be served locally with predictable source IDs.
- Basemap behavior must support:
  - custom style JSON URL/file
  - Mapbox default style with access token fallback

## Engineering standards

- Keep commits small and reviewable.
- Add tests for parsing, metadata, and routing logic.
- Preserve cross-platform compatibility for release binaries.
- Avoid breaking CLI compatibility without documenting migration notes.

## Release standards

- CI (`.github/workflows/ci.yml`) must stay green.
- Tagged releases must produce binaries with GoReleaser.
- Version should be injected at build time through ldflags.

## Security and reliability

- Never log access tokens.
- Validate and sanitize user-provided file paths and URLs.
- Handle malformed MBTiles metadata gracefully.
