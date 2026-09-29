# Recommend command

The `recommend` command ranks available players for their upcoming fixtures. Ranking is deterministic: the score is calculated in Go before any optional Ollama explanation is requested.

## Usage

```sh
# Print the top five and write data/model.json
go run . recommend

# Generate a model-only payload for comparison or other tooling
go run . recommend --model-only --limit 100

# Choose another output file
go run . recommend --model-only --limit 100 --model-output gw5-model.json
```

The command reads player data from `--data-dir` (default `data`) and writes the JSON payload to `--model-output` inside that directory (default `model.json`). `--limit` defaults to `5`. With `--model-only`, it stops after writing the JSON and does not call Ollama.

## Score formula

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

All components use the values currently available in `players.json` and the per-player match files. There is no normalization or percentage weighting of the final player score, so a score is most useful for comparing players produced by the same run.

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

Players whose availability contains `injured` or `suspended` are excluded. If `--limit` is greater than the number of eligible players, all eligible players are returned. A non-positive limit returns no recommendations.

Results are sorted by:

1. Score, descending
2. Display name, ascending, when scores are equal

The console output and generated `model.json` use this same order. The JSON also includes `player_id`, which allows the `compare` command to match recommendations to historical gameweek results.

## Ollama

Without `--model-only`, Ollama receives the generated JSON after deterministic ranking has completed. The current prompt asks it to preserve the ranking and summarize every player and fixture. It does not calculate the score or change the ranking. Use `--model-only` when only the deterministic evidence and ranking are needed.
