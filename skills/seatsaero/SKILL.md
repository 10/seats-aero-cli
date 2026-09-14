---
name: seatsaero
description: Search award flights with seats.aero and hotel stays with rooms.aero through the seatsaero CLI; inspect itineraries, room rates, booking links, and route history using an existing Pro key.
---

# seatsaero

Use `seatsaero` for cached award flights and experimental route history, and `seatsaero rooms` for hotel stays. Both use the same key configuration. Stdout is the original API JSON except flight `--all`, which produces a combined page envelope. Check the exit code before consuming results; usage errors are text, other failures are one JSON envelope on stderr. Live search, retries, ranking, and refresh polling are not built in.

## Setup

From a published module version:

```sh
go install github.com/10/seats-aero-cli/cmd/seatsaero@latest
```

This installs `seatsaero` into `GOBIN`, or `~/go/bin` by default; put that directory on `PATH`. While the repo is private, set `GOPRIVATE=github.com/10/seats-aero-cli` and use authenticated GitHub access. From a checkout, `go build -o seatsaero ./cmd/seatsaero` creates `./seatsaero`.

Use an existing `SEATSAERO_API_KEY`, `--api-key KEY`, or `seatsaero auth KEY`. Precedence: flag → environment → config. `auth` saves the key in `$XDG_CONFIG_HOME/seatsaero/config.json`, otherwise `~/.config/seatsaero/config.json`, and prints only the path. CLI key arguments may remain in shell history. Never put real keys in committed files or reports. `seatsaero --version` and all `--help` commands work without a key.

## Flight command reference

| Command | Purpose |
| --- | --- |
| `search SFO,LAX NRT,HND --cabins business --start-date 2026-10-01 --end-date 2026-10-31` | Find space on one or several routes |
| `availability --source aeroplan --cabin first --origin-region "North America" --destination-region Asia` | Scan one program |
| `trips AVAILABILITY_ID` | Get concrete itineraries, segments, and booking links |
| `routes --source united` | Get the program's tracked routes (bare array) |
| `history ROUTE_ID --kind price --start-date 2026-09-01 --end-date 2026-09-14` | Get historical award mileage prices; omit kind for seat availability |
| `destinations --origin-airport SFO` | Get cheapest raw nonstop mileage to reachable airports |
| `destinations --destination-airport NRT` | Get nonstop origins serving an airport |
| `refresh ID [ID...]` | Queue or check the same IDs once |
| `alerts` | List already-configured alerts |
| `auth KEY` | Save a key locally; no network request |

Required inputs are shown above. Destinations requires exactly one airport flag. Search and availability API flags map to parameters by replacing `-` with `_`; `--all` and `--max-pages` control the CLI locally. Unset API flags are omitted, except **`--take 50`**. Rows are about 2.2 KB each; increase `take` only deliberately. Airports and carriers are uppercased; other search values go upstream exactly as typed. Let an API `400` identify invalid values instead of inventing client-side vocabularies or date shorthand.

### Search flags

| Flag | Type / meaning |
| --- | --- |
| `--start-date`, `--end-date` | Inclusive `YYYY-MM-DD` departure bounds |
| `--sources` | Comma-separated program identifiers |
| `--cabins` | Comma-separated `economy,premium,business,first` |
| `--carriers` | Comma-separated IATA airline codes |
| `--only-direct-flights` | Boolean; nonstop only |
| `--include-trips` | Boolean; embeds `AvailabilityTrips`, slower and larger |
| `--minify-trips` | Boolean; reduced fields with `--include-trips` |
| `--include-filtered` | Boolean; include normally-hidden dynamically priced results |
| `--min-cabin-pct` | Integer 0–100; minimum distance in reported cabin or higher; API default 100 |
| `--order-by` | `lowest_mileage`, or omit for date then premium cabins first |
| `--take` | Integer 10–1000, CLI default **50** |
| `--skip` | Integer number of rows already retrieved |
| `--cursor` | Opaque integer from the first response |

### Availability flags

| Flag | Type / meaning |
| --- | --- |
| `--source` | Required; exactly one program |
| `--cabin` | Exactly one cabin |
| `--start-date`, `--end-date` | Inclusive `YYYY-MM-DD` bounds |
| `--origin-region`, `--destination-region` | Region names; quote multi-word names |
| `--include-filtered` | Boolean |
| `--min-cabin-pct` | Integer 0–100; API default 100 |
| `--take` | Integer 10–1000, CLI default **50** |
| `--skip`, `--cursor` | Pagination integers as above |

Trips accepts `--include-filtered` and `--min-cabin-pct`. Routes requires `--source` and has no other command flags. Destinations accepts only `--origin-airport` or `--destination-airport`. Refresh takes space-separated IDs and sends `{"availability_ids":["ID", "ID"]}`. Alerts has no command flags. All commands accept the global `--api-key`, `--help`, and `--version` flags.

## Hotels (Rooms.aero)

