package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/10/seats-aero-cli/internal/api"
	"github.com/10/seats-aero-cli/internal/cli"
	"github.com/10/seats-aero-cli/internal/config"
	"github.com/alecthomas/kong"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithBaseURLs(args, stdout, stderr, api.BaseURL, api.RoomsBaseURL)
}

// URLs are injected only here for tests; the binary uses the fixed API hosts.
func runWithBaseURLs(args []string, stdout, stderr io.Writer, baseURL, roomsBaseURL string) int {
	var root cli.Root
	var parseStderr bytes.Buffer
	parserExit := -1
	parser, err := kong.New(&root,
		kong.Name("seatsaero"),
		kong.Description("Query seats.aero flights and rooms.aero hotels as JSON."),
		kong.Vars{"version": "seatsaero " + version()},
		kong.Writers(stdout, &parseStderr),
		kong.Exit(func(code int) { parserExit = code }),
	)
	if err != nil {
		return reportError(stderr, &api.Error{Code: "config_error", Message: "could not initialize command parser"})
	}
	parsed, err := parser.Parse(args)
	if parserExit >= 0 {
		return parserExit
	}
	if err != nil {
		parser.FatalIfErrorf(err)
		io.WriteString(stderr, redactUsage(parseStderr.String(), args, &root))
		return 2
	}
	ctx := &cli.Context{Stdout: stdout, Stderr: stderr}
	if parsed.Command() == "auth <key>" {
		ctx.ConfigPath, err = config.Path()
		if err != nil {
			return reportError(stderr, &api.Error{Code: "config_error", Message: "could not locate config file"})
		}
	} else {
		key, err := root.ResolveKey()
		if err != nil {
			return reportError(stderr, err)
		}
		if strings.HasPrefix(parsed.Command(), "rooms ") {
			baseURL = roomsBaseURL
		}
		ctx.Client = api.NewClient(baseURL, key, version())
	}
	if err := parsed.Run(ctx); err != nil {
		return reportError(stderr, err)
	}
	return 0
}

func redactUsage(message string, args []string, root *cli.Root) string {
	keys := []string{root.APIKey, root.Auth.Key, os.Getenv("SEATSAERO_API_KEY")}
	for i, arg := range args {
		if value, ok := strings.CutPrefix(arg, "--api-key="); ok {
			keys = append(keys, value)
		}
		if arg == "--api-key" && i+1 < len(args) {
			keys = append(keys, args[i+1])
		}
		if arg == "auth" {
			// Parse can fail before Kong assigns the auth positional to root.
			keys = append(keys, args[i+1:]...)
		}
	}
	for _, key := range keys {
		if key != "" {
			message = strings.ReplaceAll(message, key, "[REDACTED]")
		}
	}
	return message
}

func reportError(stderr io.Writer, err error) int {
	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		apiErr = &api.Error{Code: "upstream_error", Message: "command failed"}
	}
	if err := json.NewEncoder(stderr).Encode(struct {
		Error *api.Error `json:"error"`
	}{apiErr}); err != nil {
		return 1
	}
	return apiErr.ExitCode()
}
