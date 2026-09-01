// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strconv"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newMarketsCmd(flags))
	})
}

func newMarketsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "markets",
		Short:       "markets subcommands: list",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newMarketsListCmd(flags))
	return cmd
}

func newMarketsListCmd(flags *rootFlags) *cobra.Command {
	var flagSport string
	var flagSiteID int
	var flagDomainID int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List market types, optionally filtered by sport id",
		Long: "List market types (moneyline, spread, total, props, ...). mtid 1/2/3 recur across every " +
			"sport as \"Winner\" (moneyline-shaped, 1/x/2 outcomes), \"Total\" (over/under), and " +
			"\"Handicap\" (spread-shaped, 1/x/2) — confirmed live across soccer, baseball, football, " +
			"basketball, and hockey. Everything else is sport- and market-specific. " +
			"Note: unlike sportsbooks list, this upstream field takes sitid/did as STRINGS, not integers.",
		Example:     "  bookmakersreview-pp-cli markets list --sport 4 --json",
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
				MarketTypes []bmr.MarketType `json:"marketTypes"`
			}
			spidArg := ""
			if len(spids) > 0 {
				spidArg = fmt.Sprintf(", spid: %s", intLiteralList(spids))
			}
			// sitid/did are String here (quoted), unlike sportsbooks (Int)
			// — a genuine upstream inconsistency, confirmed live. Values
			// are inlined literally (see leagues.go comment).
			query := fmt.Sprintf(`query { marketTypes(sitid: %s, did: %s%s) { mtid nam des spid } }`,
				stringLiteral(strconv.Itoa(flagSiteID)), stringLiteral(strconv.Itoa(flagDomainID)), spidArg)
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if result.MarketTypes == nil {
				result.MarketTypes = make([]bmr.MarketType, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result.MarketTypes, flags)
			}
			for _, m := range result.MarketTypes {
				cmd.Printf("%d\t%s\t%s\n", m.MTID, m.Name, m.Des)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagSport, "sport", "", "Filter by sport id(s), comma-separated (e.g. 4 for American football)")
	cmd.Flags().IntVar(&flagSiteID, "site-id", bmr.DefaultSiteID, "Site scope id (5 = BookmakersReview's own catalog)")
	cmd.Flags().IntVar(&flagDomainID, "domain-id", bmr.DefaultDomainID, "Domain scope id")
	return cmd
}
