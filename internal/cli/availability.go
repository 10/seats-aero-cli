package cli

import "github.com/10/seats-aero-cli/internal/api"

type AvailabilityCmd struct {
	Source            string  `required:"" help:"One mileage program, e.g. aeroplan."`
	Cabin             *string `help:"One cabin: economy, premium, business, or first."`
	StartDate         *string `help:"Depart on or after YYYY-MM-DD."`
	EndDate           *string `help:"Depart on or before YYYY-MM-DD."`
	OriginRegion      *string `help:"Origin region: North America, South America, Africa, Asia, Europe, or Oceania."`
	DestinationRegion *string `help:"Destination region; quote multi-word names, e.g. 'North America'."`
	PageFlags
	FilterFlags
}

func (c *AvailabilityCmd) Run(ctx *Context) error {
	q := c.PageFlags.query()
	q.Set("source", c.Source)
	setString(q, "cabin", c.Cabin)
	setString(q, "start_date", c.StartDate)
	setString(q, "end_date", c.EndDate)
	setString(q, "origin_region", c.OriginRegion)
	setString(q, "destination_region", c.DestinationRegion)
	c.FilterFlags.addQuery(q)
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/availability", Query: q}, ctx.Stdout)
}
