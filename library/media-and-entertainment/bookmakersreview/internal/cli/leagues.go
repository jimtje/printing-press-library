// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newLeaguesCmd(flags))
	})
}

func newLeaguesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "leagues",
		Short:       "leagues subcommands: list",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newLeaguesListCmd(flags))
	return cmd
}

func parseIntCSV(s string) ([]int, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// intLiteralList renders a Go int slice as a GraphQL list literal, e.g.
// "[1, 2, 3]". Numeric formatting only — no injection risk.
func intLiteralList(ints []int) string {
	if len(ints) == 0 {
		return "[]"
	}
	parts := make([]string, len(ints))
	for i, n := range ints {
		parts[i] = strconv.Itoa(n)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// stringLiteral JSON-encodes s for safe inline interpolation into a GraphQL
// query string (used for the handful of upstream args that are declared as
// String rather than Int, e.g. marketTypes' sitid/did).
func stringLiteral(s string) string {
	return strconv.Quote(s)
}

func newLeaguesListCmd(flags *rootFlags) *cobra.Command {
	var flagSport string
	var flagEnabled bool
	var flagLimit int

	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List leagues, optionally filtered by sport id",
		Long:        "List leagues. Filter by sport id (spid) — see 'sports list' for the sport id catalog. The upstream top-level 'sports' field is a broken federation passthrough on BookmakersReview's own backend, so 'sports list' uses getSportsWithSettingsV2 instead.",
		Example:     "  bookmakersreview-pp-cli leagues list --sport 4 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			spids, err := parseIntCSV(flagSport)
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
				Leagues []bmr.League `json:"leagues"`
			}
			// NOTE: literal values are inlined directly into the query
			// string rather than passed via GraphQL $variables. The outer
			// federation gateway and the inner backend service disagree on
			// several argument types (Int vs [Int]) across this schema;
			// named variables trigger contradictory type-check errors
			// between the two layers, while literal-argument coercion is
			// lenient enough to work. Confirmed live.
			// spid is declared LIST/Int at the outer gateway but the real
			// backend rejects a list literal here ("Expected type Int, found
			// [4]") — confirmed live. Pass it as a bare scalar; only the
			// first --sport value is honored if more than one is given,
			// since the backend has no multi-value form for this field.
			spidArg := ""
			if len(spids) > 0 {
				spidArg = fmt.Sprintf(", spid: %d", spids[0])
			}
			query := fmt.Sprintf(`query { leagues(enabled: %t, limit: %d%s) { lid nam spid } }`, flagEnabled, flagLimit, spidArg)
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if result.Leagues == nil {
				result.Leagues = make([]bmr.League, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result.Leagues, flags)
			}
			for _, l := range result.Leagues {
				cmd.Printf("%d\t%s\t(sport %d)\n", l.LID, l.Name, l.SpID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagSport, "sport", "", "Filter by sport id(s), comma-separated (e.g. 4 for American football)")
	cmd.Flags().BoolVar(&flagEnabled, "enabled", true, "Only include enabled leagues")
	cmd.Flags().IntVar(&flagLimit, "limit", 200, "Maximum leagues to return (upstream may cap this lower)")
	return cmd
}
