// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newSportsbooksCmd(flags))
	})
}

// defaultBMRSportsbookIDs are the sbid values for BookmakersReview's own
// 26-book catalog (sitid=5, did=1), confirmed live via "sportsbooks list".
// Used as the default book set for queries whose sbid argument is required
// (e.g. propLinesByEntry) when the caller doesn't override --books.
var defaultBMRSportsbookIDs = []int{
	19, 23, 43, 118, 1252, 1096, 1389, 93, 1618, 1602,
	169, 180, 442, 186, 1275, 423, 626, 1680, 238, 274,
	279, 289, 300, 227, 414, 139,
}

func newSportsbooksCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "sportsbooks",
		Short:       "sportsbooks subcommands: list",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSportsbooksListCmd(flags))
	return cmd
}

func newSportsbooksListCmd(flags *rootFlags) *cobra.Command {
	var flagSiteID int
	var flagDomainID int
	var flagEnabled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sportsbooks in BookmakersReview's own catalog",
		Long: "List sportsbooks. Defaults to sitid=5/did=1 — confirmed live to be BookmakersReview's own " +
			"26-book offshore catalog (Bovada, MyBookie, BetOnline, Pinnacle, ...), distinct from sibling " +
			"Better Collective properties sharing this backend (sitid 1-4 return 256-535 books including " +
			"state-licensed variants like DraftKingsNJ). Override --site-id/--domain-id to see a sibling " +
			"property's catalog.",
		Example:     "  bookmakersreview-pp-cli sportsbooks list --json",
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

			var result struct {
				Sportsbooks []bmr.Sportsbook `json:"sportsbooks"`
			}
			// Literal values inlined — see leagues.go's comment on why
			// named GraphQL $variables are avoided against this backend.
			query := fmt.Sprintf(`query { sportsbooks(sitid: %d, did: %d, enabled: %t) { sbid paid nam } }`,
				flagSiteID, flagDomainID, flagEnabled)
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if result.Sportsbooks == nil {
				result.Sportsbooks = make([]bmr.Sportsbook, 0)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), result.Sportsbooks, flags)
			}
			for _, s := range result.Sportsbooks {
				cmd.Printf("%d\t%s\t(book id %d)\n", s.PAID, s.Name, s.SBID)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagSiteID, "site-id", bmr.DefaultSiteID, "Site scope id (5 = BookmakersReview's own offshore catalog)")
	cmd.Flags().IntVar(&flagDomainID, "domain-id", bmr.DefaultDomainID, "Domain scope id (affiliate/tracking-link scope; does not change the book catalog at site-id=5)")
	cmd.Flags().BoolVar(&flagEnabled, "enabled", true, "Only include enabled sportsbooks")
	return cmd
}
