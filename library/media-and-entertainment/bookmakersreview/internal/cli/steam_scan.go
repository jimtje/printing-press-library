// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/cliutil"

	"github.com/spf13/cobra"
)

type steamRow struct {
	EventID         int     `json:"event_id"`
	Selection       string  `json:"selection"`
	OpenPct         float64 `json:"open_pct"`
	CurrentPct      float64 `json:"current_pct"`
	MovePct         float64 `json:"move_pct"`
	SamplesInWindow int     `json:"samples_in_window"`
}

type steamScanResult struct {
	Signals       []steamRow `json:"signals"`
	ScannedEvents int        `json:"scanned_events"`
	FailedEvents  int        `json:"failed_events,omitempty"`
	NoDataEvents  int        `json:"no_data_events,omitempty"`
	Note          string     `json:"note,omitempty"`
}

func newNovelSteamScanCmd(flags *rootFlags) *cobra.Command {
	var flagLeague int
	var flagSince string
	var flagMarket int
	var flagMaxScanEvents int
	var flagMinMove float64

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan today's whole slate for fast, coordinated line moves that signal sharp money before the market fully reacts.",
		Long: "Scans today's upcoming events for one league, pulls each event's consensus history for the given " +
			"market, and flags selections whose consensus percentage moved fast and far within the lookback window " +
			"(a steam signal). Use this to scan a whole day's slate. Do NOT use it to inspect one event's full price " +
			"history over time; use 'odds movement' for that, or 'consensus history' for raw deltas.",
		Example:     "  bookmakersreview-pp-cli steam scan --league 16 --since 3h --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("league") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--league is required"))
			}
			since, err := cliutil.ParseDurationLoose(flagSince)
			if err != nil {
				return usageErr(fmt.Errorf("invalid --since duration %q: %w", flagSince, err))
			}
			maxScan := flagMaxScanEvents
			if cliutil.IsDogfoodEnv() && maxScan > 3 {
				maxScan = 3
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}

			nowMs := time.Now().UnixMilli()
			hoursRange := 48
			eventsQuery := fmt.Sprintf(`query { eventsByDateNew(startDate: %d, hoursRange: %d, lid: [%d], limit: %d) { events { eid } } }`, nowMs, hoursRange, flagLeague, maxScan)
			var eventsResp struct {
				EventsByDateNew struct {
					Events []bmr.Event `json:"events"`
				} `json:"eventsByDateNew"`
			}
			if err := c.Query(ctx, eventsQuery, nil, &eventsResp); err != nil {
				return apiErr(fmt.Errorf("listing events for league %d: %w", flagLeague, err))
			}
			events := eventsResp.EventsByDateNew.Events
			scanned := 0
			failed := 0
			noData := 0
			rows := make([]steamRow, 0)
			cutoff := time.Now().Add(-since)

			for _, ev := range events {
				if scanned >= maxScan {
					break
				}
				scanned++
				// sortBy/order are declared as optional args on consensusHistory
				// but confirmed live to break the backend ("sort.by.map is not
				// a function") whenever set to any value — omit them and sort
				// client-side instead.
				historyQuery := fmt.Sprintf(`query { consensusHistory(eid: %d, mtid: %d) { boid perc tim } }`, ev.EID, flagMarket)
				var histResp struct {
					History []bmr.Consensus `json:"consensusHistory"`
				}
				if err := c.Query(ctx, historyQuery, nil, &histResp); err != nil {
					// A single event's history failing (no market posted yet,
					// event cancelled, etc.) should not abort the whole scan,
					// but the failure count is surfaced in the result so an
					// all-failures run doesn't look identical to a genuine
					// all-quiet scan.
					failed++
					continue
				}
				if len(histResp.History) == 0 {
					// A successful call with zero rows is NOT the same as "no
					// steam move" — it means no consensus data exists yet for
					// this event+market (too far out, market not posted).
					// Counting it as a clean "quiet" scan would hide a
					// missing-data condition behind a "0 signals" result.
					noData++
					continue
				}
				byBoid := map[int][]bmr.Consensus{}
				for _, h := range histResp.History {
					byBoid[h.BOID] = append(byBoid[h.BOID], h)
				}
				for boid, samples := range byBoid {
					inWindow := make([]bmr.Consensus, 0, len(samples))
					for _, s := range samples {
						if time.UnixMilli(int64(s.Time)).After(cutoff) {
							inWindow = append(inWindow, s)
						}
					}
					if len(inWindow) < 2 {
						continue
					}
					// consensusHistory is not guaranteed to arrive in any
					// particular order once sortBy/order are omitted (see
					// comment above) — sort ascending by time ourselves so
					// index 0 is genuinely the oldest sample in the window.
					sort.Slice(inWindow, func(i, j int) bool { return inWindow[i].Time < inWindow[j].Time })
					openPct := inWindow[0].Perc
					currentPct := inWindow[len(inWindow)-1].Perc
					move := currentPct - openPct
					if move < 0 {
						move = -move
					}
					if move < flagMinMove {
						continue
					}
					boidNames, _ := resolveBettingOptionNames(ctx, c, ev.EID, flagMarket)
					rows = append(rows, steamRow{
						EventID:         ev.EID,
						Selection:       boidNames[boid],
						OpenPct:         openPct,
						CurrentPct:      currentPct,
						MovePct:         currentPct - openPct,
						SamplesInWindow: len(inWindow),
					})
				}
			}
			sort.Slice(rows, func(i, j int) bool {
				a, b := rows[i].MovePct, rows[j].MovePct
				if a < 0 {
					a = -a
				}
				if b < 0 {
					b = -b
				}
				return a > b
			})

			result := steamScanResult{Signals: rows, ScannedEvents: scanned, FailedEvents: failed, NoDataEvents: noData}
			if len(rows) == 0 {
				switch {
				case scanned > 0 && failed+noData == scanned:
					result.Note = fmt.Sprintf("all %d scanned events returned either an error (%d) or zero consensus history (%d) — this is a missing-data condition, not a confirmed-quiet market", scanned, failed, noData)
				case noData > 0:
					result.Note = fmt.Sprintf("scanned %d events (%d with no consensus data yet, %d failed) without finding a move >= %.1f pts among the %d events that had data; lower --min-move, widen --since, or re-check closer to game time", scanned, noData, failed, flagMinMove, scanned-failed-noData)
				default:
					result.Note = fmt.Sprintf("scanned %d events (%d failed) without finding a move >= %.1f pts; lower --min-move or widen --since to search further", scanned, failed, flagMinMove)
				}
			}

			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), result.Note)
				return nil
			}
			for _, r := range rows {
				fmt.Fprintf(cmd.OutOrStdout(), "event %d: %s moved %.1f -> %.1f%% (%+0.1f pts)\n", r.EventID, r.Selection, r.OpenPct, r.CurrentPct, r.MovePct)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagLeague, "league", 0, "League id to scan (see 'leagues list'; e.g. 16=NFL, 5=NBA)")
	cmd.Flags().StringVar(&flagSince, "since", "3h", "Lookback window for the consensus move (e.g. 3h, 30m, 1d)")
	cmd.Flags().IntVar(&flagMarket, "market", 1, "Market type id to scan (default 1 = moneyline/Winner)")
	cmd.Flags().IntVar(&flagMaxScanEvents, "max-scan-events", 15, "Maximum events to scan (bounds live API calls; live scan does one GraphQL call per event)")
	cmd.Flags().Float64Var(&flagMinMove, "min-move", 5.0, "Minimum consensus percentage-point move to report as a steam signal")
	return cmd
}
