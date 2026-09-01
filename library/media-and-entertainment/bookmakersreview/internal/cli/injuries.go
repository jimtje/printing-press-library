// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newInjuriesCmd(flags))
	})
}

type injuryRow struct {
	PID      int    `json:"pid"`
	TmID     int    `json:"tmid"`
	Typ      string `json:"typ"`
	Stt      string `json:"stt"`
	Loc      string `json:"loc"`
	Det      string `json:"det"`
	NewsDate string `json:"newsdate"`
}

func newInjuriesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "injuries",
		Short:       "injuries subcommands: list",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newInjuriesListCmd(flags))
	return cmd
}

func newInjuriesListCmd(flags *rootFlags) *cobra.Command {
	var flagTeam int
	var flagSeason int
	var flagPriorityOnly bool
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List injury reports for a team",
		Long: "List injury reports for a team. --season is a required upstream season id (seid) — " +
			"confirmed live via GraphQL schema introspection as Int!, not optional despite reading like " +
			"a filter. This CLI has no 'seasons list' lookup; the season id is typically visible in " +
			"upstream URLs/payloads that reference a specific season.",
		Example:     "  bookmakersreview-pp-cli injuries list --team 1535 --season 2025 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("team") || !cmd.Flags().Changed("season") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--team and --season are both required"))
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			query := fmt.Sprintf(`query { injuries(tmid: %s, seid: %d, isPriority: %t, limit: %d) { pid tmid typ stt loc det newsdate } }`,
				intLiteralList([]int{flagTeam}), flagSeason, flagPriorityOnly, flagLimit)
			var result struct {
				Injuries []injuryRow `json:"injuries"`
			}
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if result.Injuries == nil {
				result.Injuries = make([]injuryRow, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result.Injuries, flags)
			}
			for _, i := range result.Injuries {
				cmd.Printf("player %d\t%s (%s)\t%s\t%s\n", i.PID, i.Typ, i.Stt, i.Loc, i.Det)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagTeam, "team", 0, "Team id (required)")
	cmd.Flags().IntVar(&flagSeason, "season", 0, "Season id, seid (required by the upstream API)")
	cmd.Flags().BoolVar(&flagPriorityOnly, "priority-only", false, "Only include high-priority injuries")
	cmd.Flags().IntVar(&flagLimit, "limit", 50, "Maximum injury reports to return")
	return cmd
}
