package cli

import (
	"bytes"
	"encoding/json"

	"github.com/10/seats-aero-cli/internal/api"
)

type RefreshCmd struct {
	AvailabilityID []string `arg:"" help:"One or more availability IDs, separated by spaces."`
}

func (c *RefreshCmd) Run(ctx *Context) error {
	body, err := json.Marshal(struct {
		AvailabilityIDs []string `json:"availability_ids"`
	}{c.AvailabilityID})
	if err != nil {
		return err
	}
	return ctx.Client.Do(api.Request{Method: "POST", Path: "/refresh", Body: bytes.NewReader(body)}, ctx.Stdout)
}
