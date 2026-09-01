// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"
)

// defaultBookPAIDs is the deduplicated set of Provider Account Ids (paid)
// across BookmakersReview's own 26-sportsbook catalog (sitid=5, did=1,
// confirmed live). Used as the default book set for commands that compare
// across every tracked sportsbook (odds best, odds value, arb scan, steam
// scan) when the caller doesn't narrow with --books.
var defaultBookPAIDs = []int{3, 4, 5, 8, 9, 10, 15, 16, 18, 20, 22, 28, 29, 35, 38, 44, 54, 65, 82, 83, 84}

// intListLiteral renders a Go int slice as a GraphQL list literal, e.g.
// "[1, 2, 3]". Every query in this file inlines literal values directly
// into the query string instead of using named GraphQL variables: the
// upstream federation gateway and its inner "lines"/"lookups" backend
// services disagree on whether several arguments (mtid on
// bettingOptionsByEvent, at least) are scalar Int or [Int] — using named
// variables makes the two layers flip-flop contradictory type-mismatch
// errors no matter which type is declared. Inline literals sidestep this:
// GraphQL's literal-argument coercion (auto-wrapping a scalar into a
// single-element list when needed) is far more lenient than its
// variable-type-checking. Confirmed live during this build. All values
// interpolated here are ints/floats we constructed ourselves (ids, limits)
// — never raw user free-text — so there is no injection concern.
func intListLiteral(vals []int) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func parseCSVInts(csv string) ([]int, error) {
	fields := strings.Split(csv, ",")
	out := make([]int, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		v, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q: %w", f, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// bettingOption is the shape of one bettingOptionsByEvent row: the
// human-readable name (team, "Draw", "Over 45.5", etc.) behind a boid.
type bettingOption struct {
	BOID   int    `json:"boid"`
	Name   string `json:"nam"`
	PartID int    `json:"partid"`
}

// resolveBettingOptionNames maps boid -> selection name for one event and
// market, so raw Line.BOID values can be rendered as "Birmingham City" /
// "Draw" / "Southampton FC" instead of an opaque integer.
func resolveBettingOptionNames(ctx context.Context, c *bmr.Client, eid, mtid int) (map[int]string, error) {
	query := fmt.Sprintf(`query { bettingOptionsByEvent(eid: %d, mtid: %d) { boid nam partid } }`, eid, mtid)
	var resp struct {
		Options []bettingOption `json:"bettingOptionsByEvent"`
	}
	if err := c.Query(ctx, query, nil, &resp); err != nil {
		return nil, err
	}
	names := make(map[int]string, len(resp.Options))
	for _, o := range resp.Options {
		names[o.BOID] = o.Name
	}
	return names, nil
}

type sportsbookName struct {
	PAID int    `json:"paid"`
	Name string `json:"nam"`
}

// resolveSportsbookNames maps paid (Provider Account Id) -> sportsbook name
// for the given site/domain scope, so raw Line.PAID values can be rendered
// as "Bovada" instead of an opaque integer.
func resolveSportsbookNames(ctx context.Context, c *bmr.Client, siteID, domainID int) (map[int]string, error) {
	query := fmt.Sprintf(`query { sportsbooks(sitid: %d, did: %d, enabled: true) { paid nam } }`, siteID, domainID)
	var resp struct {
		Books []sportsbookName `json:"sportsbooks"`
	}
	if err := c.Query(ctx, query, nil, &resp); err != nil {
		return nil, err
	}
	names := make(map[int]string, len(resp.Books))
	for _, b := range resp.Books {
		// Multiple sbid "skins" can share one paid (provider account);
		// keep the first name seen for that paid.
		if _, ok := names[b.PAID]; !ok {
			names[b.PAID] = b.Name
		}
	}
	return names, nil
}

// enrichedLine is the user-facing rendering of a bmr.Line: cryptic ids
// resolved to names, prices exposed in both decimal and American form.
type enrichedLine struct {
	Selection string  `json:"selection"`
	Book      string  `json:"book"`
	Price     float64 `json:"price"`
	American  int     `json:"american"`
	Delta     float64 `json:"delta_pct,omitempty"`
	Time      string  `json:"time,omitempty"`
	BOID      int     `json:"boid"`
	PAID      int     `json:"paid"`
}

func enrichLines(lines []bmr.Line, boidNames, paidNames map[int]string) []enrichedLine {
	out := make([]enrichedLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, enrichedLine{
			Selection: boidNames[l.BOID],
			Book:      paidNames[l.PAID],
			Price:     l.Price,
			American:  l.American,
			Delta:     l.Delta,
			Time:      l.Time,
			BOID:      l.BOID,
			PAID:      l.PAID,
		})
	}
	return out
}

// impliedProbability converts a decimal price to its implied probability
// (e.g. 2.0 -> 0.5). Used by odds value / arb scan for vig math.
func impliedProbability(decimalPrice float64) float64 {
	if decimalPrice <= 0 {
		return 0
	}
	return 1 / decimalPrice
}
