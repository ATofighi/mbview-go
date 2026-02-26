package mbview

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// Version is stamped at build time with -ldflags.
var Version = "dev"

const defaultUsage = `usage: mbview-go [options] [files]

 --port sets port to use (default: 3000)
 --host sets host to use (default: localhost)
 --quiet or -q suppress all logging except the address to visit
 -n don't automatically open the browser on start
 --basemap, --base or --map sets the default mapbox basemap style (default: dark)
 --basemap-style-url points to a style JSON URL or local style JSON file
 --mapbox-access-token sets mapbox token (env MAPBOX_ACCESS_TOKEN also supported)
 --center sets lon,lat,zoom (e.g. --center -122.42,37.75,12)
 --version returns module version
 --help prints this message
`

type Center struct {
	Lon  float64 `json:"lon"`
	Lat  float64 `json:"lat"`
	Zoom float64 `json:"zoom"`
}

type Options struct {
	Host              string
	Port              int
	Quiet             bool
	NoOpen            bool
	Basemap           string
	BasemapStyleURL   string
	MapboxAccessToken string
	Files             []string
	CenterOverride    *Center
	ShowHelp          bool
	ShowVersion       bool
}

func ParseOptions(args []string, getenv func(string) string) (Options, error) {
	var opts Options

	flags := pflag.NewFlagSet("mbview-go", pflag.ContinueOnError)
	flags.SortFlags = false
	flags.Usage = func() {}

	var center string
	var baseAlias string
	var mapAlias string
	var tokenFlag string

	flags.IntVar(&opts.Port, "port", 3000, "port")
	flags.StringVar(&opts.Host, "host", "localhost", "host")
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "quiet")
	flags.BoolVarP(&opts.NoOpen, "no-open", "n", false, "do not open browser")
	flags.StringVar(&opts.Basemap, "basemap", "dark", "mapbox basemap style")
	flags.StringVar(&baseAlias, "base", "", "alias for --basemap")
	flags.StringVar(&mapAlias, "map", "", "alias for --basemap")
	flags.StringVar(&opts.BasemapStyleURL, "basemap-style-url", "", "custom style JSON URL or local file")
	flags.StringVar(&tokenFlag, "mapbox-access-token", "", "mapbox token")
	flags.StringVar(&center, "center", "", "center lon,lat,zoom")
	flags.BoolVarP(&opts.ShowVersion, "version", "v", false, "version")
	flags.Bool("help", false, "help")

	if err := flags.Parse(args); err != nil {
		return Options{}, fmt.Errorf("%w\n\n%s", err, defaultUsage)
	}

	opts.ShowHelp, _ = flags.GetBool("help")
	if opts.ShowHelp {
		return opts, nil
	}

	if mapAlias != "" {
		opts.Basemap = mapAlias
	} else if baseAlias != "" {
		opts.Basemap = baseAlias
	}

	if center != "" {
		parsedCenter, err := parseCenter(center)
		if err != nil {
			return Options{}, fmt.Errorf("invalid --center value: %w", err)
		}
		opts.CenterOverride = &parsedCenter
	}

	opts.MapboxAccessToken = firstNonEmpty(
		tokenFlag,
		getenv("MAPBOX_ACCESS_TOKEN"),
		getenv("MapboxAccessToken"),
	)

	files := flags.Args()
	if opts.ShowVersion || opts.ShowHelp {
		return opts, nil
	}
	if len(files) == 0 {
		return Options{}, fmt.Errorf("missing MBTiles input file(s)\n\n%s", defaultUsage)
	}

	for _, file := range files {
		if ext := strings.ToLower(filepath.Ext(file)); ext != ".mbtiles" {
			return Options{}, fmt.Errorf("unsupported file extension %q for %s; expected .mbtiles", ext, file)
		}
	}

	if opts.BasemapStyleURL == "" && opts.MapboxAccessToken == "" {
		return Options{}, fmt.Errorf("missing mapbox access token, set MAPBOX_ACCESS_TOKEN or pass --mapbox-access-token")
	}

	opts.Files = files
	return opts, nil
}

func HelpText() string {
	return defaultUsage
}

func parseCenter(raw string) (Center, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 3 {
		return Center{}, fmt.Errorf("expected lon,lat,zoom")
	}

	vals := make([]float64, 3)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return Center{}, err
		}
		vals[i] = v
	}

	return Center{Lon: vals[0], Lat: vals[1], Zoom: vals[2]}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
