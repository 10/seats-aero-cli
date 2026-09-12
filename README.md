<div align="center">

# seatsaero

*Search seats.aero award flights from your terminal.*

[![Go](https://img.shields.io/badge/Go-1.26.3+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![API](https://img.shields.io/badge/API-seats.aero-3665F3?style=flat)](https://developers.seats.aero/reference/getting-started-p)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=flat)](LICENSE)

</div>

---

```console
$ seatsaero search SFO NRT --cabins business | jq '.data[0].JMileageCost'
"75000"
```

Search cached award space across mileage programs, then drill into the flights
behind a result. One Go binary and a seats.aero Pro API key.

## Features

- **Flexible routes** — search multiple origin and destination airports in one call, with cabin, airline, and nonstop filters.
- **Itinerary details** — flight numbers, times, mileage, taxes, seats, and booking links.
- **Explore a program** — browse its availability and tracked routes, or find nonstop destinations from an airport.

## Install

```bash
go install github.com/10/seats-aero-cli/cmd/seatsaero@latest
```

Requires Go `1.26.3` or later, with `~/go/bin` (or `GOBIN`) on `PATH`.
While the repo is private, set `GOPRIVATE=github.com/10/seats-aero-cli` and use
authenticated GitHub access.

## CLI

```bash
seatsaero search SFO,LAX NRT,HND --cabins business --only-direct-flights
seatsaero availability --source aeroplan --cabin first
seatsaero trips AVAILABILITY_ID             # itineraries and booking links
seatsaero routes --source united
seatsaero destinations --origin-airport SFO  # nonstop destinations
seatsaero refresh AVAILABILITY_ID           # queue or check a refresh
seatsaero alerts
```

`--start-date` and `--end-date` narrow a search. Search and availability default
to `--take 50`. Use a command's `--help` for all filters.

Stdout is the API's JSON. Errors go to stderr, so results work directly with `jq`:

```bash
set -o pipefail
seatsaero search SFO NRT --cabins business |
  jq '[.data[] | select(.JAvailable)] | sort_by(.JMileageCost | tonumber)'
```

Exit codes: `0` ok, `1` runtime error, `2` bad request or usage, `3` missing or
unauthorized key, `4` not found, `5` quota exceeded. API errors carry a stable
`error.code`; usage errors are plain text.

Get a Pro key from seats.aero Settings → API and save it once:

```bash
seatsaero auth YOUR_PRO_KEY
```

Keys resolve from `--api-key`, then `SEATSAERO_API_KEY`, then
`~/.config/seatsaero/config.json`. `XDG_CONFIG_HOME` overrides the config root.
`auth` writes the file with mode `0600`.

## Availability

| Field | Read it as |
| --- | --- |
| `YMileageCost`, `WMileageCost`, `JMileageCost`, `FMileageCost` | Miles as strings, for economy, premium economy, business, and first |
| `JRemainingSeats: 0` | Seat count unknown, not sold out |
| `TotalTaxes` | Minor units of `TaxesCurrency` |
| Trip `DepartsAt` / `ArrivesAt` | Airport local time, **despite the `Z` suffix** |

To paginate, keep the first response's `cursor` and repeat the query with
`--cursor CURSOR --skip N`, where `N` is the number of rows already retrieved.
Stop when `hasMore` is false; deduplicate overlapping pages by `ID`.

Repeat `refresh` with the same IDs to check progress. `complete: true` means
finished. Pro keys allow 1,000 calls per UTC day; each page costs a call and
queued refresh IDs spend credits from the same pool. Refresh also has an hourly cap.

Pro access is for personal, non-commercial use. Live search requires separate
commercial access and is not included.

## Agents

```bash
mkdir -p ~/.claude/skills
cp -r skills/seatsaero ~/.claude/skills/seatsaero
```

[`SKILL.md`](skills/seatsaero/SKILL.md) covers the full command surface, program
identifiers, pagination, and response fields. It uses the CLI's existing key configuration.

## Development

```bash
go test ./... -race
go vet ./... && gofmt -l .
```

With `SEATSAERO_API_KEY` set:

```bash
go test -tags integration ./... -v -count=1
```

The live suite makes up to six read-only calls. Refresh is tested locally.
