package cli

import (
	"io"
	"net/url"
	"os"
	"strconv"

	"github.com/10/seats-aero-cli/internal/api"
	"github.com/10/seats-aero-cli/internal/config"
	"github.com/alecthomas/kong"
)

type Root struct {
	APIKey       string           `name:"api-key" help:"Pro API key (otherwise SEATSAERO_API_KEY, then config file)."`
	Version      kong.VersionFlag `help:"Print the version."`
	Search       SearchCmd        `cmd:"" help:"Search cached award availability for a route."`
	Availability AvailabilityCmd  `cmd:"" help:"Scan cached availability for one mileage program."`
	Trips        TripsCmd         `cmd:"" help:"Get itineraries and booking links for an availability ID."`
	Routes       RoutesCmd        `cmd:"" help:"List routes tracked by one mileage program."`
	History      HistoryCmd       `cmd:"" help:"Get daily route history (experimental website API)."`
	Destinations DestinationsCmd  `cmd:"" help:"Discover nonstop airports and cheapest raw mileage."`
	Refresh      RefreshCmd       `cmd:"" help:"Queue or check a refresh once; repeat the same IDs to poll (queued IDs spend credits)."`
	Alerts       AlertsCmd        `cmd:"" help:"List alerts already configured on the account."`
	Rooms        RoomsCmd         `cmd:"" help:"Search rooms.aero hotel awards with the same Pro key."`
	Auth         AuthCmd          `cmd:"" help:"Save a Pro API key in config (the argument is visible in shell history)."`
}

type Context struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Client     *api.Client
	ConfigPath string
}

func (r *Root) ResolveKey() (string, error) {
	if r.APIKey != "" {
		return r.APIKey, nil
	}
	if key := os.Getenv("SEATSAERO_API_KEY"); key != "" {
		return key, nil
	}
	path, err := config.Path()
	if err != nil {
		return "", &api.Error{Code: "config_error", Message: "could not locate config file"}
	}
	key, err := config.Read(path)
	if err != nil {
		return "", &api.Error{Code: "config_error", Message: "could not read config file; check its JSON and permissions"}
	}
	if key == "" {
		return "", &api.Error{Code: "missing_api_key", Message: "provide --api-key, SEATSAERO_API_KEY, or run seatsaero auth KEY"}
	}
	return key, nil
}

// Pointers distinguish an omitted parameter from an explicitly supplied zero or empty string.
func setString(q url.Values, name string, value *string) {
	if value != nil {
		q.Set(name, *value)
	}
}

func setInt(q url.Values, name string, value *int64) {
	if value != nil {
		q.Set(name, strconv.FormatInt(*value, 10))
	}
}

func setFloat(q url.Values, name string, value *float64) {
	if value != nil {
		q.Set(name, strconv.FormatFloat(*value, 'f', -1, 64))
	}
}

func setBool(q url.Values, name string, value bool) {
	if value {
		q.Set(name, "true")
	}
}

type PageFlags struct {
	Take     int64  `default:"50" help:"Rows per page (API range 10–1000); raise deliberately as rows are large."`
	Skip     *int64 `help:"Number of rows already retrieved."`
	Cursor   *int64 `help:"Opaque cursor integer from the first response."`
	All      bool   `help:"Collect and deduplicate pages within a request budget; check hasMore in the result."`
	MaxPages *int   `help:"Maximum requests with --all (default 10); requires --all."`
}

func (p *PageFlags) query() url.Values {
	q := url.Values{"take": {strconv.FormatInt(p.Take, 10)}}
	setInt(q, "skip", p.Skip)
	setInt(q, "cursor", p.Cursor)
	return q
}

type FilterFlags struct {
	IncludeFiltered bool   `help:"Include dynamically priced rows normally hidden by the API."`
	MinCabinPct     *int64 `help:"Minimum distance percentage in the reported cabin (0–100; API default 100)."`
}

func (f *FilterFlags) addQuery(q url.Values) {
	setBool(q, "include_filtered", f.IncludeFiltered)
	setInt(q, "min_cabin_pct", f.MinCabinPct)
}