```sh
seatsaero rooms search --location Tokyo --start-date 2026-11-20 --end-date 2026-11-27 --nights 3 --source hyatt
seatsaero rooms hotels --source hyatt --search "Park Hyatt"
seatsaero rooms availability --hotel-id HOTEL_ID --nights 3
seatsaero rooms details AVAILABILITY_ID
seatsaero rooms refresh HOTEL_ID
seatsaero rooms alerts --include-expired
```

Search returns `hotel.id`; use it with `rooms availability --hotel-id`. Pass an
availability row's `id` to `rooms details` for room types and `booking_link`.
`rooms hotels` also finds IDs using country/city/name or program property codes.
Availability accepts `--source`, `--hotel-id`, or both; `--rate-type cash` scans
cash rates and `--room-type suite` scopes rate filters to suites. Each command's
`--help` lists its filters; values pass upstream as typed.

Dates bound check-in days, not check-out. `--nights` selects 1–5 nights; omission
includes all lengths. Points and cash cover the whole stay; cash uses the named
currency's minor units. Use returned `*_available` flags to interpret zero prices.
`last_checked_at` is UTC; search/calendar data may be up to 21 days old.

Rooms lists default to `--take 50`. Inspect `has_more` (snake case); advance
`--skip` by `--take` until false or your request budget is reached. There is no
Rooms `--all` or `--cursor`. Deduplicate by `id`, or `hotel.id` for search.

