// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package bmr

// League is the "leagues" resource. spid links to the parent sport; the
// top-level "sports" query field is a broken federation passthrough on
// BookmakersReview's own backend (confirmed live: it unconditionally errors
// demanding internal sitid/did context), so sport identity is resolved
// through League.SpID/League.Sport instead of a direct sport lookup.
type League struct {
	LID  int    `json:"lid"`
	Name string `json:"nam"`
	SpID int    `json:"spid"`
}

// Sportsbook is one entry from the sitid/did-scoped "sportsbooks" catalog.
// PAID (Provider Account Id) is the id used to filter/tag odds lines
// elsewhere in the schema (Line.PAID); SBID is a separate internal id.
type Sportsbook struct {
	SBID int    `json:"sbid"`
	PAID int    `json:"paid"`
	Name string `json:"nam"`
}

// MarketType describes one bettable market (moneyline, spread, total, props,
// ...). mtid 1/2/3 recur across every sport as "Winner" (moneyline-shaped,
// 1/x/2), "Total" (over/under), and "Handicap" (spread-shaped, 1/x/2) —
// confirmed live across spid 2 (soccer), 3 (baseball), 4 (football), 5
// (basketball), 6 (hockey). Everything else is sport- and market-specific.
type MarketType struct {
	MTID int    `json:"mtid"`
	Name string `json:"nam"`
	Des  string `json:"des"`
	SpID int    `json:"spid"`
}

// Line is the shared shape returned by currentLines, openingLines,
// bestLines, historyLines, and lineHistory. These four query fields declare
// their GraphQL return type as an opaque JSON scalar (not an introspectable
// object), so this struct was reverse-engineered from live responses rather
// than the schema. Confirmed fields; anything not listed here was not
// observed in sampled responses.
//
// Price is the decimal price (e.g. 2.45); American is the American-odds
// equivalent (e.g. 145). Delta is the fractional change from the previous
// snapshot (e.g. -0.0345 = price dropped ~3.45%). BOID identifies which
// selection/outcome this line is for (see EventBettingOption for the
// human-readable name); PAID identifies which sportsbook posted it.
type Line struct {
	MTID     int     `json:"mtid"`
	EID      int     `json:"eid"`
	BOID     int     `json:"boid"`
	PAID     int     `json:"paid"`
	SBID     int     `json:"sbid,omitempty"`
	Time     string  `json:"tim"`
	PartID   int     `json:"partid"`
	LineID   string  `json:"lineid"`
	Sequence int64   `json:"sequence"`
	Adj      float64 `json:"adj"`
	Price    float64 `json:"pri"`
	Delta    float64 `json:"dp"`
	American int     `json:"ap"`
	FPD      int     `json:"fpd"`
	FPN      int     `json:"fpn"`
	EntrID   *int    `json:"entrid"`
	MTGroup  string  `json:"mtgrp"`
	TeamID   *int    `json:"tmid"`
	Sort     string  `json:"sort,omitempty"` // bestLines only
}

// EventBettingOption names one selection/outcome (BOID) within an event's
// market — the human-readable label for Line.BOID.
type EventBettingOption struct {
	EID    int    `json:"eid"`
	PartID int    `json:"partid"`
	BOID   int    `json:"boid"`
	Name   string `json:"nam"`
	MTID   int    `json:"mtid"`
	SpID   int    `json:"spid"`
}

// Consensus is the typed (non-JSON-scalar) consensus/fair-value line shape.
// Perc is the vig-free consensus percentage/probability; Line is the
// consensus price point (spread/total number) where applicable.
type Consensus struct {
	EID      int     `json:"eid"`
	MTID     int     `json:"mtid"`
	BOID     int     `json:"boid"`
	PartID   int     `json:"partid"`
	SBID     int     `json:"sbid,omitempty"`
	PAID     int     `json:"paid,omitempty"`
	LineID   string  `json:"lineid,omitempty"`
	Wager    float64 `json:"wag,omitempty"`
	Perc     float64 `json:"perc"`
	Vol      float64 `json:"vol,omitempty"`
	TVol     float64 `json:"tvol,omitempty"`
	Sequence int64   `json:"sequence"`
	// Time is milliseconds since epoch as a JSON number here, unlike Line.Time
	// (a numeric string like "1788285180000.000000") — confirmed live; the two
	// families of timestamp encoding are genuinely inconsistent upstream.
	Time float64 `json:"tim,omitempty"`
	WB   float64 `json:"wb,omitempty"`
}

// EventScore is one participant-period score row from Event.scores. PN is
// the period number (1-4 = quarters for football, etc.; higher numbers may
// represent overtime periods depending on sport).
type EventScore struct {
	EID    int    `json:"eid"`
	PartID int    `json:"partid"`
	PN     int    `json:"pn"`
	Value  string `json:"val"`
}

// Event is the shared event/game shape used by events, eventsByDateNew, and
// upcomingEvents. DT is milliseconds since epoch (confirmed live: eid=1's dt
// of 1249862400000 decodes to August 2009, the oldest indexed NFL event).
// STA/ST are venue state and playing-surface strings (e.g. "Pennsylvania",
// "Grass") despite the terse names suggesting "status" — do not treat them
// as event status fields.
type Event struct {
	EID     int    `json:"eid"`
	DT      int64  `json:"dt"`
	LID     int    `json:"lid,omitempty"`
	SpID    int    `json:"spid,omitempty"`
	State   string `json:"sta,omitempty"`
	Surface string `json:"st,omitempty"`
	League  *struct {
		Name string `json:"nam"`
	} `json:"league,omitempty"`
	Scores []EventScore `json:"scores,omitempty"`
}
