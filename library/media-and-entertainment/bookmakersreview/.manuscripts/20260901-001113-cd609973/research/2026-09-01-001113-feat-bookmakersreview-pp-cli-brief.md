# BookmakersReview Odds CLI Brief

## API Identity
- Domain: `ms.virginia.us-east-1.bookmakersreview.com/ms-odds-v2/odds-v2-service` — the internal GraphQL microservice backing BookmakersReview's odds-comparison tool (`odds.bookmakersreview.com`). BookmakersReview ("the online sportsbook authority") is a free sportsbook-review/odds-comparison site.
- Auth: **none**. Introspection is open and unauthenticated; real data queries (leagues, sports, events) return live data with no headers or credentials. Confirmed via multiple live GraphQL calls in this session.
- Data profile: Real-time and historical sports betting odds. 216 GraphQL types, 174 top-level Query fields. Covers sports/leagues/teams/players, events/scores, sportsbooks, market types, current/opening/historical lines per sportsbook, consensus (vig-free "fair value") lines and consensus history, live in-play lines, player props, injuries, weather (for outdoor sports), and team/player statistics.
- Field naming is terse but self-documented: 691 field descriptions and 544 argument descriptions exist in the schema (e.g. `eid`="Event Id", `mtid`="Market Type Id", `paid`="Provider Account Id" (sportsbook), `lid`="League Id", `spid`="Sport Id", `boid`="Betting Option Id", `partid`="Participant Id"). These require `flag_name` enrichment before generation per the Public Parameter Name Enrichment step.
- Known quirk: the top-level `sports` field is a broken federation passthrough (`errors.path` shows `["sports","Proxy","sports","sports",...]`) that unconditionally demands internal `sitid`/`did` context the outer schema doesn't expose as usable args. Use `leagues`, `sportsWithLiveEvents`, or `leagueHierarchy` (which take real args) instead — do not build a promoted `sports list` command around the bare `sports` field.
- Confirmed working sample IDs: NFL lid=16, NBA lid=5, MLB lid=3, NHL lid=7, NCAA Football lid=6, WNBA lid=15, UFC lid=26, Premier League lid=2, MLS lid=4.
- `dt` (event date/time) is **milliseconds** since epoch, not seconds — confirmed live (`events(lid:[16], limit:10)` with no filter returned `eid=1, dt=1249862400000` = Aug 2009, the oldest indexed NFL event). Full historical game results are queryable via `events(lid:[...], dt: {between: [fromMs, toMs]})`, including quarter-by-quarter per-participant box scores (`scores { partid val pn }`, where `pn`=period number). This is distinct from `odds history`/`consensus history` (price history) — it is actual final-score/results history, confirmed working back to 2009.
- Several Query fields expose sequence-based incremental sync (`getUpdatedEvents`, `getUpdatedCurrentLines`, `getUpdatedStatistics`, `updatedConsensus` — "get all updated X for event ids greater than sequence") — a native cursor for cheap polling instead of full re-fetch.

