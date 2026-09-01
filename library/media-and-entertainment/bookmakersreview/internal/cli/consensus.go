// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newConsensusCmd(flags))
	})
}

func newConsensusCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "consensus",
		Short:       "consensus subcommands: current, history",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newConsensusCurrentCmd(flags))
	cmd.AddCommand(newConsensusHistoryCmd(flags))
	return cmd
}

type consensusView struct {
	Selection string  `json:"selection"`
	Percent   float64 `json:"consensus_pct"`
	Volume    float64 `json:"volume,omitempty"`
	Time      float64 `json:"time,omitempty"`
	BOID      int     `json:"boid"`
}

func enrichConsensus(rows []bmr.Consensus, boidNames map[int]string) []consensusView {
	out := make([]consensusView, 0, len(rows))
	for _, r := range rows {
		out = append(out, consensusView{
			Selection: boidNames[r.BOID],
			Percent:   r.Perc,
			Volume:    r.Vol,
			Time:      r.Time,
			BOID:      r.BOID,
		})
	}
	return out
}

func newConsensusCurrentCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int

	cmd := &cobra.Command{
		Use:         "current",
		Short:       "Current vig-free consensus (fair-value) percentages for one event and market",
		Example:     "  bookmakersreview-pp-cli consensus current --event 4802244 --market 1 --json",
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
			query := fmt.Sprintf(`query { consensus(eid: [%d], mtid: [%d]) { eid mtid boid partid perc vol sequence tim } }`, flagEvent, flagMarket)
			var resp struct {
				Consensus []bmr.Consensus `json:"consensus"`
			}
			if err := c.Query(ctx, query, nil, &resp); err != nil {
				return apiErr(err)
			}
			boidNames, _ := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
			return printJSONFiltered(cmd.OutOrStdout(), enrichConsensus(resp.Consensus, boidNames), flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	return cmd
}

func newConsensusHistoryCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int

	cmd := &cobra.Command{
		Use:         "history",
		Short:       "Consensus percentage history for one event and market",
		Long:        "Use this to see how the fair-value consensus moved over time for a specific event+market. For a whole-slate sharp/steam scan, use 'steam scan' instead.",
		Example:     "  bookmakersreview-pp-cli consensus history --event 4802244 --market 1 --json",
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
			query := fmt.Sprintf(`query { consensusHistory(eid: %d, mtid: %d) { eid mtid boid partid perc vol sequence tim } }`, flagEvent, flagMarket)
			var resp struct {
				Consensus []bmr.Consensus `json:"consensusHistory"`
			}
			if err := c.Query(ctx, query, nil, &resp); err != nil {
				return apiErr(err)
			}
			boidNames, _ := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
			return printJSONFiltered(cmd.OutOrStdout(), enrichConsensus(resp.Consensus, boidNames), flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	return cmd
}
