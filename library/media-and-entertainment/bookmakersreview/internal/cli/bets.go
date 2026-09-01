// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelBetsCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "bets",
		Short:       "bets subcommands: grade, record, report",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newNovelBetsGradeCmd(flags))
	cmd.AddCommand(newNovelBetsRecordCmd(flags))
	cmd.AddCommand(newNovelBetsReportCmd(flags))
	return cmd
}
