# Recommend command

The `recommend` command ranks available players for their upcoming fixtures. Ranking is deterministic: the score is calculated in Go before any optional Ollama explanation is requested.

## Usage

```sh
# Print the top five and write data/model.json
go run . recommend

# Generate a model-only payload for comparison or other tooling
go run . recommend --model-only --limit 100

# Generate separate baseline and relative models from the same data
go run . recommend --model-only --limit 100 --engine legacy --model-output legacy.json
go run . recommend --model-only --limit 100 --engine relative --model-output relative.json

# Increase the impact of recent form and reduce accumulated season scoring
go run . recommend --model-only --engine relative --weights 'all_position=0.5,last_three=3'

# Rank only strikers
go run . recommend --position str

# Rank only goalkeepers
go run . recommend --position gk

# Choose another output file
go run . recommend --model-only --limit 100 --model-output gw5-model.json
```

The command reads player data from `--data-dir` (default `data`) and writes the JSON payload to `--model-output` inside that directory (default `model.json`). `--limit` defaults to `5`. With `--model-only`, it stops after writing the JSON and does not call Ollama.

The optional `--position` restricts recommendations to one position: `GK`,
`DEF`, `MID`, or `STR`. The value is case-insensitive. When set, the command
ranks all players first so team-strength and fixture-difficulty calculations
still use the full player pool, then filters to the selected position and
applies `--limit`. It cannot be combined with `--print`, which submits an
existing model payload without recalculating recommendations.

## Ranking engines

The default `--engine legacy` retains the original scoring formula below.
`--engine relative` independently normalizes each component across all eligible
players before weighting it. For each raw component, the lowest value becomes
0 and the highest becomes 1; other values are `(value - minimum) / (maximum -
minimum)`. If all values are equal (including a single eligible player), that
component contributes 0. Negative raw values are handled by the same formula.

The relative score is the sum of each normalized value multiplied by its
weight. Every weight defaults to 1; use `--weights` with comma-separated
`component=number` assignments to override any of them. Weights must be finite
and non-negative; zero disables a component. Available names are:

| Weight name | Raw component |
| --- | --- |
| `all_position` | All-position score below |
| `appearance` | Appearance score |
| `last_three` | Positive last-three average, otherwise 0 |
| `ppm` | PPM bonus band |
| `defensive` | Defensive score for GK/DEF, otherwise 0 |
| `fixture_count` | Fixture-count bonus |
| `fixture_adjustment` | Sum of venue and difficulty adjustments |

With equal weights, each component can contribute at most one point. Weighting
changes the maximum possible contribution; the final sum is not constrained
to 0–1. A relative model includes `score_components` in each JSON entry, with
the raw value, normalized fraction, weight and contribution. Legacy JSON is
unchanged. Scores depend on the eligible player pool, so compare rankings using
the same input snapshot. An outlier changes the scale for that component.
Normalization is performed before `--position` and
`--limit` are applied, while injured and suspended players are excluded from
the normalization pool. Team strengths still use all players. Raw defensive
values of 0 for other positions participate in the common player pool; a player
with no fixtures has zero raw fixture adjustment and count bonus. As in the
legacy engine, a difficult fixture may have a negative adjustment, below the
zero adjustment for no fixture.

For comparison against actual results, pass either model file to `compare`;
use separate `--output` paths to preserve both reports. `--weights` is valid only
with the relative engine. The relative engine and weights cannot be used with
`--print`, which submits a previously generated model unchanged.

## Legacy score formula and raw components

For each eligible player, the score starts with the all-position score:

```text
all-position score =
    goals * 6
  + assists * 3
  + shots on target
  + chances created
  + tackles / 2
  - yellow cards
  - red cards * 3
  - penalty misses * 3
  - own goals * 2
```

The following values are then added:

```text
score = all-position score
      + appearance score
      + last-three average
      + PPM bonus
      + defensive score       # GK and DEF only
      + fixture-count bonus
      + fixture adjustments
```

