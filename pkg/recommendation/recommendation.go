package recommendation

import (
	"fmt"
	"sort"
	"strings"

	"guysports/playerstats/pkg/types"
)

// Recommendation is a deterministic player opportunity result. The score is
// deliberately transparent so an LLM can explain it without making the ranking.
type Recommendation struct {
	Player       types.Player
	Score        float64
	Opportunity  string
	Reasons      []string
	FixtureCount int
	Fixtures     []FixtureAssessment
}

type FixtureAssessment struct {
	Opponent         string
	Venue            string
	Gameweek         int
	Difficulty       string
	OpponentStrength float64
	RelativeStrength float64
}

// Rank returns the strongest available players for the next gameweek.
// It does not call an LLM and produces the same results for the same input.
func Rank(players []types.Player, limit int) []Recommendation {
	return rank(players, limit)
}

// RankWithResults ranks players using the per-player match records loaded from
// the UUID-matches.json files, including appearance minutes.
func RankWithResults(players []types.Player, results map[string][]types.GameWeekMatch, limit int) []Recommendation {
	enriched := make([]types.Player, len(players))
	copy(enriched, players)
	for i := range enriched {
		enriched[i].Results = results[enriched[i].PlayerId]
	}
	return rank(enriched, limit)
}

func rank(players []types.Player, limit int) []Recommendation {
	if limit <= 0 {
		return []Recommendation{}
	}

	teamStrengths := buildTeamStrengths(players)
	recommendations := make([]Recommendation, 0, len(players))
	for _, player := range players {
		if unavailable(player) {
			continue
		}
		recommendations = append(recommendations, score(player, teamStrengths))
	}

	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Score != recommendations[j].Score {
			return recommendations[i].Score > recommendations[j].Score
		}
		return recommendations[i].Player.DisplayName < recommendations[j].Player.DisplayName
	})
	if limit > len(recommendations) {
		limit = len(recommendations)
	}
	return recommendations[:limit]
}

func score(player types.Player, teamStrengths map[string]float64) Recommendation {
	appearancePoints := appearanceScore(player.Results)
	score := allPositionScore(player) + bonusScore(player.PpmPoints) + appearancePoints
	opportunity := "all-position scoring"
	reasons := []string{"all-position scoring rules applied"}
	if appearancePoints > 0 {
		reasons = append(reasons, fmt.Sprintf("%.0f recent appearance points", appearancePoints))
	}

	if player.Last3Average > 0 {
		score += player.Last3Average
		reasons = append(reasons, fmt.Sprintf("last three average %.1f points", player.Last3Average))
	}

	if player.PpmPoints >= 5 {
		reasons = append(reasons, fmt.Sprintf("PPM bonus band %.0f points", bonusScore(player.PpmPoints)))
	}

	if player.Position == "GK" || player.Position == "DEF" {
		opportunity = "all-position scoring plus defensive return"
		score += defensiveScore(player)
		if player.CleanSheet > 0 {
			reasons = append(reasons, fmt.Sprintf("%d clean sheets", player.CleanSheet))
		}
	}
	if player.Position == "GK" {
		opportunity = "all-position scoring plus goalkeeper return"
		if player.Saves > 0 {
			reasons = append(reasons, fmt.Sprintf("%d saves", player.Saves))
		}
	}

	fixtureAssessments := assessFixtures(player, teamStrengths)
	fixtureCount := len(fixtureAssessments)
	if fixtureCount > 0 {
		score += float64(fixtureCount - 1)
		reasons = append(reasons, fmt.Sprintf("%d upcoming fixture(s)", fixtureCount))
	}
	for _, fixture := range fixtureAssessments {
		score += fixtureAdjustment(fixture)
		reasons = append(reasons, fmt.Sprintf("%s fixture against %s", fixture.Difficulty, fixture.Opponent))
	}

	return Recommendation{Player: player, Score: score, Opportunity: opportunity, Reasons: reasons, FixtureCount: fixtureCount, Fixtures: fixtureAssessments}
}

func buildTeamStrengths(players []types.Player) map[string]float64 {
	byTeam := make(map[string][]types.Player)
	for _, player := range players {
		byTeam[player.ContestantId] = append(byTeam[player.ContestantId], player)
	}
	strengths := make(map[string]float64, len(byTeam))
	for teamID, squad := range byTeam {
		sort.Slice(squad, func(i, j int) bool {
			return float64(squad[i].TotalPoints)+squad[i].Last3Average > float64(squad[j].TotalPoints)+squad[j].Last3Average
		})
		if len(squad) > 11 {
			squad = squad[:11]
		}
		for _, player := range squad {
			strengths[teamID] += float64(player.TotalPoints) + player.Last3Average
		}
		if len(squad) > 0 {
			strengths[teamID] /= float64(len(squad))
		}
	}
	return strengths
}

func assessFixtures(player types.Player, teamStrengths map[string]float64) []FixtureAssessment {
	assessments := make([]FixtureAssessment, 0, len(player.NextGameweekFixtures))
	ownStrength := teamStrengths[player.ContestantId]
	for _, fixture := range player.NextGameweekFixtures {
		opponentStrength, knownOpponent := teamStrengths[fixture.OpponentId]
		relative := opponentStrength - ownStrength
		venue := "away"
		if fixture.IsHome {
			venue = "home"
		}
		difficulty := "unknown"
		if knownOpponent {
			difficulty = "even"
			if relative <= -5 {
				difficulty = "favorable"
			} else if relative >= 5 {
				difficulty = "difficult"
			}
		}
		assessments = append(assessments, FixtureAssessment{
			Opponent: fixture.OpponentName, Venue: venue, Gameweek: fixture.GameWeek,
			Difficulty: difficulty, OpponentStrength: opponentStrength, RelativeStrength: relative,
		})
	}
	return assessments
}

func fixtureAdjustment(fixture FixtureAssessment) float64 {
	adjustment := 0.0
	if fixture.Venue == "home" {
		adjustment++
	}
	switch fixture.Difficulty {
	case "favorable":
		adjustment += 3
	case "difficult":
		adjustment -= 2
	}
	return adjustment
}

func allPositionScore(player types.Player) float64 {
	return float64(player.Goals*6+player.Assists*3+player.ShotsOnTarget+player.ChancesCreated+player.YellowCards*-1+player.RedCards*-3+player.PenaltyMisses*-3+player.OwnGoals*-2) + float64(player.Tackles)/2
}

func defensiveScore(player types.Player) float64 {
	goalsConcededScore := 0
	if player.GoalsConceded >= 2 {
		goalsConcededScore = -(player.GoalsConceded - 1)
	}
	return float64(player.CleanSheet*5 + goalsConcededScore)
}

func bonusScore(ppmPoints float64) float64 {
	switch {
	case ppmPoints >= 12:
		return 5
	case ppmPoints >= 8:
		return 3
	case ppmPoints >= 5:
		return 1
	default:
		return 0
	}
}

func appearanceScore(matches []types.GameWeekMatch) float64 {
	points := 0
	for _, match := range matches {
		for _, stat := range match.Stats {
			if strings.EqualFold(stat.Label, "Minutes played") {
				minutes, err := stat.TotalAsInt()
				if err != nil || minutes <= 0 {
					continue
				}
				points++
				if minutes >= 60 {
					points++
				}
			}
		}
	}
	return float64(points)
}

func unavailable(player types.Player) bool {
	availability := strings.ToLower(player.AvailabilityDisplay)
	return strings.Contains(availability, "injured") || strings.Contains(availability, "suspended")
}
