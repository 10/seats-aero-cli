package cli

import (
	"net/url"
	"strings"

	"github.com/10/seats-aero-cli/internal/api"
)

type DestinationsCmd struct {
	OriginAirport      *string `xor:"airport" required:"" help:"Origin IATA code; choose exactly one airport flag."`
	DestinationAirport *string `xor:"airport" required:"" help:"Destination IATA code; choose exactly one airport flag."`
}

func (c *DestinationsCmd) Run(ctx *Context) error {
	q := url.Values{}
	if c.OriginAirport != nil {
		q.Set("origin_airport", strings.ToUpper(*c.OriginAirport))
	}
	if c.DestinationAirport != nil {
		q.Set("destination_airport", strings.ToUpper(*c.DestinationAirport))
	}
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/destinations", Query: q}, ctx.Stdout)
}
