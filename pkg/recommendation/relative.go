package recommendation

import (
	"fmt"
	"math"

	"guysports/playerstats/pkg/types"
)

var componentNames = []string{
	"all_position", "appearance", "last_three", "ppm", "defensive", "fixture_count", "fixture_adjustment",
}

// RelativeRecommendationEngine normalizes each component across eligible players.
// Its zero value uses equal weights.
type RelativeRecommendationEngine struct {
	weights map[string]float64
}

// NewRelativeRecommendationEngine overrides equal default weights by component name.
func NewRelativeRecommendationEngine(weights map[string]float64) (*RelativeRecommendationEngine, error) {
	copyWeights := make(map[string]float64, len(weights))
	for name, weight := range weights {
		known := false
		for _, component := range componentNames {
			if component == name {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("unknown score component %q", name)
		}
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 {
			return nil, fmt.Errorf("weight for %s must be a finite non-negative number", name)
		}
		copyWeights[name] = weight
	}
	total := 0.0
	for _, name := range componentNames {
		weight, ok := copyWeights[name]
		if !ok {
			weight = 1
		}
		total += weight
	}
	if math.IsInf(total, 0) {
		return nil, fmt.Errorf("sum of component weights is too large")
	}
	return &RelativeRecommendationEngine{weights: copyWeights}, nil
}

func (e *RelativeRecommendationEngine) Rank(players []types.Player, limit int) []Recommendation {
	return e.rank(players, nil, limit)
}

func (e *RelativeRecommendationEngine) RankWithResults(players []types.Player, results map[string][]types.GameWeekMatch, limit int) []Recommendation {
	return e.RankWithTable(players, results, nil, limit)
}

func (e *RelativeRecommendationEngine) RankWithTable(players []types.Player, results map[string][]types.GameWeekMatch, table []types.LeagueTableEntry, limit int) []Recommendation {
	enriched := make([]types.Player, len(players))
	copy(enriched, players)
	for i := range enriched {
		enriched[i].Results = results[enriched[i].PlayerId]
	}
	return e.rank(enriched, table, limit)
}

func (e *RelativeRecommendationEngine) rank(players []types.Player, table []types.LeagueTableEntry, limit int) []Recommendation {
	if limit <= 0 {
		return []Recommendation{}
	}
	strengths := buildTeamStrengths(players, table)
	recommendations := make([]Recommendation, 0, len(players))
	minimum := make(map[string]float64, len(componentNames))
	maximum := make(map[string]float64, len(componentNames))
	for _, player := range players {
		if unavailable(player) {
			continue
		}
		item := score(player, strengths, table)
		item.ScoreComponents = rawComponents(item)
		for name, component := range item.ScoreComponents {
			if len(recommendations) == 0 || component.Raw < minimum[name] {
				minimum[name] = component.Raw
			}
			if len(recommendations) == 0 || component.Raw > maximum[name] {
				maximum[name] = component.Raw
			}
		}
		recommendations = append(recommendations, item)
	}
	for i := range recommendations {
		item := &recommendations[i]
		item.Score = 0
		for _, name := range componentNames {
			component := item.ScoreComponents[name]
			if maximum[name] > minimum[name] {
				component.Normalized = (component.Raw - minimum[name]) / (maximum[name] - minimum[name])
			}
			component.Weight = 1
			if weight, ok := e.weights[name]; ok {
				component.Weight = weight
			}
			component.Contribution = component.Normalized * component.Weight
			item.Score += component.Contribution
			item.ScoreComponents[name] = component
		}
	}
	return sortAndLimit(recommendations, limit)
}

func rawComponents(item Recommendation) map[string]ScoreComponent {
	player := item.Player
	defensive := 0.0
	if player.Position == "GK" || player.Position == "DEF" {
		defensive = defensiveScore(player)
	}
	fixtureCount := 0.0
	if item.FixtureCount > 0 {
		fixtureCount = float64(item.FixtureCount - 1)
	}
	fixtureAdjustmentTotal := 0.0
	for _, fixture := range item.Fixtures {
		fixtureAdjustmentTotal += fixtureAdjustment(fixture)
	}
	return map[string]ScoreComponent{
		"all_position":       {Raw: allPositionScore(player)},
		"appearance":         {Raw: appearanceScore(player.Results)},
		"last_three":         {Raw: math.Max(0, player.Last3Average)},
		"ppm":                {Raw: bonusScore(player.PpmPoints)},
		"defensive":          {Raw: defensive},
		"fixture_count":      {Raw: fixtureCount},
		"fixture_adjustment": {Raw: fixtureAdjustmentTotal},
	}
}
