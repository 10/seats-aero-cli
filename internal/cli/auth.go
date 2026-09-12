package cli

import (
	"encoding/json"

	"github.com/10/seats-aero-cli/internal/api"
	"github.com/10/seats-aero-cli/internal/config"
)

type AuthCmd struct {
	Key string `arg:"" help:"Pro API key to save."`
}

func (c *AuthCmd) Run(ctx *Context) error {
	if err := config.Write(ctx.ConfigPath, c.Key); err != nil {
		return &api.Error{Code: "config_error", Message: "could not write config file; check the directory and permissions"}
	}
	return json.NewEncoder(ctx.Stdout).Encode(struct {
		Config string `json:"config"`
	}{ctx.ConfigPath})
}