All components use the values currently available in `players.json` and the per-player match files. The legacy score does not normalize or weight them; the relative engine uses these same values as its raw components.

### Appearance score

Each match result contributes points for minutes played:

- `0` minutes or missing/invalid minutes: `0` points
- More than `0` minutes: `1` point
- `60` minutes or more: an additional `1` point

Therefore, a player receives at most `2` appearance points per recorded match.

### Last-three average

A positive `last3Average` value is added directly to the score. A zero or negative value adds nothing.

### PPM bonus

`ppmPoints` is converted into a small bonus band:

| PPM points | Bonus |
| ---: | ---: |
| 12 or more | 5 |
| 8 to 11.99 | 3 |
| 5 to 7.99 | 1 |
| Below 5 | 0 |

The bonus is applied at every position.

### Defensive and goalkeeper score

For defenders and goalkeepers, the score also includes:

```text
defensive score = clean sheets * 5 - max(goals conceded - 1, 0)
```

The goals-conceded penalty is applied only when at least two goals have been conceded. Goalkeepers use the same defensive score; saves currently affect the opportunity label/reasons but do not add another numeric score.

## Fixture scoring

Every upcoming fixture is assessed against the player's team strength.

### Fixture count

If a player has `n` upcoming fixtures, the score receives:

```text
fixture-count bonus = n - 1
```

The first fixture adds no count bonus. Additional fixtures add one point each.

### Venue and difficulty

Each fixture contributes:

- Home fixture: `+1`
- Away fixture: `0`
- Favorable difficulty: `+3`
- Even difficulty: `0`
- Difficult difficulty: `-2`
- Unknown difficulty: `0`

Difficulty is based on the difference between opponent and player-team strength:

```text
relative strength = opponent strength - player-team strength
```

| Relative strength | Difficulty |
| ---: | :--- |
| `<= -5` | Favorable |
| `-5` to `< 5` | Even |
| `>= 5` | Difficult |
| Unknown opponent team | Unknown |

The fixture adjustment is the sum of the venue and difficulty adjustments. Fixture metadata also carries the opponent's league position, recent form, and form rate when the league-table lookup matches the team name; those fields are evidence in the model output but are not separately added to the player score.

## Team-strength calculation

The recommendation engine builds one strength value per team from the available player data.

1. Group players by `contestantId`.
2. Sort each squad by `totalPoints + last3Average`.
3. Keep the top 11 players for that team.
4. Average `totalPoints + last3Average` across those players.
5. If a league table entry is available, replace that player-derived value with the weighted table value below:

```text
position score       = (21 - league position) / 20 * 100
points-rate score    = points / (played * 3) * 100
                          # 0 when played is 0
goal-difference rate  = 50 + goal difference / played * 10
                          # clamped to 0..100; 50 when played is 0
form score            = form rate * 100

team strength = position score * 0.50
              + points-rate score * 0.15
              + goal-difference rate * 0.15
              + form score * 0.20
```

The current league-table source is football-data.org. Its standings response does not reliably provide form on the free tier, so recent form is derived from finished Premier League matches and the last six results per team.

## Eligibility and ordering

Players whose availability contains `injured` or `suspended` are excluded. If
`--position` is set, only players in that position are eligible for the output.
If `--limit` is greater than the number of eligible players, all eligible
players are returned. A non-positive limit returns no recommendations.

Results are sorted by:

1. Score, descending
2. Display name, ascending, when scores are equal

The console output and generated `model.json` use this same order. The JSON also includes `player_id`, which allows the `compare` command to match recommendations to historical gameweek results.

## Ollama

Without `--model-only`, Ollama receives the generated JSON after deterministic ranking has completed. The current prompt asks it to preserve the ranking and summarize every player and fixture. It does not calculate the score or change the ranking. Use `--model-only` when only the deterministic evidence and ranking are needed.
