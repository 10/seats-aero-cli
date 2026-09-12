---
name: seatsaero
description: Query seats.aero award-flight availability with a Pro key; inspect itineraries, program routes, nonstop destinations, refresh status, and existing alerts from the shell.
---

# seatsaero

Use `seatsaero` for cached award availability. Read stdout as the original API JSON. Check the exit code before consuming results; usage errors are text, other failures are one JSON envelope on stderr. No live search, retries, automatic pagination, ranking, or refresh polling is built in.

## Setup

From a published module version:

```sh
go install github.com/10/seats-aero-cli/cmd/seatsaero@latest
```

This installs `seatsaero` into `GOBIN`, or `~/go/bin` by default; put that directory on `PATH`. While the repo is private, set `GOPRIVATE=github.com/10/seats-aero-cli` and use authenticated GitHub access. From a checkout, `go build -o seatsaero ./cmd/seatsaero` creates `./seatsaero`.

Use an existing `SEATSAERO_API_KEY`, `--api-key KEY`, or `seatsaero auth KEY`. Precedence: flag → environment → config. `auth` saves the key in `$XDG_CONFIG_HOME/seatsaero/config.json`, otherwise `~/.config/seatsaero/config.json`, and prints only the path. CLI key arguments may remain in shell history. Never put real keys in committed files or reports. `seatsaero --version` and all `--help` commands work without a key.

## Command reference

| Command | Purpose |
| --- | --- |
| `search SFO,LAX NRT,HND --cabins business --start-date 2026-10-01 --end-date 2026-10-31` | Find space on one or several routes |
| `availability --source aeroplan --cabin first --origin-region "North America" --destination-region Asia` | Scan one program |
| `trips AVAILABILITY_ID` | Get concrete itineraries, segments, and booking links |
| `routes --source united` | Get the program's tracked routes (bare array) |
| `destinations --origin-airport SFO` | Get cheapest raw nonstop mileage to reachable airports |
| `destinations --destination-airport NRT` | Get nonstop origins serving an airport |
| `refresh ID [ID...]` | Queue or check the same IDs once |
| `alerts` | List already-configured alerts |
| `auth KEY` | Save a key locally; no network request |

Required inputs are shown above. Destinations requires exactly one airport flag. Flags below map to API parameters by replacing `-` with `_`. Flags not supplied are omitted, except **`--take 50`**. Rows are about 2.2 KB each; increase `take` only deliberately. Airports and carriers are uppercased; other values go upstream exactly as typed. Let an API `400` identify invalid values instead of inventing client-side vocabularies or date shorthand.

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

## Identifiers

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

## Reading results

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

## Pagination and refresh

For search or availability:

1. Read `data`, `hasMore`, and `cursor` from the first response. Preserve that first cursor as an opaque integer.
2. If `hasMore` is true and more data is needed, repeat the same command and filters with `--cursor CURSOR --skip N`. `N` is the cumulative number of rows retrieved, including duplicates; it is not the number of unique IDs.
3. Add the new page's row count to `N`, retaining the first cursor. Repeat until `hasMore` is false or the requested scope is satisfied. Each page costs a quota call.
4. Deduplicate collected rows by `ID`. For newline-separated page envelopes, use `jq -s '[.[].data[]] | unique_by(.ID)'`. The upstream `moreURL` also describes the next page, but the CLI does not follow it automatically.

`refresh ID...` posts once. Re-run the identical IDs to check completion. Stop at `complete: true`. Item states include `queued`, `processing`, `succeeded`, `failed`, `fresh`, `skipped_outage`, `not_refreshable`, `not_found`, and `insufficient_quota`. Only `queued` spends a credit; `fresh` means updated within three hours. The CLI does not wait or poll itself.

## Failures and quota

| Exit | Code / condition | Next action |
| --- | --- | --- |
| 0 | Success | Read stdout |
| 1 | `network_error` | Check connectivity; decide whether to spend another call |
| 1 | `upstream_error` | Unexpected status or `5xx`; inspect the error before retrying |
| 1 | `config_error` | Fix config JSON or filesystem access |
| 2 | `bad_request` (`400`) | Fix parameters using `upstream` and `--help` |
| 2 | Plain-text Kong usage error | Supply required arguments / correct flags |
| 3 | `missing_api_key` | Configure a key |
| 3 | `unauthorized` (`401`/`403`) | Fix key, Pro subscription, or endpoint access |
| 4 | `not_found` (`404`) | Recheck the availability ID |
| 5 | `quota_exceeded` (`429`) | Stop and back off until reset; use reset duration in the message |

JSON errors have `error.code`, `error.message`, `error.status` (zero without a response), and `error.upstream` (trimmed, capped at 64 KiB, key redacted). A broken response stream can leave partial stdout; discard it on nonzero exit. Enable `pipefail` in shell pipelines.

Pro keys allow **1,000 calls per calendar day, resetting at midnight UTC**. This is a daily budget. Each page is a call; each queued refresh ID spends a credit from the same pool. Refresh also has an hourly request cap. No automatic retries or quota accounting occur. Pro access is personal and non-commercial; live search requires separate commercial access and has no CLI command.
