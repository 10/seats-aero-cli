<div align="center">

# seatsaero

*Search seats.aero award flights and rooms.aero hotel stays from your terminal.*

[![Go](https://img.shields.io/badge/Go-1.26.3+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![API](https://img.shields.io/badge/API-seats.aero-3665F3?style=flat)](https://developers.seats.aero/reference/getting-started-p)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=flat)](LICENSE)

</div>

---

```console
$ seatsaero search SFO NRT --cabins business | jq '.data[0].JMileageCost'
"75000"

$ seatsaero rooms search --location Tokyo --source hyatt \
    --start-date 2026-10-14 --end-date 2026-10-21 --nights 3 --take 3 |
    jq '.data[0] | {hotel: .hotel.name, points: .lowest_award_standard}'
{
  "hotel": "Hyatt Regency Tokyo Bay",
  "points": 15000
}
```

Search cached award space across mileage programs, then drill into the flights
behind a result. Find hotel awards through Rooms.aero with the same Go binary
and seats.aero Pro API key.

## Features

- **Flexible routes** — search multiple origin and destination airports in one call, with cabin, airline, and nonstop filters.
- **Itinerary details** — flight numbers, times, mileage, taxes, seats, and booking links.
- **Explore a program** — browse its availability and tracked routes, or find nonstop destinations from an airport.
- **Collect pages** — optionally paginate and deduplicate within a request budget, with metadata for resuming.
- **Route history** — retrieve daily availability and award mileage prices through the experimental website API.
- **Hotel stays** — search an area, inspect hotel calendars and room prices, and get booking links through Rooms.aero.

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
seatsaero search SFO CAN --sources aeroplan --all --max-pages 3
seatsaero availability --source aeroplan --cabin first
seatsaero trips AVAILABILITY_ID             # itineraries and booking links
seatsaero routes --source united
seatsaero history ROUTE_ID --kind price --start-date 2026-09-01 --end-date 2026-09-14
seatsaero destinations --origin-airport SFO  # nonstop destinations
seatsaero refresh AVAILABILITY_ID           # queue or check a refresh
seatsaero alerts
```

`--start-date` and `--end-date` narrow a search. Search and availability default
to `--take 50`. Use a command's `--help` for all filters.

By default, stdout is the API's JSON. `--all` produces a combined page envelope.
Errors go to stderr, so results work directly with `jq`:

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

Use `--all` on search or availability to collect pages, with `--max-pages N`
limiting requests (default **10**). Rows are deduplicated by `ID`, keeping the
first occurrence and all of its original fields. Output contains `data`,
`count` (unique rows returned), `hasMore`, the original `cursor`, `nextSkip`
(upstream rows traversed, including duplicates and any starting skip), and `pages`.

**Check `hasMore` even on exit 0.** If true, the request budget stopped collection.
Resume with the same query, `--cursor CURSOR --skip NEXT_SKIP --all`. When combining
resumed runs, deduplicate their rows too. `hasMore: false` means this cached query
is exhausted; it does not establish that every airline or live award was searched.
Collection buffers results until all requested pages succeed; any request or
response-shape failure exits nonzero without printing partial results. It does not retry.

For manual pagination, keep the first response's `cursor` and repeat the query with
`--cursor CURSOR --skip N`, where `N` is the number of rows already retrieved.
Stop when `hasMore` is false; deduplicate overlapping pages by `ID`.

Repeat `refresh` with the same IDs to check progress. `complete: true` means
finished. Pro keys allow 1,000 calls per UTC day; each page costs a call and
queued refresh IDs spend credits from the same pool. Refresh also has an hourly cap.

Pro access is for personal, non-commercial use. Live search requires separate
commercial access and is not included.

## Route history

Find the route's `ID` using `routes --source PROGRAM`, then pass it to `history`.
The ID identifies a route in one program; it is not an availability ID from search.

```bash
seatsaero routes --source aeroplan |
  jq '.[] | select(.OriginAirport == "SFO" and .DestinationAirport == "HKG") | .ID'
seatsaero history ROUTE_ID --start-date 2026-09-01 --end-date 2026-09-14
seatsaero history ROUTE_ID --kind price --start-date 2026-09-01 --end-date 2026-09-14 --cabins J
```

Dates are observation dates, not flight departure dates. The default kind is
`availability` (`Y/W/J/FRemainingSeats`); `price` returns award mileage costs
(`Y/W/J/FMileageCost`), not cash fares. Both return the original JSON array with
`ObservationDay`. The default stops filter is `direct`, and the default aggregation
metric is `sum` for availability or `max` for price. `--cabins` accepts comma-separated
`Y,W,J,F`; `--stops` and `--metric` pass website filter values through.

This command uses undocumented seats.aero website endpoints that worked with a
Pro key during testing. Their availability and quota rules are not a supported
Partner API contract. A history call makes one request, follows no redirects,
and rejects unexpected HTML or response shapes with `error.code: invalid_response`.

## Hotels

Your existing key also works with the [Rooms.aero API](https://developers.seats.aero/reference/rooms-getting-started).
Use the `rooms` command group; authentication and JSON/error handling are shared.

```bash
seatsaero rooms search --location Tokyo --start-date 2026-11-20 --end-date 2026-11-27 --nights 3 --source hyatt
seatsaero rooms hotels --source hyatt --search "Park Hyatt"
seatsaero rooms availability --hotel-id HOTEL_ID --nights 3
seatsaero rooms details AVAILABILITY_ID
seatsaero rooms refresh HOTEL_ID
seatsaero rooms alerts
```

Search dates bound **check-in dates**, not check-out. `--nights` selects a stay
of 1–5 nights; omit it for all lengths. Points and cash prices are for the whole
stay, with cash in the named currency's minor units. Details takes an availability
`id`; search returns `hotel.id` for calendar lookups. See the
[Rooms concepts](https://developers.seats.aero/reference/rooms-concepts) for response fields.

Hotel lists default to `--take 50` and return the original `data`, `count`,
`has_more`, and optional `more_url`. While `has_more` is true, repeat the same
query with `--skip` increased by `--take`. Deduplicate by `id` (`hotel.id` for
search). Rooms commands use manual pagination and have no `--all` or `--cursor`.
Use each command's `--help` for the available filters.

Rooms has a separate **1,000-call UTC-day quota**. Hotel refresh is one request
per user per 24 hours, shared with the website. After requesting it once, read
`rooms availability --hotel-id HOTEL_ID` and watch `last_checked_at` for new data;
repeating `rooms refresh` is not a status check. [Refresh documentation](https://developers.seats.aero/reference/rooms-refresh-hotel).

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
