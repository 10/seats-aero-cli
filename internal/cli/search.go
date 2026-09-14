package cli

import (
	"strings"

	"github.com/10/seats-aero-cli/internal/api"
)

type SearchCmd struct {
	Origin            string  `arg:"" help:"Origin IATA airport code or comma-separated codes."`
	Destination       string  `arg:"" help:"Destination IATA airport code or comma-separated codes."`
	StartDate         *string `help:"Depart on or after YYYY-MM-DD."`
	EndDate           *string `help:"Depart on or before YYYY-MM-DD."`
	Sources           *string `help:"Comma-separated mileage programs, e.g. aeroplan,united."`
	Cabins            *string `help:"Comma-separated cabins: economy,premium,business,first."`
	Carriers          *string `help:"Comma-separated IATA airline codes (uppercased)."`
	OnlyDirectFlights bool    `help:"Only nonstop flights."`
	IncludeTrips      bool    `help:"Embed AvailabilityTrips (slower and larger)."`
	MinifyTrips       bool    `help:"Reduce embedded trip fields; combine with --include-trips."`
	OrderBy           *string `help:"API ordering, e.g. lowest_mileage; omit for date and premium cabins first."`
	PageFlags
	FilterFlags
}

func (c *SearchCmd) Run(ctx *Context) error {
	q := c.PageFlags.query()
	q.Set("origin_airport", strings.ToUpper(c.Origin))
	q.Set("destination_airport", strings.ToUpper(c.Destination))
	setString(q, "start_date", c.StartDate)
	setString(q, "end_date", c.EndDate)
	setString(q, "sources", c.Sources)
	setString(q, "cabins", c.Cabins)
	if c.Carriers != nil {
		q.Set("carriers", strings.ToUpper(*c.Carriers))
	}
	setString(q, "order_by", c.OrderBy)
	setBool(q, "only_direct_flights", c.OnlyDirectFlights)
	setBool(q, "include_trips", c.IncludeTrips)
	setBool(q, "minify_trips", c.MinifyTrips)
	c.FilterFlags.addQuery(q)
	return c.PageFlags.run(ctx, api.Request{Method: "GET", Path: "/search", Query: q})
}
