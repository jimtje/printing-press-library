// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

type movementPoint struct {
	Time    string  `json:"time"`
	Percent float64 `json:"consensus_pct"`
}

type movementSeries struct {
	Selection  string          `json:"selection"`
	OpenPct    float64         `json:"open_pct"`
	CurrentPct float64         `json:"current_pct"`
	MovePct    float64         `json:"move_pct"`
	Points     []movementPoint `json:"points"`
}

func newNovelOddsMovementCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int

	cmd := &cobra.Command{
		Use:   "movement",
		Short: "See the full open-to-current consensus timeline for one event and market, across selections.",
		Long: "Renders a formatted open-to-current delta timeline for one event+market from BookmakersReview's " +
			"consensus history. Use this to see how a specific line moved over time. Use 'steam scan' instead to find " +
			"movement across the whole day's slate; use 'consensus history' for the raw unformatted snapshots. " +
			"Note: BookmakersReview's own per-book price-history endpoints (historyLines/lineHistory) are confirmed " +
			"broken upstream, so this command tracks consensus percentage movement rather than a specific book's price.",
		Example:     "  bookmakersreview-pp-cli odds movement --event 4802244 --market 1 --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") || !cmd.Flags().Changed("market") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event and --market are required"))
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}

			query := fmt.Sprintf(`query { consensusHistory(eid: %d, mtid: %d) { boid perc tim } }`, flagEvent, flagMarket)
			var resp struct {
				History []bmr.Consensus `json:"consensusHistory"`
			}
			if err := c.Query(ctx, query, nil, &resp); err != nil {
				return apiErr(fmt.Errorf("fetching consensus history: %w", err))
			}
			if len(resp.History) == 0 {
				return apiErr(fmt.Errorf("no consensus history for event %d market %d", flagEvent, flagMarket))
			}

			byBoid := map[int][]bmr.Consensus{}
			for _, h := range resp.History {
				byBoid[h.BOID] = append(byBoid[h.BOID], h)
			}
			boidNames, _ := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)

			series := make([]movementSeries, 0, len(byBoid))
			for boid, samples := range byBoid {
				sort.Slice(samples, func(i, j int) bool { return samples[i].Time < samples[j].Time })
				points := make([]movementPoint, 0, len(samples))
				for _, s := range samples {
					points = append(points, movementPoint{
						Time:    time.UnixMilli(int64(s.Time)).Format(time.RFC3339),
						Percent: s.Perc,
					})
				}
				series = append(series, movementSeries{
					Selection:  boidNames[boid],
					OpenPct:    samples[0].Perc,
					CurrentPct: samples[len(samples)-1].Perc,
					MovePct:    samples[len(samples)-1].Perc - samples[0].Perc,
					Points:     points,
				})
			}
			sort.Slice(series, func(i, j int) bool { return series[i].Selection < series[j].Selection })

			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), series, flags)
			}
			for _, s := range series {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %.1f%% -> %.1f%% (%+0.1f pts) over %d samples\n", s.Selection, s.OpenPct, s.CurrentPct, s.MovePct, len(s.Points))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	return cmd
}
