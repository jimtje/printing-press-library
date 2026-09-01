## Customer model

**Line-Shopping Lou** — an NFL/NBA bettor with active accounts at 6-8 sportsbooks specifically so he can always place at the best number.
*Today (without this CLI):* Lou has book apps and browser tabs open in parallel, tabbing between them by hand to compare moneyline/spread/total prices for the game he's about to bet, doing vig math in his head as he goes.
*Weekly ritual:* Thursday through Sunday in-season (nightly for NBA/MLB), he scans the upcoming slate an hour or two before first pitch/kickoff and works down his shortlist of games, checking each book's number before locking in a wager.
*Frustration:* By the time he's finished checking all 6-8 books manually, a number has often already moved against him, and there's no quick way to tell if a price like -110 across the board is fair value or if he's just eating extra vig everywhere.

**Sharp Steve** — an advantage bettor who watches for "steam" (fast, coordinated line moves that signal sharp money) so he can get a bet down before the market fully reacts.
*Today (without this CLI):* Steve keeps a browser tab open and refreshes odds pages constantly, and follows Twitter accounts that manually call out steam moves after the fact.
*Weekly ritual:* In the hours before kickoff, he obsessively watches how the consensus and individual-book lines move from the opener, trying to catch the moment several books shift the same direction fast.
*Frustration:* He has no way to script his own steam detector — he either pays for a third-party steam-alert service or relies on eyeballing constant page refreshes, and by the time he notices a move by eye it's often too late to beat it.

**CLV Chris** — a bettor who tracks closing line value (CLV) as his primary skill metric, on the theory that beating the closing number is a better long-run predictor of edge than short-term win rate.
*Today (without this CLI):* Chris logs every bet (event, market, price, book, time) in a personal spreadsheet, then manually looks up what the closing line ended up being, hours or days later, to compute his CLV by hand.
*Weekly ritual:* Every Monday he reconciles the weekend's bet log against the final closing numbers and updates his running CLV% in the spreadsheet.
*Frustration:* Finding "what was the exact closing consensus for that game/market" after the fact means digging through memory, old screenshots, or a UI that doesn't preserve history — there's no tool that ties his own bet log to BMR's closing data automatically.

**Weekend Warrior Wendy** — a casual bettor and fan who checks scores, odds, injuries, and weather once or twice a day to stay informed before making a few recreational picks.
*Today (without this CLI):* Wendy bounces between the BMR website, ESPN, and a weather app to piece together the full picture for outdoor games before deciding on a bet.
*Weekly ritual:* Sunday morning, coffee in hand, she scrolls through the day's slate — odds, injury reports, and (for outdoor games) weather — before texting picks to a group chat.
*Frustration:* No single place shows scores, odds, injuries, and weather together; she has to reassemble the picture herself from multiple sources every time.

## Candidates (pre-cut)

| Candidate | Command | Description | Persona | Source | Long Description |
|---|---|---|---|---|---|
| Odds value (no-vig fair-value / +EV finder) | `odds value` | Computes the vig-free fair line from consensus and flags which book's current price beats fair value (positive expected value) | Line-Shopping Lou | (e) User Vision / Build Priorities transcendence #1 | none |
| Steam move detector | `steam scan` | Scans today's board for rapid, large consensus swings from open (steam) using consensus history deltas | Sharp Steve | (e) User Vision / Build Priorities transcendence #2 | none |
| Bet record | `bets record` | Records a personal bet (event, market, price, book, timestamp) to a local ledger | CLV Chris | (e) User Vision / Build Priorities transcendence #3 | none |
| Bet grade | `bets grade` | Grades a recorded bet's CLV by comparing its price to the market's closing line/consensus | CLV Chris | (e) User Vision / Build Priorities transcendence #3 | none |
| Bet report | `bets report` | Aggregates all recorded/graded bets into running CLV%, win rate vs. CLV, by league/market | CLV Chris | (c) Cross-entity local queries | none |
| Arbitrage scan | `arb scan` | Finds mismatched opposite-side prices across books for an event that guarantee profit regardless of outcome | Line-Shopping Lou | (b) Service-specific content patterns | none |
| Line movement chart | `odds movement` | Renders the full open-to-current price timeline for one event+market across books, with deltas | Sharp Steve / CLV Chris | (c) Cross-entity local queries | none |
| Vig calculator | `odds vig` | Computes implied probability / hold % for a single market's current prices | Line-Shopping Lou | (b) Service-specific content patterns | none |
| Props value shop | `props value` | Same no-vig/EV computation as odds value, applied to player prop lines | Line-Shopping Lou | (a) Persona-driven | none |
| Soft-book report | `books deviation` | Aggregates historical deviation of each sportsbook's prices from consensus to find consistently "soft" books | Sharp Steve | (c) Cross-entity local queries | none |
| Game-day brief | `today brief` | Joins today's events, scores, injuries, and weather for a league into one combined view | Weekend Warrior Wendy | (a) Persona-driven | none |
| ID resolver | `resolve` | Name-to-id lookup across leagues/teams/sportsbooks/market types in one command | Weekend Warrior Wendy | (b) Service-specific content patterns | none |
| Weather impact flag | `weather impact` | Flags outdoor-game totals lines where wind/precipitation exceeds a threshold that historically affects scoring | Weekend Warrior Wendy | (a) Persona-driven | none |
| Auto closing-line snapshot | `bets watch` | Automatically captures consensus/lines at scheduled game start for later CLV grading | CLV Chris | (a) Persona-driven | none |

