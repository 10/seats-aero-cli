package cli

import (
	"net/url"
	"strconv"

	"github.com/10/seats-aero-cli/internal/api"
)

type RoomsCmd struct {
	Search       RoomsSearchCmd       `cmd:"" help:"Find hotel awards in an area and check-in date range."`
	Availability RoomsAvailabilityCmd `cmd:"" help:"Get stay dates and rates for programs or hotel IDs."`
	Details      RoomsDetailsCmd      `cmd:"" help:"Get room types, prices and a booking link for an availability ID."`
	Hotels       RoomsHotelsCmd       `cmd:"" help:"Browse the tracked hotel directory."`
	Refresh      RoomsRefreshCmd      `cmd:"" help:"Request a hotel refresh once (one per user per 24 hours, shared with the website)."`
	Alerts       RoomsAlertsCmd       `cmd:"" help:"List existing hotel alerts."`
}

// Rooms lists use offset pagination, not the flight API's cursor protocol.
type RoomsPageFlags struct {
	Take int64  `default:"50" help:"Rows per page (1–500; hotels allows up to 1000)."`
	Skip *int64 `help:"Rows to skip; add take for the next page while has_more is true."`
}

func (p *RoomsPageFlags) query() url.Values {
	q := url.Values{"take": {strconv.FormatInt(p.Take, 10)}}
	setInt(q, "skip", p.Skip)
	return q
}

type RoomsSearchCmd struct {
	StartDate string   `required:"" help:"Earliest check-in date, YYYY-MM-DD."`
	EndDate   string   `required:"" help:"Latest check-in date, not check-out; use --nights for stay length."`
	Location  *string  `help:"Place to search, e.g. Tokyo; alternatively supply all four bounding-box coordinates."`
	SwLat     *float64 `help:"South-west latitude of the bounding box."`
	SwLng     *float64 `help:"South-west longitude of the bounding box."`
	NeLat     *float64 `help:"North-east latitude of the bounding box."`
	NeLng     *float64 `help:"North-east longitude of the bounding box."`
	Nights    *int64   `help:"Stay length, 1–5 nights; omit for all lengths."`
	Source    *string  `help:"Comma-separated programs, e.g. hyatt,hilton."`
	MaxPoints *int64   `help:"Maximum award points for the stay."`
	MinCPP    *float64 `name:"min-cpp" help:"Minimum value in US cents per point."`
	RoomsPageFlags
}

func (c *RoomsSearchCmd) Run(ctx *Context) error {
	q := c.RoomsPageFlags.query()
	q.Set("start_date", c.StartDate)
	q.Set("end_date", c.EndDate)
	setString(q, "location", c.Location)
	setFloat(q, "sw_lat", c.SwLat)
	setFloat(q, "sw_lng", c.SwLng)
	setFloat(q, "ne_lat", c.NeLat)
	setFloat(q, "ne_lng", c.NeLng)
	setInt(q, "nights", c.Nights)
	setString(q, "source", c.Source)
	setInt(q, "max_points", c.MaxPoints)
	setFloat(q, "min_cpp", c.MinCPP)
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/search", Query: q}, ctx.Stdout)
}

type RoomsAvailabilityCmd struct {
	Source       *string  `help:"Comma-separated programs; required unless --hotel-id is given."`
	HotelID      *string  `help:"Comma-separated hotel IDs (up to 50); required unless --source is given."`
	StartDate    *string  `help:"Earliest check-in date, YYYY-MM-DD."`
	EndDate      *string  `help:"Latest check-in date, not check-out."`
	Nights       *int64   `help:"Stay length, 1–5 nights; omit for all lengths."`
	Brand        *string  `help:"Comma-separated program brand codes."`
	Category     *string  `help:"Comma-separated award categories."`
	Country      *string  `help:"Comma-separated country names, e.g. Japan."`
	RateType     *string  `help:"Rate type: award, cash, or any (API default award)."`
	RoomType     *string  `help:"Room class: standard, suite, or any (API default any)."`
	MaxPoints    *int64   `help:"Maximum award points for the stay."`
	MinCPP       *float64 `name:"min-cpp" help:"Minimum value in US cents per point."`
	OrderBy      *string  `help:"Order: arrival_date, lowest_award, lowest_cash, or last_checked."`
	IncludeHotel *bool    `help:"Embed the hotel and booking link (API default true); use --include-hotel=false for smaller output."`
	RoomsPageFlags
}

func (c *RoomsAvailabilityCmd) Run(ctx *Context) error {
	q := c.RoomsPageFlags.query()
	setString(q, "source", c.Source)
	setString(q, "hotel_id", c.HotelID)
	setString(q, "start_date", c.StartDate)
	setString(q, "end_date", c.EndDate)
	setInt(q, "nights", c.Nights)
	setString(q, "brand", c.Brand)
	setString(q, "category", c.Category)
	setString(q, "country", c.Country)
	setString(q, "rate_type", c.RateType)
	setString(q, "room_type", c.RoomType)
	setInt(q, "max_points", c.MaxPoints)
	setFloat(q, "min_cpp", c.MinCPP)
	setString(q, "order_by", c.OrderBy)
	if c.IncludeHotel != nil {
		q.Set("include_hotel", strconv.FormatBool(*c.IncludeHotel))
	}
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/availability", Query: q}, ctx.Stdout)
}

type RoomsDetailsCmd struct {
	AvailabilityID string `arg:"" help:"Availability ID from rooms availability (not a hotel ID)."`
}

func (c *RoomsDetailsCmd) Run(ctx *Context) error {
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/availability/" + url.PathEscape(c.AvailabilityID)}, ctx.Stdout)
}

type RoomsHotelsCmd struct {
	ID         *string `help:"Comma-separated rooms.aero hotel IDs (up to 100)."`
	InternalID *string `help:"Comma-separated program property codes; combine with --source."`
	Source     *string `help:"Comma-separated programs, e.g. hyatt,hilton."`
	Country    *string `help:"Comma-separated country names, e.g. United States,Japan."`
	State      *string `help:"Comma-separated state or region names."`
	City       *string `help:"Comma-separated city names."`
	Brand      *string `help:"Comma-separated program brand codes."`
	Category   *string `help:"Comma-separated award categories."`
	Search     *string `help:"Case-insensitive substring of the hotel name."`
	RoomsPageFlags
}

func (c *RoomsHotelsCmd) Run(ctx *Context) error {
	q := c.RoomsPageFlags.query()
	setString(q, "id", c.ID)
	setString(q, "internal_id", c.InternalID)
	setString(q, "source", c.Source)
	setString(q, "country", c.Country)
	setString(q, "state", c.State)
	setString(q, "city", c.City)
	setString(q, "brand", c.Brand)
	setString(q, "category", c.Category)
	setString(q, "search", c.Search)
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/hotels", Query: q}, ctx.Stdout)
}

type RoomsRefreshCmd struct {
	HotelID string `arg:"" help:"Hotel ID from rooms hotels or search."`
}

func (c *RoomsRefreshCmd) Run(ctx *Context) error {
	return ctx.Client.Do(api.Request{Method: "POST", Path: "/hotels/" + url.PathEscape(c.HotelID) + "/refresh"}, ctx.Stdout)
}

type RoomsAlertsCmd struct {
	IncludeExpired bool `help:"Include expired one-off alerts."`
	RoomsPageFlags
}

func (c *RoomsAlertsCmd) Run(ctx *Context) error {
	q := c.RoomsPageFlags.query()
	setBool(q, "include_expired", c.IncludeExpired)
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/alerts", Query: q}, ctx.Stdout)
}
