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

// leagueRow is the enriched leagues-list view sourced from
// getLeaguesWithSettingsV2, which returns the full league catalog (no
// pagination cap — confirmed live: 3002 rows) plus an abbreviation and
// region, unlike the plain "leagues" field this replaced (capped at ~5 rows
// regardless of the requested limit — an upstream quirk, not fixable
// client-side).
type leagueRow struct {
	LID        int    `json:"lid"`
	Name       string `json:"nam"`
	Abbrev     string `json:"sn,omitempty"`
	SpID       int    `json:"spid"`
	RegionID   int    `json:"rid,omitempty"`
	RegionName string `json:"region,omitempty"`
}

func newLeaguesListCmd(flags *rootFlags) *cobra.Command {
	var flagSport int
	var flagFeaturedOnly bool
	var flagSearch string
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List leagues, optionally filtered by sport id or name",
		Long: "List leagues via getLeaguesWithSettingsV2 — see 'sports list' for the sport id catalog. " +
			"Note: the upstream 'enabled' field means \"featured on the site\", not \"currently active\" — most " +
			"real, in-season leagues report enabled=false, so this command does not filter on it by default; " +
			"pass --featured-only to restrict to the small featured subset.",
		Example:     "  bookmakersreview-pp-cli leagues list --sport 4 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			// NOTE: literal values are inlined directly into the query
			// string rather than passed via GraphQL $variables — the outer
			// federation gateway and the inner backend service disagree on
			// several argument types (Int vs [Int]) across this schema;
			// named variables trigger contradictory type-check errors
			// between the two layers, while literal-argument coercion is
			// lenient enough to work. Confirmed live.
			// spid is declared LIST/Int at the outer gateway but the real
			// backend rejects a list literal here ("Expected type Int, found
			// [4]") — confirmed live. Pass it as a bare scalar.
			spidArg := ""
			if cmd.Flags().Changed("sport") {
				spidArg = fmt.Sprintf(", spid: %d", flagSport)
			}
			enabledArg := ""
			if flagFeaturedOnly {
				enabledArg = ", enabled: true"
			}
			var raw struct {
				Leagues []struct {
					LID    int    `json:"lid"`
					Name   string `json:"nam"`
					Abbrev string `json:"sn"`
					SpID   int    `json:"spid"`
					RID    int    `json:"rid"`
					Region *struct {
						Name string `json:"nam"`
					} `json:"region"`
				} `json:"getLeaguesWithSettingsV2"`
			}
			query := fmt.Sprintf(`{getLeaguesWithSettingsV2(sitid:%d,did:%d%s%s){lid nam sn spid rid region{nam}}}`,
				bmr.DefaultSiteID, bmr.DefaultDomainID, spidArg, enabledArg)
			if err := c.Query(ctx, query, nil, &raw); err != nil {
				return apiErr(err)
			}

			search := strings.ToLower(strings.TrimSpace(flagSearch))
			results := make([]leagueRow, 0, len(raw.Leagues))
			for _, l := range raw.Leagues {
				if search != "" && !strings.Contains(strings.ToLower(l.Name), search) && !strings.Contains(strings.ToLower(l.Abbrev), search) {
					continue
				}
				row := leagueRow{LID: l.LID, Name: l.Name, Abbrev: l.Abbrev, SpID: l.SpID, RegionID: l.RID}
				if l.Region != nil {
					row.RegionName = l.Region.Name
				}
				results = append(results, row)
				if flagLimit > 0 && len(results) >= flagLimit {
					break
				}
			}

			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), results, flags)
			}
			for _, l := range results {
				cmd.Printf("%d\t%s (%s)\t(sport %d, %s)\n", l.LID, l.Name, l.Abbrev, l.SpID, l.RegionName)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagSport, "sport", 0, "Filter by sport id (see 'sports list')")
	cmd.Flags().BoolVar(&flagFeaturedOnly, "featured-only", false, "Only include leagues BookmakersReview marks as featured (most real leagues are not)")
	cmd.Flags().StringVar(&flagSearch, "search", "", "Case-insensitive substring filter on league name or abbreviation")
	cmd.Flags().IntVar(&flagLimit, "limit", 200, "Maximum leagues to return (0 = no limit; the catalog has ~3000 entries)")
	return cmd
}
