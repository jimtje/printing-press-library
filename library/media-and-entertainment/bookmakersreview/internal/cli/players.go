// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newPlayersCmd(flags))
	})
}

type playerDetailRow struct {
	PID    int    `json:"pid"`
	FN     string `json:"fn"`
	LNam   string `json:"lnam"`
	City   string `json:"cit"`
	St     string `json:"sta"`
	Cou    string `json:"cou"`
	Age    int    `json:"age"`
	Height string `json:"hei"`
	Weight string `json:"wei"`
	School string `json:"sch"`
}

func newPlayersCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "players",
		Short:       "players subcommands: get",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newPlayersGetCmd(flags))
	return cmd
}

func newPlayersGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <pid>",
		Short: "Get one player by id",
		Long: "Get one player by id. The upstream 'players' query requires a pid and has no working name-search " +
			"or team-roster mode (confirmed live) — this CLI has no way to look up a player id from a name.",
		Example:     "  bookmakersreview-pp-cli players get 12345 --json",
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
				return usageErr(fmt.Errorf("pid argument is required"))
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("invalid pid %q: %w", args[0], err))
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			query := fmt.Sprintf(`query { players(pid: %s) { pid fn lnam cit sta cou age hei wei sch } }`, intLiteralList([]int{pid}))
			var result struct {
				Players []playerDetailRow `json:"players"`
			}
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if len(result.Players) == 0 {
				return notFoundErr(fmt.Errorf("no player found with pid %d", pid))
			}
			p := result.Players[0]
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), p, flags)
			}
			cmd.Printf("%d\t%s %s\t%s, %s %s\t(age %d, %s/%s, %s)\n", p.PID, p.FN, p.LNam, p.City, p.St, p.Cou, p.Age, p.Height, p.Weight, p.School)
			return nil
		},
	}
	return cmd
}
