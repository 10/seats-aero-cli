package cli

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/10/seats-aero-cli/internal/api"
)

type HistoryCmd struct {
	RouteID   string  `arg:"" help:"Route ID from routes --source PROGRAM (not an availability ID)."`
	Kind      string  `default:"availability" enum:"availability,price" help:"History type: availability or award mileage price."`
	StartDate string  `required:"" help:"First observation date, YYYY-MM-DD (not a flight departure date)."`
	EndDate   string  `required:"" help:"Last observation date, YYYY-MM-DD."`
	Cabins    *string `help:"Website cabin filter: comma-separated Y,W,J,F; omit for all cabins."`
	Stops     string  `default:"direct" help:"Website stops filter (default direct)."`
	Metric    *string `help:"Website aggregation metric; defaults to sum for availability, max for price."`
}

func (c *HistoryCmd) Run(ctx *Context) error {
	path, metric := "/daily_route_availability/", "sum"
	if c.Kind == "price" {
		path, metric = "/daily_price_availability/", "max"
	}
	q := url.Values{"start": {c.StartDate}, "end": {c.EndDate}, "stops": {c.Stops}, "metric": {metric}}
	setString(q, "metric", c.Metric)
	if c.Cabins != nil {
		q.Set("cabins", strings.ToUpper(*c.Cabins))
	}
	var body bytes.Buffer
	if err := ctx.Client.Do(api.Request{Method: "GET", Path: path + url.PathEscape(c.RouteID), Query: q, Website: true}, &body); err != nil {
		return err
	}
	var rows []struct{ ObservationDay string }
	if err := json.Unmarshal(body.Bytes(), &rows); err != nil || rows == nil {
		return &api.Error{Code: "invalid_response", Message: "website API did not return a history array"}
	}
	for _, row := range rows {
		if row.ObservationDay == "" {
			return &api.Error{Code: "invalid_response", Message: "website history row is missing ObservationDay"}
		}
	}
	_, err := ctx.Stdout.Write(body.Bytes())
	return err
}
