# BookmakersReview CLI

**Every sportsbook's price, BMR's own sharp consensus, and full line history — for free, from the terminal, no API key required.**

BookmakersReview runs a free, unauthenticated GraphQL feed behind its odds-comparison tool that already includes consensus, opening/current/historical lines, injuries, and weather in one graph. This CLI wraps it, adds a local SQLite history no paid odds API gives you by default, and layers on vig-free value finding, steam detection, and closing-line-value grading that no sportsbook data API — paid or free — ships out of the box.

## Install

The recommended path installs both the `bookmakersreview-pp-cli` binary and the `pp-bookmakersreview` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install bookmakersreview
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install bookmakersreview --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install bookmakersreview --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install bookmakersreview --agent claude-code
npx -y @mvanhorn/printing-press-library install bookmakersreview --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.5 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/cmd/bookmakersreview-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/bookmakersreview-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install bookmakersreview --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-bookmakersreview --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-bookmakersreview --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install bookmakersreview --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/bookmakersreview-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/cmd/bookmakersreview-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "bookmakersreview": {
      "command": "bookmakersreview-pp-mcp"
    }
  }
}
```

</details>

## Authentication

No authentication is required. Every query in this CLI works unauthenticated, exactly like the public odds.bookmakersreview.com tool.

## Quick Start

```bash
# confirm the CLI can reach the BMR GraphQL service
bookmakersreview-pp-cli doctor --dry-run

# find the league id for the sport you care about (NFL=16, NBA=5, MLB=3, NHL=7)
bookmakersreview-pp-cli leagues list --json --select lid,nam

# see today's/upcoming NFL games
bookmakersreview-pp-cli events list --league 16 --json

# find the best moneyline price across every tracked sportsbook
bookmakersreview-pp-cli odds best --event 4802244 --market 1 --json

# check whether that best price is actually positive expected value, not just the highest number
bookmakersreview-pp-cli odds value --event 4802244 --market 1 --agent

# scan the whole slate for sharp/steam moves before kickoff
bookmakersreview-pp-cli steam scan --league 16 --since 3h --agent

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Line shopping
- **`odds value`** — See which sportsbook's current price beats the vig-free fair line before you bet.

  _Reach for this before placing a bet to confirm you are not just getting the highest number, but a genuinely positive-EV price._

  ```bash
  bookmakersreview-pp-cli odds value --event 123456 --market 1 --agent
  ```
- **`steam scan`** — Scan today's whole slate for fast, coordinated line moves that signal sharp money before the market fully reacts.

  _Use this when you want to catch sharp action across an entire day's games rather than watching one event._

  ```bash
  bookmakersreview-pp-cli steam scan --league 16 --since 3h --agent
  ```
- **`arb scan`** — Find risk-free two-sided price mismatches across sportsbooks for one event.

  _Use this to find guaranteed-profit spreads across books; do not use it to judge single-side value, use odds value for that._

  ```bash
  bookmakersreview-pp-cli arb scan --event 123456 --market 1
  ```

### Closing line value
- **`bets record`** — Log your own wager (event, market, price, book, timestamp) to a local ledger.

  _Use this immediately after placing a real bet so it can later be graded against the closing line._

  ```bash
  bookmakersreview-pp-cli bets record --event 123456 --market 3 --price 2.5 --book 9 --boid 1
  ```
- **`bets grade`** — Compare one recorded bet's price to the market's closing line to compute its CLV.

  _Use this after a game closes to see whether you beat the closing number, the standard measure of betting skill._

  ```bash
  bookmakersreview-pp-cli bets grade --bet-id 42 --agent
  ```
- **`bets report`** — See your running closing-line-value percentage and win rate across every recorded bet.

  _Use this weekly/monthly to track betting performance over time instead of grading bets one at a time._

  ```bash
  bookmakersreview-pp-cli bets report --since 30d --group-by market --agent
  ```
- **`odds movement`** — See the full open-to-current price timeline for one event and market, across books.

  _Use this to see how a specific line moved over time; use steam scan instead to find movement across the whole day's slate._

  ```bash
  bookmakersreview-pp-cli odds movement --event 123456 --market 2 --agent
  ```

## Recipes

### Find the best price across books

```bash
bookmakersreview-pp-cli odds best --event 123456 --market 3 --json
```

Returns every tracked sportsbook's current spread price for the event, sorted best-first.

### Narrow a large event-history payload to just the fields you need

```bash
bookmakersreview-pp-cli events history --league 16 --from 2025-12-14 --to 2025-12-16 --agent --select eid,dt
```

A date-range history query returns every event's full per-period score breakdown; --select keeps agent context small by returning only the id and kickoff time when you just need to enumerate events (follow up with 'events get <eid>' for full scores on the ones you care about).

### Check if a price is actually good value, not just the highest number

```bash
bookmakersreview-pp-cli odds value --event 123456 --market 1 --agent
```

Strips the vig from consensus to compute fair value, then flags any book beating it — the highest number isn't always +EV.

### Scan for sharp/steam action across today's slate

```bash
bookmakersreview-pp-cli steam scan --league 16 --since 3h --agent
```

Flags markets where consensus moved fast and far from the opener across the board, a signal of sharp money.

### Track your own closing line value over time