The same Pro key works with a separate 1,000-call UTC-day quota. **Hotel refresh
is one request per user every 24 hours, shared with the website.** Request it
once, then read availability and watch `last_checked_at` advance. Repeating the
refresh POST within 24 hours returns a limit error, not progress. Alerts are read-only.
Sources: `hilton`, `hyatt`, `ihg`, `marriott`, `choice`, `wyndham`, `iprefer`.
Reference: [Rooms API](https://developers.seats.aero/reference/rooms-getting-started),
[data conventions](https://developers.seats.aero/reference/rooms-concepts),
[refresh](https://developers.seats.aero/reference/rooms-refresh-hotel).

## Flight identifiers

Sources as published **2026-08-28**; these describe upstream, not a local allowlist:

| Source | Program | Source | Program |
| --- | --- | --- | --- |
| `aeromexico` | Aeromexico Club Premier | `flyingblue` | Air France/KLM Flying Blue |
| `aeroplan` | Air Canada Aeroplan | `frontier` | Frontier Airlines |
| `alaska` | Alaska Mileage Plan | `jetblue` | JetBlue TrueBlue |
| `american` | American Airlines | `lufthansa` | Lufthansa Miles & More |
| `azul` | Azul TudoAzul | `qantas` | Qantas Frequent Flyer |
| `connectmiles` | Copa ConnectMiles | `qatar` | Qatar Privilege Club |
| `delta` | Delta SkyMiles | `saudia` | Saudia AlFursan |
| `emirates` | Emirates Skywards | `singapore` | Singapore KrisFlyer |
| `ethiopian` | Ethiopian ShebaMiles | `smiles` | GOL Smiles |
| `etihad` | Etihad Guest | `spirit` | Spirit Airlines |
| `eurobonus` | SAS EuroBonus | `turkish` | Turkish Miles & Smiles |
| `finnair` | Finnair Plus | `united` | United MileagePlus |
| | | `velocity` | Virgin Australia Velocity |
| | | `virginatlantic` | Virgin Atlantic Flying Club |

Cabins: `economy`, `premium`, `business`, `first`.

Regions: `North America`, `South America`, `Africa`, `Asia`, `Europe`, `Oceania`.

## Reading flight results

Availability rows use cabin prefixes `Y` economy, `W` premium economy, `J` business, and `F` first. `XAvailable` is a boolean; `XMileageCost` is a **string** (`"0"` when unavailable); `XRemainingSeats: 0` means **unknown**, not sold out. `XAirlines` is comma-separated text. `XDirect` marks nonstop availability. Raw and Direct field variants describe unfiltered and nonstop-only values.

Trips have integer `MileageCost`, `AvailabilitySegments`, `RemainingSeats`, and `TotalDuration` in minutes. Taxes are minor units of `TaxesCurrency`. Trip `DepartsAt` and `ArrivesAt` carry `Z` but are **airport local times**: do not convert them as UTC. `CreatedAt` and `UpdatedAt` are UTC. `MixedCabinPct`, when present, is the distance percentage below the reported cabin; it is inverse to the `min_cabin_pct` filter.

Destinations prices are cheapest raw nonstop miles across programs; `null` means no availability for that cabin. Inspect trips and booking links before making a booking decision.

```sh
set -o pipefail
# Numeric sort is necessary because availability mileage values are strings.
seatsaero search SFO NRT --cabins business |
  jq '[.data[] | select(.JAvailable)] | sort_by(.JMileageCost | tonumber) |
      map({ID, Date, miles: .JMileageCost, seats: .JRemainingSeats, airlines: .JAirlines})'

seatsaero trips AVAILABILITY_ID |
  jq '{booking_links, trips: [.data[] | {Cabin, FlightNumbers, DepartsAt, ArrivesAt, MileageCost, TotalTaxes, TaxesCurrency, RemainingSeats}]}'

seatsaero routes --source united |
  jq '[.[] | select(.OriginAirport == "SFO") | .DestinationAirport] | unique'
```

## Flight pagination and refresh

Use `--all --max-pages N` on search or availability for automatic collection within
a request budget (default **10** pages). It keeps the original cursor, advances
skip by upstream rows including duplicates, and keeps the first row for each `ID`.
It buffers the collected rows; a failed request or invalid response returns an
error with empty stdout. Default single-page commands still stream the API response.

The combined envelope contains `data`, `count` (unique rows), `hasMore`, `cursor`,
`nextSkip` (offset including any starting skip), and `pages` (requests made).
**Exit 0 can still have `hasMore: true`** when the budget is reached. Resume with
the same filters and `--cursor CURSOR --skip NEXT_SKIP --all`, then deduplicate
across runs. `hasMore: false` means this cached query is exhausted, not that every
airline or live award was searched. Choose the program allowlist with `--sources`;
verify its transfer eligibility for the traveler's points separately.

For manual pagination without `--all`:

1. Read `data`, `hasMore`, and `cursor` from the first response. Preserve that first cursor as an opaque integer.
2. If `hasMore` is true and more data is needed, repeat the same command and filters with `--cursor CURSOR --skip N`. `N` is the cumulative number of rows retrieved, including duplicates; it is not the number of unique IDs.
3. Add the new page's row count to `N`, retaining the first cursor. Repeat until `hasMore` is false or the requested scope is satisfied. Each page costs a quota call.
4. Deduplicate collected rows by `ID`. For newline-separated page envelopes, use `jq -s '[.[].data[]] | unique_by(.ID)'`. The upstream `moreURL` also describes the next page, but the CLI does not follow it automatically.

`refresh ID...` posts once. Re-run the identical IDs to check completion. Stop at `complete: true`. Item states include `queued`, `processing`, `succeeded`, `failed`, `fresh`, `skipped_outage`, `not_refreshable`, `not_found`, and `insufficient_quota`. Only `queued` spends a credit; `fresh` means updated within three hours. The CLI does not wait or poll itself.

## Route history

Get the `ID` from `routes --source PROGRAM`; it identifies a program's route,
not a search availability record. Call `history ID --start-date YYYY-MM-DD
--end-date YYYY-MM-DD`. Dates refer to **observation days**, not travel dates.

The default kind is `availability`, returning daily `Y/W/J/FRemainingSeats`.
`--kind price` returns `Y/W/J/FMileageCost`, which are award mileage prices,
not cash fares. Both preserve the website's JSON array with `ObservationDay`.
Optional filters: `--cabins Y,W,J,F` (all if omitted), `--stops` (default `direct`),
and `--metric` (default `sum` for availability, `max` for price). These map to
the website's `cabins`, `stops`, and `metric`; dates map to `start` and `end`.

This is an experimental website API using the existing Pro key with no browser
cookies or redirects. It was verified separately from the supported Partner API;
availability and quota behavior may differ. HTML, a non-array response or rows
without `ObservationDay` fail as `invalid_response` rather than masquerading as
empty history. It makes one request and buffers the response for validation.

## Failures and quota

| Exit | Code / condition | Next action |
| --- | --- | --- |
| 0 | Success | Read stdout |
| 1 | `network_error` | Check connectivity; decide whether to spend another call |
| 1 | `upstream_error` | Unexpected status or `5xx`; inspect the error before retrying |
| 1 | `invalid_response` | Unexpected page/history schema or stalled pagination; discard output and inspect endpoint compatibility |
| 1 | `config_error` | Fix config JSON or filesystem access |
| 2 | `bad_request` | Fix API parameters or local pagination flags using the message and `--help` |
| 2 | Plain-text Kong usage error | Supply required arguments / correct flags |
| 3 | `missing_api_key` | Configure a key |
| 3 | `unauthorized` (`401`/`403`) | Fix key, Pro subscription, or endpoint access |
| 4 | `not_found` (`404`) | Recheck the resource ID |
| 5 | `quota_exceeded` (`429`) | Stop and back off until reset; use reset duration in the message |

JSON errors have `error.code`, `error.message`, `error.status` (zero when unavailable), and `error.upstream` (trimmed, capped at 64 KiB, key redacted). A broken response stream can leave partial stdout; discard it on nonzero exit. Enable `pipefail` in shell pipelines.

The flight API allows **1,000 calls per calendar day, resetting at midnight UTC**. This is a daily budget. Each page is a call; each queued flight refresh ID spends a credit from the same pool. Flight refresh also has an hourly request cap. No automatic retries or quota accounting occur. Pro access is personal and non-commercial; live flight search requires separate commercial access and has no CLI command.
