# Player ranking inputs and rules

This document describes the current deterministic ranking behavior implemented
by `pkg/recommendation`. It records the inputs and rules that a future
document-driven ranking implementation would need to represent. The code does
not currently read this document; changing it alone does not change rankings.

## Ranking inputs

The ranking uses the following player fields:

| Input | Used for |
| --- | --- |
| `playerId` | Looking up the player's match results in `RankWithResults` and `RankWithTable`. |
| `contestantId` | Grouping players by team and looking up the player's team strength. |
| `contestantName` | Matching the player's team against a league-table entry. |
| `displayName` | Alphabetical tie-break when scores are equal. |
| `position` | Applying defensive scoring for `DEF` and `GK`; selecting the goalkeeper opportunity label for `GK`. |
| `availabilityDisplay` | Excluding players whose availability contains `injured` or `suspended`, ignoring case. |
| `goals`, `assists`, `shotsOnTarget`, `chancesCreated`, `tackles` | All-position score. |
| `yellowCards`, `redCards`, `penaltyMisses`, `ownGoals` | Deductions in the all-position score. |
| `cleanSheet`, `goalsConceded` | Defensive score for defenders and goalkeepers. |
| `saves` | Adds a goalkeeper explanation reason when positive; does not add score. |
| `last3Average` | Added directly when positive; also used in team-strength calculations. |
| `ppmPoints` | Selects the PPM bonus band. |
| `totalPoints` | Used with `last3Average` in fallback team-strength calculations. |
| `nextGameweekFixtures` | Determines fixture count and per-fixture adjustments. |
| `results` | Match stat records used to calculate appearance score. Populated from the `results` argument by `RankWithResults` and `RankWithTable`. |

Each upcoming fixture uses:

| Fixture input | Used for |
| --- | --- |
| `opponentId` | Looking up opponent team strength and determining whether difficulty can be assessed. |
| `opponentName` | Explanation text and league-table team matching. |
| `isHome` | Home-venue adjustment. |
| `gameweek` | Included in the fixture assessment output; does not directly change score. |

Each match result uses its `stats` entries. Entries whose label equals
`Minutes played` case-insensitively contribute to appearance score. The
`total` value must parse as an integer; absent, invalid, zero, or negative
minutes contribute no points.

When supplied to `RankWithTable`, league-table entries use:
`team`, `position`, `points`, `played`, `goalDifference`, `recentForm`, and
`formRate`. `position`, `points`, `played`, `goalDifference`, and `formRate`
contribute to team strength; `recentForm` and `formRate` are also copied into
opponent fixture assessment output.

## Player score

The initial score is the sum of the all-position score, PPM bonus, and
appearance score:

```text
all-position score =
    goals * 6
  + assists * 3
  + shotsOnTarget
  + chancesCreated
  + tackles / 2
  - yellowCards
  - redCards * 3
  - penaltyMisses * 3
  - ownGoals * 2

score = all-position score
      + PPM bonus
      + appearance score
      + last-three average
      + defensive score (DEF and GK only)
      + fixture-count bonus
      + sum of fixture adjustments
```

All calculations use the values as supplied; the final score is not normalized.

### PPM bonus

| `ppmPoints` | Bonus |
| ---: | ---: |
| 12 or more | 5 |
| 8 to less than 12 | 3 |
| 5 to less than 8 | 1 |
| Below 5 | 0 |

### Appearance score

For every match-result stat entry labeled `Minutes played`:

| Minutes | Points |
| ---: | ---: |
| Missing, invalid, zero, or negative | 0 |
| 1 to 59 | 1 |
| 60 or more | 2 |

Each qualifying stat entry is scored independently. A player with multiple
matching entries in a match therefore receives points for each entry.

### Last-three average

Add `last3Average` when it is greater than zero. Zero and negative values add
nothing.

### Defensive score

For `DEF` and `GK` players only:

```text
defensive score = cleanSheet * 5 - max(goalsConceded - 1, 0)
```

Goalkeeper saves do not affect the score under the current rules.

## Team strength and fixture difficulty

Team strengths are built from the complete player input before unavailable
players are filtered out.

### Fallback team strength

For each team:

1. Group players by `contestantId`.
2. Sort the squad by `totalPoints + last3Average`, descending.
3. Keep at most the first 11 players.
4. Use the average `totalPoints + last3Average` of those players as team strength.

### League-table team strength

If a league-table entry matches the team's `contestantName`, its strength
replaces the player-derived fallback:

```text
position score        = (21 - position) / 20 * 100
points-rate score     = points / (played * 3) * 100, or 0 when played is 0
goal-difference score = 50 + goalDifference / played * 10, or 50 when played is 0
                         clamped to 0..100
form score            = formRate * 100

team strength = position score * 0.50
              + points-rate score * 0.15
              + goal-difference score * 0.15
              + form score * 0.20
```

Team matching trims surrounding whitespace, lowercases names, and removes
`" fc"`, `" afc"`, and periods; ampersands are replaced with `"and"`.

For each upcoming fixture:

```text
relative strength = opponent strength - player's team strength
```

| Condition | Difficulty |
| --- | --- |
| Opponent strength is not available | Unknown |
| Relative strength is less than or equal to -5 | Favorable |
| Relative strength is greater than -5 and less than 5 | Even |
| Relative strength is greater than or equal to 5 | Difficult |

### Fixture adjustments

Each fixture contributes the sum of a venue adjustment and a difficulty
adjustment:

| Fixture property | Adjustment |
| --- | ---: |
| Home | +1 |
| Away | 0 |
| Favorable difficulty | +3 |
| Even difficulty | 0 |
| Difficult difficulty | -2 |
| Unknown difficulty | 0 |

The player also receives a fixture-count bonus of `fixture count - 1` when at
least one upcoming fixture exists. Thus one fixture adds 0, two add 1, and so
on. Fixture assessments include opponent strength, relative strength, league
position, and recent form for output, but those informational fields do not
add score separately.

## Eligibility and ranking order

- Players whose lowercased `availabilityDisplay` contains `injured` or
  `suspended` are omitted.
- A limit less than or equal to zero returns an empty result.
- Eligible players are sorted by score descending, then `displayName`
  ascending.
- If the requested limit is greater than the number of eligible players, all
  eligible players are returned.

## Output-only information

These values may affect recommendation descriptions or output but do not
directly add to the numeric score:

- `saves`: adds a reason for a goalkeeper when positive.
- `cleanSheet`: adds a reason when positive, in addition to its defensive
  score for `DEF` and `GK`.
- Fixture `gameweek`, opponent position, and opponent form: included in
  fixture assessments.
- Opportunity labels and explanation reasons: descriptive output only.
