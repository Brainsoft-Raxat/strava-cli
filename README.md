# strava-cli

A production-ready CLI for the [Strava API v3](https://developers.strava.com/), built on a generated OpenAPI client.

## Features

- **26 commands** across athlete, activities, clubs, gear, routes, segments, and uploads
- OAuth2 with automatic token refresh (6-hour Strava tokens are handled silently)
- `--json` flag on every read command for scripting / `jq` pipelines
- Write commands require `--yes` or interactive confirmation; `--dry-run` on all of them
- Retries with exponential backoff on HTTP 429 / 5xx
- Token + credentials stored in `~/.config/strava-cli/config.json` (mode 0600)
- Shell completion for bash, zsh, fish, PowerShell

## Installation

**Pre-built binary** (recommended) — download from [GitHub Releases](https://github.com/Brainsoft-Raxat/strava-cli/releases/latest):

```bash
# Linux / macOS (replace VERSION and ARCH as needed)
curl -L https://github.com/Brainsoft-Raxat/strava-cli/releases/latest/download/stravacli_VERSION_linux_amd64.tar.gz | tar xz
sudo mv stravacli /usr/local/bin/
```

**Go install:**

```bash
go install github.com/Brainsoft-Raxat/strava-cli@latest
```

**Build from source:**

```bash
git clone https://github.com/Brainsoft-Raxat/strava-cli
cd strava-cli
make build     # → ./stravacli
make install   # → $GOPATH/bin/stravacli
```

## Setup

### 1. Create a Strava API application

1. Go to <https://www.strava.com/settings/api>
2. Create an application (any name/website works for personal use)
3. Set **Authorization Callback Domain** to `localhost`
4. Note your **Client ID** and **Client Secret**

### 2. Authenticate

```bash
stravacli auth login
# prompts for Client ID + Client Secret (or reads STRAVA_CLIENT_ID / STRAVA_CLIENT_SECRET)
# opens your browser at the Strava authorization page
# captures the callback automatically on localhost:8089
```

Using an external redirect URI (e.g. a hosted callback):

```bash
STRAVA_CLIENT_ID=12345 \
STRAVA_CLIENT_SECRET=abc \
STRAVA_REDIRECT_URI=https://example.com/auth/callback \
stravacli auth login
# prints the auth URL and prompts you to paste the redirect URL back
```

## Commands

### auth

```bash
stravacli auth login    # OAuth2 browser flow; stores tokens
stravacli auth status   # show token validity and expiry
stravacli auth logout   # delete stored credentials (prompts for confirmation)
```

### athlete

```bash
stravacli athlete me                  # your profile
stravacli athlete stats               # lifetime totals (runs, rides, swims)
stravacli athlete stats 12345678      # another athlete's stats by ID
stravacli athlete zones               # heart rate and power zones
```

### activities

```bash
# List
stravacli activities list
stravacli activities list --page 2 --per-page 50
stravacli activities list --after $(date -d '7 days ago' +%s)   # last 7 days
stravacli activities list --before $(date -d 'yesterday' +%s)

# Get
stravacli activities get 12345678901
stravacli activities laps 12345678901
stravacli activities zones 12345678901
stravacli activities comments 12345678901
stravacli activities kudos 12345678901

# Streams (time-series sensor data)
stravacli activities streams 12345678901
stravacli activities streams 12345678901 --keys time,heartrate,watts,cadence

# Update (write — requires --yes or interactive confirm)
stravacli activities update 12345678901 --name "Morning 10k" --yes
stravacli activities update 12345678901 --commute --hide --yes
stravacli activities update 12345678901 --type Run --gear-id b12345678 --yes
stravacli activities update 12345678901 --name "Test" --dry-run   # preview only

# Upload (write — requires --yes or interactive confirm)
stravacli activities upload --file morning.gpx --yes
stravacli activities upload --file workout.fit --name "Intervals" --wait --yes
stravacli activities upload --file ride.tcx --commute --yes
stravacli activities upload --file archive.fit.gz --data-type fit.gz --yes
stravacli activities upload --file morning.gpx --dry-run   # preview only
```

**Supported upload formats:** `fit`, `fit.gz`, `tcx`, `tcx.gz`, `gpx`, `gpx.gz`

`--wait` polls every 3 seconds until Strava finishes processing and prints the new activity ID.

### uploads

```bash
stravacli uploads get 18561703846    # check processing status by upload ID
```

### clubs

```bash
stravacli clubs list                 # clubs you belong to
stravacli clubs get 12345
stravacli clubs members 12345
stravacli clubs activities 12345
```

### gear

```bash
stravacli gear get b12345678         # bike (b prefix)
stravacli gear get g12345678         # shoes (g prefix)
```

### routes

```bash
stravacli routes list                # your routes
stravacli routes list 12345678       # another athlete's routes by ID
stravacli routes get 12345678

# Export — downloads a GPX or TCX file
stravacli routes export 12345678 --format gpx
stravacli routes export 12345678 --format tcx --out /tmp/my-route.tcx
# defaults to route-<id>.<format> in the current directory
```

### segments

```bash
stravacli segments get 12345678
stravacli segments starred
stravacli segments starred --page 2 --per-page 50

# Explore popular segments in a bounding box
stravacli segments explore --bounds 51.5,-0.2,51.6,-0.1
stravacli segments explore --bounds 51.5,-0.2,51.6,-0.1 --activity-type running
stravacli segments explore --bounds 51.5,-0.2,51.6,-0.1 --min-cat 2 --max-cat 4

# Segment efforts
stravacli segments efforts list --segment-id 12345678
stravacli segments efforts list --segment-id 12345678 --start-date 2024-01-01T00:00:00Z
stravacli segments efforts get 98765432
```

## JSON output

Every read command supports `--json` for clean machine-readable output:

```bash
stravacli activities list --json | jq '.[].name'
stravacli athlete me --json | jq '{name: (.firstname + " " + .lastname), city}'
stravacli segments explore --bounds 51.5,-0.2,51.6,-0.1 --json | jq '.[].name'
stravacli activities streams 12345 --keys heartrate --json | jq '.heartrate.data | max'
```

## Write safety

All commands that modify Strava data require explicit confirmation:

```bash
# Interactive prompt (default — safe for ad-hoc use)
stravacli activities update 12345 --name "New name"
# → About to update activity 12345 (name=New name)
# → Proceed? [y/N]

# Skip prompt — logs an AUDIT line to stderr instead
stravacli activities update 12345 --name "New name" --yes
# → AUDIT: update activity 12345 (name=New name)

# Dry-run — prints what would happen, makes no API call
stravacli activities upload --file run.gpx --dry-run
# → DRY RUN: would upload run.gpx (data_type=gpx)
```

## Shell completion

```bash
# bash
stravacli completion bash > /etc/bash_completion.d/strava

# zsh
stravacli completion zsh > "${fpath[1]}/_strava"

# fish
stravacli completion fish > ~/.config/fish/completions/strava.fish
```

## Version

```bash
stravacli --version
```

## Development

```bash
make build      # compile to ./stravacli
make install    # install to $GOPATH/bin
make test       # run tests with -race
make lint       # run golangci-lint (auto-installs if missing)
make generate   # regenerate OpenAPI client from strava.minimal.json
make snapshot   # local cross-platform build via GoReleaser (no tag needed)
make release    # publish tagged release to GitHub (requires GITHUB_TOKEN)
make clean      # remove binary and dist/
```

## Project structure

```
.
├── cmd/                    # Cobra commands
│   ├── root.go             # --json flag, --version
│   ├── auth.go             # login, status, logout
│   ├── athlete.go          # me, stats, zones
│   ├── activities.go       # list, get, laps, zones, comments, kudos, streams, update, upload
│   ├── clubs.go            # list, get, members, activities
│   ├── gear.go             # get
│   ├── routes.go           # list, get, export
│   ├── segments.go         # get, starred, explore, efforts list/get
│   ├── uploads.go          # get + polling helpers
│   └── helpers.go          # apiClient, rawClient, confirmMutation
├── internal/
│   ├── auth/               # OAuth2 login + token refresh
│   ├── client/             # Generated OpenAPI client + retrying transport
│   ├── config/             # JSON config persistence (~/.config/strava-cli/)
│   └── output/             # Human-readable and JSON printers
├── main.go
├── strava.minimal.json     # Trimmed OpenAPI 3.0 spec (26 operations)
├── oapi-codegen.yaml       # Code generation config
└── Makefile
```

## Token storage

Credentials and tokens are stored in `~/.config/strava-cli/config.json` (mode 0600).
Treat this file like a password — it contains your Client Secret and refresh token.

Strava access tokens expire after 6 hours; the CLI refreshes them automatically before
each request with no interruption.
