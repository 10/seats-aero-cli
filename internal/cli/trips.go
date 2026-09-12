package cli

import (
	"net/url"

	"github.com/10/seats-aero-cli/internal/api"
)

type TripsCmd struct {
	AvailabilityID string `arg:"" help:"Availability row ID, sent as typed."`
	FilterFlags
}

func (c *TripsCmd) Run(ctx *Context) error {
	q := url.Values{}
	c.FilterFlags.addQuery(q)
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/trips/" + url.PathEscape(c.AvailabilityID), Query: q}, ctx.Stdout)
}