```bash
bookmakersreview-pp-cli bets report --since 30d --group-by market --agent
```

Aggregates every bet you've recorded and graded into a running CLV percentage, the standard long-run skill metric for bettors.

## Usage

Run `bookmakersreview-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.json` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `BOOKMAKERSREVIEW_CONFIG_DIR`, `BOOKMAKERSREVIEW_DATA_DIR`, `BOOKMAKERSREVIEW_STATE_DIR`, or `BOOKMAKERSREVIEW_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `BOOKMAKERSREVIEW_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export BOOKMAKERSREVIEW_HOME=/srv/bookmakersreview
bookmakersreview-pp-cli doctor
```

Under `BOOKMAKERSREVIEW_HOME=/srv/bookmakersreview`, the four dirs resolve to `/srv/bookmakersreview/config`, `/srv/bookmakersreview/data`, `/srv/bookmakersreview/state`, and `/srv/bookmakersreview/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "bookmakersreview": {
      "command": "bookmakersreview-pp-mcp",
      "env": {
        "BOOKMAKERSREVIEW_HOME": "/srv/bookmakersreview"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `BOOKMAKERSREVIEW_DATA_DIR` overrides an explicit `--home` for that kind. Use `BOOKMAKERSREVIEW_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `BOOKMAKERSREVIEW_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `bookmakersreview-pp-cli doctor --fail-on warn` to check path warnings in automation.

## Commands

### graphql

Raw GraphQL passthrough for the BookmakersReview odds-v2 service (fallback for any of the 174 query fields not promoted as a typed command)

- **`bookmakersreview-pp-cli graphql`** - Execute a raw GraphQL query against the odds-v2 service


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`bookmakersreview-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`bookmakersreview-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`bookmakersreview-pp-cli learnings list`** - Inspect taught rows
- **`bookmakersreview-pp-cli learnings forget <query>`** - Undo a teach
- **`bookmakersreview-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`bookmakersreview-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`bookmakersreview-pp-cli teach-pattern`** - Install a query/resource template up front
- **`bookmakersreview-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `BOOKMAKERSREVIEW_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `bookmakersreview-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
bookmakersreview-pp-cli leagues list --sport 4

# JSON for scripting and agents
bookmakersreview-pp-cli leagues list --sport 4 --json

# Filter to specific fields
bookmakersreview-pp-cli leagues list --sport 4 --json --select lid,nam

# CSV for spreadsheets
bookmakersreview-pp-cli leagues list --sport 4 --csv

# Dry run — show the request without sending
bookmakersreview-pp-cli leagues list --sport 4 --dry-run

# Agent mode — JSON + compact + no prompts in one flag
bookmakersreview-pp-cli leagues list --sport 4 --agent
```

Raw `graphql --query '...'` responses are wrapped as `{"data": {...}}`; `--select` on that command only matches against the top-level `data` field, not nested fields inside it. Use the typed commands above (or `--select` on any typed command's own JSON output) when you need field filtering.

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
bookmakersreview-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Run `bookmakersreview-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/bookmakersreview-pp-cli/config.json`; `--home`, `BOOKMAKERSREVIEW_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Variable | Purpose |
|----------|---------|
| `BOOKMAKERSREVIEW_HOME` | Relocate all four path kinds (config/data/state/cache) under one root |
| `BOOKMAKERSREVIEW_CONFIG_DIR` / `_DATA_DIR` / `_STATE_DIR` / `_CACHE_DIR` | Relocate one path kind independently (overrides `--home`/`BOOKMAKERSREVIEW_HOME` for that kind) |
| `BOOKMAKERSREVIEW_CONFIG` | Point at a specific config file path instead of the resolved config directory's `config.json` |
| `BOOKMAKERSREVIEW_BASE_URL` | Override the BMR GraphQL base URL (useful for pointing at a mock/test server) |
| `BOOKMAKERSREVIEW_NO_LEARN` | Set to `true` to disable the teach/recall learning loop CLI-wide (same as `--no-learn` per invocation) |
| `BOOKMAKERSREVIEW_FEEDBACK_ENDPOINT` | Remote URL for `feedback --send`/auto-send to POST captured feedback to, in addition to the local `feedback.jsonl` |
| `BOOKMAKERSREVIEW_FEEDBACK_AUTO_SEND` | Set to `true` to always POST feedback to the endpoint above without passing `--send` |

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **`sports` (the raw GraphQL field, e.g. via the `graphql` passthrough command) returns nothing or errors** — the upstream top-level `sports` field is a known broken federation endpoint on BMR's side; use the `sports list` command instead, which queries the working `getSportsWithSettingsV2` field, or `leagues list`, which carries the sport id (`spid`) per league
- **`odds value` / `arb scan` / `odds movement` report no lines/history for a market** — this CLI has no local sync step; every query hits BMR's live GraphQL feed directly. An empty result means BMR hasn't posted lines for that event/market yet (common for games several days out) or the market type doesn't apply to that sport. Confirm data exists first with `odds current --event <id> --market <id> --books <ids>` or `consensus current --event <id> --market <id>`, and try again closer to game time
- **A query returns an empty array with no error** — double-check the numeric id you passed (event/league/market-type ids are BMR-internal integers); use the matching `list` command with `--select` to look the id up by name first
