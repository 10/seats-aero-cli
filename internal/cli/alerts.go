package cli

import "github.com/10/seats-aero-cli/internal/api"

type AlertsCmd struct{}

func (c *AlertsCmd) Run(ctx *Context) error {
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/alerts"}, ctx.Stdout)
}
