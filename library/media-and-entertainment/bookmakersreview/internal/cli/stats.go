// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newStatsCmd(flags))
	})
}

type statRow struct {
	EID  int     `json:"eid"`
	Ent  string  `json:"ent"`
	Grp  string  `json:"grp"`
	Stat string  `json:"stat"`
	Val  string  `json:"val"`
	Tim  float64 `json:"tim"`
}

func newStatsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "stats",
		Short:       "stats subcommands: list",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newStatsListCmd(flags))
	return cmd
}

func newStatsListCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int

	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List team/player statistics for an event",
		Example:     "  bookmakersreview-pp-cli stats list --event 4802244 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event is required"))
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			query := fmt.Sprintf(`query { statisticsByEvent(eids: %s) { eid ent grp stat val tim } }`,
				intLiteralList([]int{flagEvent}))
			var result struct {
				StatisticsByEvent []statRow `json:"statisticsByEvent"`
			}
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if result.StatisticsByEvent == nil {
				result.StatisticsByEvent = make([]statRow, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result.StatisticsByEvent, flags)
			}
			for _, s := range result.StatisticsByEvent {
				cmd.Printf("%s / %s: %s\n", s.Grp, s.Stat, s.Val)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (required)")
	return cmd
}