## Survivors and kills

### Survivors

| # | Feature | Command | Persona Served | Score | Buildability | How It Works | Evidence | Long Description |
|---|---------|---------|-----------------|-------|--------------|--------------|----------|-------------------|
| 1 | No-vig fair-value / +EV line shopping | `odds value` | Line-Shopping Lou | 10/10 | hand-code | Fetches `consensus`/`currentLines` live, strips the vig from consensus to compute a fair probability per market, then compares each book's current price against that fair value to flag positive-EV prices | Named directly in Build Priorities transcendence #1 and User Vision | Use this to find whether a book's current price beats the vig-free fair line (positive EV). Do NOT use it to just find the numerically highest price across books; use `odds best` for that. |
| 2 | Steam move detector | `steam scan` | Sharp Steve | 10/10 | hand-code | Reads local `consensus`/`consensusHistory` synced via `sync --resources consensus --since 24h`, computes rate-of-change per market, flags fast/large moves | Named directly in Build Priorities transcendence #2 and Top Workflow #2 | Use this to scan today's whole board for sharp/steam signals. Do NOT use it to inspect one event's full price history over time; use `odds movement` for that, or `consensus history` for raw deltas. |
| 3 | CLV bet tracking & grading | `bets record` / `bets grade` / `bets report` | CLV Chris | 10/10 | hand-code | `bets record` writes a personal bet to local SQLite; `bets grade` joins that record against `historyLines`/`lineHistory` closing data to compute CLV; `bets report` aggregates the local bets table into running CLV%/win-rate stats | Named directly in Build Priorities transcendence #3 and Top Workflow #3 | Use `bets grade`/`bets report` to evaluate your own recorded bets against the closing line. Do NOT use it for general historical line lookup with no personal bet attached; use `odds history` for that. |
| 4 | Arbitrage scan | `arb scan` | Line-Shopping Lou | 5/10 | hand-code | Fetches `bestLines`/`bestLinesV2` for both sides of a market across books, checks whether combined implied probability of best prices on each side is under 100% | Inferred from line-shopping domain and paid-competitor gap | Use this to find risk-free two-sided arbitrage opportunities across books. Do NOT use it to evaluate single-side value against fair odds; use `odds value` for that. |
| 5 | Line movement chart | `odds movement` | Sharp Steve, CLV Chris | 7/10 | hand-code | Reads locally synced `historyLines`/`lineHistory` for one event+market, renders chronological timeline of price changes per book with deltas from open | Implied by Top Workflow #3 and #4 | Use this to see the full open-to-current price path for one specific event+market. Do NOT use it to scan the whole day's slate for anomalous moves; use `steam scan` for that; do not use for unformatted raw snapshots; use `odds history` for that. |

### Killed candidates

| Feature | Kill reason | Closest surviving sibling |
|---|---|---|
| Vig calculator (`odds vig`) | Subsumed as an intermediate computation inside `odds value` | Odds value |
| Props value shop (`props value`) | Identical computation to `odds value` applied to a different data domain; better as a flag/mode | Odds value |
| Soft-book report (`books deviation`) | Not named in brief; same math as `odds value`, aggregated over time | Odds value |
| Game-day brief (`today brief`) | Fails wrapper-vs-leverage: reproducible by chaining absorbed commands, no real computation | none (absorbed: `events list`) |
| ID resolver (`resolve`) | Already absorbed as "Added Value" on leagues/sportsbooks/markets/teams/players list commands | none (already covered) |
| Weather impact flag (`weather impact`) | Score 5/10 but fails weekly-use/transcendence: one-line manual read of `weather get` output; niche | none (absorbed: `weather get`) |
| Auto closing-line snapshot (`bets watch`) | Scope creep — requires persistent scheduled/background process | Bet grade (`bets grade`) |
