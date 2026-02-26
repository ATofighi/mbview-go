package mbview

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pkg/browser"
)

type runtimeConfig struct {
	Host         string
	Port         int
	Format       string
	IsVector     bool
	Center       Center
	BasemapStyle Basemap
	SourceOrder  []string
	Sources      map[string]*Tileset
}

func Serve(opts Options) error {
	tilesets, err := LoadTilesets(opts.Files)
	if err != nil {
		return err
	}
	defer closeTilesets(tilesets)

	basemap, err := ResolveBasemap(opts)
	if err != nil {
		return err
	}

	config := buildRuntimeConfig(opts, basemap, tilesets)
	router := newRouter(config)

	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	publicURL := openableURL(opts.Host, opts.Port)
	fmt.Printf("Listening on %s\n", publicURL)

	if !opts.NoOpen {
		if err := browser.OpenURL(publicURL); err != nil && !opts.Quiet {
			log.Printf("failed to open browser: %v", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	case <-sigCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shut down server: %w", err)
		}
		return nil
	}
}

func buildRuntimeConfig(opts Options, basemap Basemap, tilesets []*Tileset) runtimeConfig {
	sources := make(map[string]*Tileset, len(tilesets))
	order := make([]string, 0, len(tilesets))

	for _, tileset := range tilesets {
		sources[tileset.ID] = tileset
		order = append(order, tileset.ID)
	}

	center := tilesets[0].Center
	if opts.CenterOverride != nil {
		center = *opts.CenterOverride
	}

	format := normalizeFormat(tilesets[0].Format)
	return runtimeConfig{
		Host:         opts.Host,
		Port:         opts.Port,
		Format:       format,
		IsVector:     format == "pbf",
		Center:       center,
		BasemapStyle: basemap,
		SourceOrder:  order,
		Sources:      sources,
	}
}

func newRouter(config runtimeConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		if config.IsVector {
			renderVectorPage(w, config)
			return
		}
		renderRasterPage(w, config)
	})

	r.Get("/config", func(w http.ResponseWriter, req *http.Request) {
		type sourceConfig struct {
			ID      string `json:"id"`
			Format  string `json:"format"`
			MaxZoom int    `json:"maxzoom"`
		}
		response := struct {
			Host            string         `json:"host"`
			Port            int            `json:"port"`
			Center          Center         `json:"center"`
			BasemapStyleURL string         `json:"basemapStyleURL"`
			Sources         []sourceConfig `json:"sources"`
		}{
			Host:            config.Host,
			Port:            config.Port,
			Center:          config.Center,
			BasemapStyleURL: config.BasemapStyle.StyleURL,
			Sources:         make([]sourceConfig, 0, len(config.SourceOrder)),
		}

		for _, id := range config.SourceOrder {
			source := config.Sources[id]
			response.Sources = append(response.Sources, sourceConfig{ID: id, Format: source.Format, MaxZoom: source.MaxZoom})
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, response)
	})

	if len(config.BasemapStyle.StyleJSON) > 0 {
		r.Get("/basemap/style.json", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(config.BasemapStyle.StyleJSON)
		})
	}

	r.Get("/{source}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{ext}", func(w http.ResponseWriter, req *http.Request) {
		sourceID := chi.URLParam(req, "source")
		tileset, ok := config.Sources[sourceID]
		if !ok {
			http.NotFound(w, req)
			return
		}

		ext := normalizeFormat(chi.URLParam(req, "ext"))
		if ext != normalizeFormat(tileset.Format) {
			http.NotFound(w, req)
			return
		}

		z, _ := strconv.Atoi(chi.URLParam(req, "z"))
		x, _ := strconv.Atoi(chi.URLParam(req, "x"))
		y, _ := strconv.Atoi(chi.URLParam(req, "y"))

		tile, err := tileset.GetTile(z, x, y)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, "failed to read tile", http.StatusInternalServerError)
			return
		}

		if isGzip(tile) {
			w.Header().Set("Content-Encoding", "gzip")
		}
		w.Header().Set("Content-Type", contentTypeForFormat(ext))
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(tile)
	})

	return r
}

func renderVectorPage(w io.Writer, config runtimeConfig) {
	sources := make([]frontendSource, 0, len(config.SourceOrder))
	for _, id := range config.SourceOrder {
		ts := config.Sources[id]
		sources = append(sources, frontendSource{
			ID:           ts.ID,
			Format:       ts.Format,
			MaxZoom:      ts.MaxZoom,
			VectorLayers: ts.VectorLayers,
		})
	}

	frontend := frontendConfig{
		Center:          [2]float64{config.Center.Lon, config.Center.Lat},
		Zoom:            config.Center.Zoom,
		BasemapStyleURL: config.BasemapStyle.StyleURL,
		Sources:         sources,
	}
	configJSON, err := marshalFrontendConfig(frontend)
	if err != nil {
		_, _ = io.WriteString(w, "failed to render page")
		return
	}

	_ = pageTemplates.ExecuteTemplate(w, "vector.tmpl", pageData{ConfigJSON: configJSON})
}

func renderRasterPage(w io.Writer, config runtimeConfig) {
	sources := make([]frontendSource, 0, len(config.SourceOrder))
	for _, id := range config.SourceOrder {
		ts := config.Sources[id]
		sources = append(sources, frontendSource{
			ID:      ts.ID,
			Format:  ts.Format,
			MaxZoom: ts.MaxZoom,
		})
	}

	frontend := frontendConfig{
		Center:          [2]float64{config.Center.Lon, config.Center.Lat},
		Zoom:            config.Center.Zoom,
		BasemapStyleURL: config.BasemapStyle.StyleURL,
		Sources:         sources,
	}
	configJSON, err := marshalFrontendConfig(frontend)
	if err != nil {
		_, _ = io.WriteString(w, "failed to render page")
		return
	}

	_ = pageTemplates.ExecuteTemplate(w, "raster.tmpl", pageData{ConfigJSON: configJSON})
}

func writeJSON(w io.Writer, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		_, _ = io.WriteString(w, "{}")
		return
	}
	_, _ = w.Write(encoded)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func openableURL(host string, port int) string {
	openHost := strings.TrimSpace(host)
	switch openHost {
	case "", "0.0.0.0", "::", "[::]":
		openHost = "localhost"
	}
	return fmt.Sprintf("http://%s:%d", openHost, port)
}

func contentTypeForFormat(format string) string {
	switch normalizeFormat(format) {
	case "pbf":
		return "application/x-protobuf"
	case "png":
		return "image/png"
	case "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func isGzip(data []byte) bool {
	return len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b
}