## Alternate Entrypoint (confirmed live)
- `https://ms.virginia.us-east-1.oddstrader.com/odds-v2/odds-v2-service` is a sibling hostname (OddsTrader, another Better Collective property) serving the **exact same backend** — confirmed identical `sitid` sportsbook counts (256/434/535/526/26 for sitid 1-5) on both hosts. `sitid` is a global site-profile selector independent of which hostname you hit, not a per-host dataset. Config: use bookmakersreview.com as primary base URL, oddstrader.com as an automatic fallback base URL on connection failure, with `sitid=5, did=1` (BMR's own offshore catalog) kept as the default regardless of which host actually serves the request. Do not expose this as a second combo-CLI source — it is the same schema, same data, just a redundant network path.

## Reachability Risk
- None. Live GraphQL introspection and data queries both succeeded repeatedly with no auth, no rate-limit errors, no bot-protection signals (plain JSON POST responses, `Content-Type: application/json`, ordinary GraphQL error envelopes for bad args).

## Top Workflows
1. **Best-odds shopping** — for a given game/market, find the best available price across every sportsbook BMR tracks (`bestLines`/`bestLinesV2`, `currentLines`).
2. **Line movement / sharp money tracking** — watch how consensus and individual-book lines move from open to now; flag steam (multiple books moving the same direction fast) (`consensus`, `consensusV2`, `consensusHistory`, `updatedConsensus`, `openingLines` vs `currentLines`).
3. **Closing line value (CLV)** — compare the line at bet time to the closing line to grade betting decisions after the fact (`historyLines`/`lineHistory`, `openingLines`).
4. **Historical odds lookup** — full line and consensus history for a given event/market (`historyLines`, `lineHistory`, `consensusHistory`).
5. **General odds + scores lookup** — today's/upcoming games by league/date, live scores, injuries, weather for outdoor games (`eventsByDateNew`, `upcomingEvents`, `scores`, `injuries`, `weather`).

## Table Stakes (vs. The Odds API / OddsJam / Unabated — all paid)
- Multi-book current odds by game and market
- Consensus / vig-free fair-value line
- Line movement history / opening vs. current
- Sport/league/team/player reference data
- Live in-play odds
- Player props

## Data Layer
- Primary entities: `sports`, `leagues`, `teams`, `players`, `events`, `sportsbooks`, `marketTypes`, `lines` (current/opening snapshots), `consensus` (snapshots), `scores`, `injuries`.
- Sync cursor: sequence-based incremental fetch via `getUpdated*` fields (event ids + sequence watermark) — cheaper than full re-sync for lines/consensus/scores/statistics.
- FTS/search: team and player name search locally; local event/line history enables closing-line-value and steam-move detection that BMR's own UI does not surface as a computed metric.

## User Vision
- User explicitly wants all of: best-odds/line-shopping, line-movement/sharp-money tracking, general odds+scores lookup, **and full history** (their exact words: "All of the above and history").
- User also directed brute-forcing `sitid`/`did` (Int args on ~20 site-config-scoped fields: `sportsbooks`, `marketTypes`, `categories`, `menuOptions`, etc.) to find geographically distinct sportsbook catalogs. Confirmed live: `sitid` 1-4 return 256/434/535/526 sportsbooks (mixed offshore + US-state-licensed books, e.g. `DraftKingsNJ`, `FoxBetNJ` — clearly sibling Better Collective properties sharing this backend), while **`sitid=5` returns exactly 26 classic offshore books** (5Dimes, Bookmaker, BetOnline, Bovada, MyBookie, Pinnacle, Heritage Sports, Intertops, JustBet, WagerWeb, YouWager, GTbets, Skybook, JazzSports, The Greek Sportsbook, etc.) with zero US-regulated-book noise — an exact match for BookmakersReview's own "offshore sportsbooks" identity (confirmed via the site's own `odds.bookmakersreview.com/odds/` branding). `did` 1-5 did not change the book count at `sitid=5` (26 in every case), so `did` is most likely an affiliate/tracking-link scope, not a catalog filter. **Default config: `sitid=5, did=1`**, with `--site-id`/`--domain-id` exposed as optional override flags on the ~20 affected commands for power users who want a sibling property's catalog.

## Product Thesis
- Name: BookmakersReview (display), CLI slug `bookmakersreview`.
- Why it should exist: No public/official API exists for BookmakersReview, and no third-party wrapper or CLI for it was found on GitHub/npm/PyPI. Commercial alternatives (The Odds API, OddsJam, Unabated) charge for consensus/line-history/CLV-shaped data that BMR's own backend already serves for free and unauthenticated. This CLI is the only way to script against BMR's odds data, and pairs it with a local SQLite history no paid API gives you by default (most charge extra for historical odds).

## Build Priorities
1. Hand-built GraphQL client (`internal/bmr/`) covering sports/leagues/teams/events/scores, sportsbooks, market types, current/opening/best lines, consensus (+history), player props, injuries, weather, statistics.
2. Local SQLite sync of events + line/consensus snapshots per tracked league/date range, enabling offline search and history-over-time that BMR's own site does not expose as a downloadable dataset.
3. Transcendence: line-shopping across books with implied-vig math, steam-move detection from consensus deltas, closing-line-value grading against a user's own recorded bet, and human-readable resolution of the schema's cryptic ID params (league/sport/sportsbook/market-type name -> id).
