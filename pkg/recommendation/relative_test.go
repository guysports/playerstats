package recommendation

import (
	"math"
	"testing"

	"guysports/playerstats/pkg/types"
)

func TestRelativeEngineImplementsInterface(t *testing.T) {
	var engine RecommendationInterface = &RelativeRecommendationEngine{}
	if got := engine.Rank(nil, 5); got == nil || len(got) != 0 {
		t.Fatalf("empty ranking = %v", got)
	}
}

func TestRelativeNormalizationAndWeights(t *testing.T) {
	players := []types.Player{
		{DisplayName: "High season", Position: "MID", Goals: 10, Last3Average: 0},
		{DisplayName: "High form", Position: "MID", Goals: 0, Last3Average: 10},
		{DisplayName: "Middle", Position: "MID", Goals: 5, Last3Average: 5},
		{DisplayName: "Injured", Position: "MID", Goals: 1000, Last3Average: 1000, AvailabilityDisplay: "injured"},
	}
	engine, err := NewRelativeRecommendationEngine(map[string]float64{"all_position": 1, "last_three": 3})
	if err != nil {
		t.Fatal(err)
	}
	got := engine.Rank(players, 10)
	if len(got) != 3 || got[0].Player.DisplayName != "High form" || got[1].Player.DisplayName != "Middle" || got[2].Player.DisplayName != "High season" {
		t.Fatalf("unexpected ranking: %+v", got)
	}
	for i, want := range []float64{3, 2, 1} {
		if math.Abs(got[i].Score-want) > 1e-9 {
			t.Fatalf("score[%d] = %v, want %v", i, got[i].Score, want)
		}
		sum := 0.0
		for name, component := range got[i].ScoreComponents {
			if component.Normalized < 0 || component.Normalized > 1 {
				t.Fatalf("%s fraction out of range: %v", name, component.Normalized)
			}
			if math.Abs(component.Contribution-component.Weight*component.Normalized) > 1e-9 {
				t.Fatalf("%s contribution incorrect: %+v", name, component)
			}
			sum += component.Contribution
		}
		if math.Abs(sum-got[i].Score) > 1e-9 {
			t.Fatalf("component sum %v differs from score %v", sum, got[i].Score)
		}
	}
	if got[1].ScoreComponents["all_position"].Normalized != 0.5 || got[1].ScoreComponents["last_three"].Normalized != 0.5 {
		t.Fatal("expected midpoint normalization")
	}
	reversed, err := NewRelativeRecommendationEngine(map[string]float64{"all_position": 3, "last_three": 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := reversed.Rank(players, 1); got[0].Player.DisplayName != "High season" {
		t.Fatalf("weight change did not change rank: %+v", got)
	}
	if top := engine.Rank(players, 1); top[0].Score != got[0].Score {
		t.Fatal("limit must not change normalization")
	}
	allPositionWeight := got[0].ScoreComponents["all_position"].Weight
	if allPositionWeight != 1 {
		t.Fatalf("unexpected all-position weight: %v", allPositionWeight)
	}
}

func TestRelativeNegativeAndConstantComponents(t *testing.T) {
	players := []types.Player{
		{DisplayName: "Negative", Position: "DEF", GoalsConceded: 4, YellowCards: 2},
		{DisplayName: "Zero", Position: "MID"},
		{DisplayName: "Positive", Position: "DEF", CleanSheet: 1, Goals: 1},
	}
	ranked := (&RelativeRecommendationEngine{}).Rank(players, 3)
	byName := make(map[string]Recommendation)
	for _, item := range ranked {
		byName[item.Player.DisplayName] = item
	}
	for _, name := range []string{"all_position", "defensive"} {
		if byName["Negative"].ScoreComponents[name].Normalized != 0 || byName["Positive"].ScoreComponents[name].Normalized != 1 {
			t.Fatalf("%s did not normalize negative and positive values", name)
		}
	}
	for _, item := range ranked {
		if item.ScoreComponents["fixture_count"].Normalized != 0 || item.ScoreComponents["appearance"].Normalized != 0 {
			t.Fatal("constant components must have zero contribution")
		}
	}
	if got := (&RelativeRecommendationEngine{}).Rank(players[:1], 1); got[0].Score != 0 {
		t.Fatalf("single eligible player score = %v, want 0", got[0].Score)
	}
	if got := (&RelativeRecommendationEngine{}).Rank(players, 0); len(got) != 0 {
		t.Fatal("nonpositive limit must return no recommendations")
	}
}

func TestRelativeUsesResultsAndFixtures(t *testing.T) {
	players := []types.Player{
		{PlayerId: "a", DisplayName: "A", ContestantId: "a", Position: "MID", NextGameweekFixtures: []types.Fixture{{OpponentId: "b", IsHome: true}, {OpponentId: "b"}}},
		{PlayerId: "b", DisplayName: "B", ContestantId: "b", Position: "MID"},
	}
	results := map[string][]types.GameWeekMatch{"a": {{Stats: []types.GameWeekStat{{Label: "Minutes played", Total: 90}}}}}
	got := (&RelativeRecommendationEngine{}).RankWithResults(players, results, 2)
	if got[0].Player.PlayerId != "a" || got[0].ScoreComponents["appearance"].Raw != 2 || got[0].ScoreComponents["fixture_count"].Raw != 1 || got[0].ScoreComponents["fixture_adjustment"].Raw != 1 {
		t.Fatalf("results and fixture components not applied: %+v", got[0].ScoreComponents)
	}
}

func TestRelativeWeightValidation(t *testing.T) {
	for _, weights := range []map[string]float64{
		{"unknown": 1}, {"ppm": -1}, {"ppm": math.NaN()}, {"ppm": math.Inf(1)},
		{"ppm": math.MaxFloat64, "appearance": math.MaxFloat64},
	} {
		if _, err := NewRelativeRecommendationEngine(weights); err == nil {
			t.Fatalf("expected invalid weights error for %v", weights)
		}
	}
	if _, err := NewRelativeRecommendationEngine(map[string]float64{"ppm": 0}); err != nil {
		t.Fatalf("zero weight should be permitted: %v", err)
	}
}
