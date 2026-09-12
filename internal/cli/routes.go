package cli

import (
	"net/url"

	"github.com/10/seats-aero-cli/internal/api"
)

type RoutesCmd struct {
	Source string `required:"" help:"One mileage program, e.g. united."`
}

func (c *RoutesCmd) Run(ctx *Context) error {
	return ctx.Client.Do(api.Request{Method: "GET", Path: "/routes", Query: url.Values{"source": {c.Source}}}, ctx.Stdout)
}
