// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newEventsCmd(flags))
	})
}

func newEventsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "events",
		Short:       "events subcommands: list, get, history",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newEventsListCmd(flags))
	cmd.AddCommand(newEventsGetCmd(flags))
	cmd.AddCommand(newEventsHistoryCmd(flags))
	return cmd
}

// parseEventDate accepts YYYY-MM-DD or RFC3339 and returns milliseconds
// since epoch — the wire format for every "dt"-bearing field in this API
// (confirmed live: eid=1's dt of 1249862400000 decodes to August 2009).
func parseEventDate(s string) (int64, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UnixMilli(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UnixMilli(), nil
	}
	return 0, fmt.Errorf("invalid date %q: expected YYYY-MM-DD or RFC3339", s)
}

func newEventsListCmd(flags *rootFlags) *cobra.Command {
	var flagLeague int
	var flagHoursRange int
	var flagLimit int

	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List upcoming events for a league",
		Example:     "  bookmakersreview-pp-cli events list --league 16 --hours-range 168 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("league") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--league is required"))
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			// eventsByDateNew wraps its result in {events, maxSequences} —
			// unlike the plain "events" query used by get/history, which
			// returns a bare list. Values inlined literally (see
			// leagues.go comment on avoiding named $variables here).
			nowMs := time.Now().UnixMilli()
			query := fmt.Sprintf(`query {
				eventsByDateNew(startDate: %d, hoursRange: %d, lid: %s, limit: %d) {
					events { eid dt league { nam } }
				}
			}`, nowMs, flagHoursRange, intLiteralList([]int{flagLeague}), flagLimit)
			var wrapper struct {
				EventsByDateNew struct {
					Events []bmr.Event `json:"events"`
				} `json:"eventsByDateNew"`
			}
			if err := c.Query(ctx, query, nil, &wrapper); err != nil {
				return apiErr(err)
			}
			events := wrapper.EventsByDateNew.Events
			if events == nil {
				events = make([]bmr.Event, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), events, flags)
			}
			for _, e := range events {
				name := ""
				if e.League != nil {
					name = e.League.Name
				}
				cmd.Printf("%d\t%s\t%s\n", e.EID, time.UnixMilli(e.DT).Format(time.RFC3339), name)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagLeague, "league", 0, "League id (e.g. 16 = NFL, 5 = NBA, 3 = MLB, 7 = NHL)")
	cmd.Flags().IntVar(&flagHoursRange, "hours-range", 168, "Hours after now to include events")
	cmd.Flags().IntVar(&flagLimit, "limit", 25, "Maximum events to return")
	return cmd
}

func newEventsGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "get <eid>",
		Short:       "Get one event by id, including per-quarter/period scores",
		Example:     "  bookmakersreview-pp-cli events get 4802244 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("eid argument is required"))
			}
			eid, err := strconv.Atoi(args[0])
			if err != nil {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("invalid eid %q: %w", args[0], err))
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			// The plain "events" query returns a bare list directly — no
			// {events: [...]} wrapper, unlike eventsByDateNew/upcomingEvents.
			var result struct {
				Events []bmr.Event `json:"events"`
			}
			query := fmt.Sprintf(`query {
				events(eid: %s) { eid dt league { nam } scores { partid val pn } }
			}`, intLiteralList([]int{eid}))
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if len(result.Events) == 0 {
				return notFoundErr(fmt.Errorf("no event found with eid %d", eid))
			}
			ev := result.Events[0]
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), ev, flags)
			}
			name := ""
			if ev.League != nil {
				name = ev.League.Name
			}
			cmd.Printf("%d\t%s\t%s\n", ev.EID, time.UnixMilli(ev.DT).Format(time.RFC3339), name)
			for _, s := range ev.Scores {
				cmd.Printf("  participant %d, period %d: %s\n", s.PartID, s.PN, s.Value)
			}
			return nil
		},
	}
	return cmd
}

func newEventsHistoryCmd(flags *rootFlags) *cobra.Command {
	var flagLeague int
	var flagFrom string
	var flagTo string
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Look up historical event results (final/period scores) for a league in a date range",
		Long: "Look up historical event results. Confirmed live: full results are available back to 2009 " +
			"(eid=1), including quarter-by-quarter per-participant box scores. This is game RESULTS " +
			"history, distinct from 'consensus history' (fair-value line history) and 'odds movement' " +
			"(formatted consensus price-movement timeline).",
		Example:     "  bookmakersreview-pp-cli events history --league 16 --from 2025-12-14 --to 2025-12-16 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("league") || flagFrom == "" || flagTo == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--league, --from, and --to are all required"))
			}
			fromMs, err := parseEventDate(flagFrom)
			if err != nil {
				_ = cmd.Usage()
				return usageErr(err)
			}
			toMs, err := parseEventDate(flagTo)
			if err != nil {
				_ = cmd.Usage()
				return usageErr(err)
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			var result struct {
				Events []bmr.Event `json:"events"`
			}
			query := fmt.Sprintf(`query {
				events(lid: %s, dt: {between: [%d, %d]}, limit: %d) {
					eid dt league { nam } scores { partid val pn }
				}
			}`, intLiteralList([]int{flagLeague}), fromMs, toMs, flagLimit)
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if result.Events == nil {
				result.Events = make([]bmr.Event, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result.Events, flags)
			}
			for _, e := range result.Events {
				name := ""
				if e.League != nil {
					name = e.League.Name
				}
				cmd.Printf("%d\t%s\t%s\t(%d score rows)\n", e.EID, time.UnixMilli(e.DT).Format(time.RFC3339), name, len(e.Scores))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagLeague, "league", 0, "League id (e.g. 16 = NFL)")
	cmd.Flags().StringVar(&flagFrom, "from", "", "Start date, YYYY-MM-DD or RFC3339 (required)")
	cmd.Flags().StringVar(&flagTo, "to", "", "End date, YYYY-MM-DD or RFC3339 (required)")
	cmd.Flags().IntVar(&flagLimit, "limit", 50, "Maximum events to return")
	return cmd
}
