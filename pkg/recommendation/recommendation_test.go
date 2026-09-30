package recommendation

import (
	"math"
	"testing"

	"guysports/playerstats/pkg/types"
)

func TestRecommendationEngineImplementsInterface(t *testing.T) {
	var engine RecommendationInterface = &RecommendationEngine{}
	if got := engine.Rank(nil, 5); got == nil {
		t.Fatal("Rank() returned nil; want an empty result")
	}
}

func TestScoreCombinesPlayerAndFixtureComponents(t *testing.T) {
	player := types.Player{
		PlayerId:       "player-1",
		ContestantId:   "team-1",
		ContestantName: "Test FC",
		DisplayName:    "Test Player",
		Position:       "DEF",
		Goals:          2,
		Assists:        1,
		ShotsOnTarget:  3,
		ChancesCreated: 4,
		Tackles:        4,
		YellowCards:    1,
		RedCards:       1,
		PenaltyMisses:  1,
		OwnGoals:       1,
		Last3Average:   5,
		PpmPoints:      12,
		CleanSheet:     2,
		GoalsConceded:  3,
		Results: []types.GameWeekMatch{{
			Stats: []types.GameWeekStat{{Label: "Minutes played", Total: 90}},
		}},
		NextGameweekFixtures: []types.Fixture{{
			OpponentId:   "team-2",
			OpponentName: "Opponent FC",
			IsHome:       true,
		}},
	}

	result := score(player, map[string]float64{"team-1": 10, "team-2": 0}, nil)
	want := 39.0
	if math.Abs(result.Score-want) > 1e-9 {
		t.Fatalf("score = %.1f, want %.1f", result.Score, want)
	}
	if result.Opportunity != "all-position scoring plus defensive return" {
		t.Fatalf("opportunity = %q, want defensive return", result.Opportunity)
	}
	if len(result.Fixtures) != 1 || result.Fixtures[0].Difficulty != "favorable" {
		t.Fatalf("fixtures = %+v, want one favorable fixture", result.Fixtures)
	}
}

func TestRankFiltersUnavailableAndUsesDeterministicTieBreak(t *testing.T) {
	players := []types.Player{
		{DisplayName: "Zed Player", Position: "MID"},
		{DisplayName: "Able Player", Position: "MID"},
		{DisplayName: "Unavailable Player", Position: "MID", AvailabilityDisplay: "Injured"},
	}

	got := (&RecommendationEngine{}).Rank(players, 10)
	if len(got) != 2 {
		t.Fatalf("Rank() returned %d recommendations, want 2", len(got))
	}
	if got[0].Player.DisplayName != "Able Player" || got[1].Player.DisplayName != "Zed Player" {
		t.Fatalf("ranking = %q, %q; want alphabetical tie-break", got[0].Player.DisplayName, got[1].Player.DisplayName)
	}
}

func TestAppearanceScore(t *testing.T) {
	matches := []types.GameWeekMatch{
		{Stats: []types.GameWeekStat{{Label: "Minutes played", Total: 59}}},
		{Stats: []types.GameWeekStat{{Label: "Minutes played", Total: 60}}},
		{Stats: []types.GameWeekStat{{Label: "Minutes played", Total: "invalid"}}},
	}
	if got := appearanceScore(matches); got != 3 {
		t.Fatalf("appearanceScore() = %.1f, want 3", got)
	}
}

func TestFixtureAdjustment(t *testing.T) {
	tests := []struct {
		name    string
		fixture FixtureAssessment
		want    float64
	}{
		{"home favorable", FixtureAssessment{Venue: "home", Difficulty: "favorable"}, 4},
		{"away difficult", FixtureAssessment{Venue: "away", Difficulty: "difficult"}, -2},
		{"home unknown", FixtureAssessment{Venue: "home", Difficulty: "unknown"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixtureAdjustment(tt.fixture); got != tt.want {
				t.Fatalf("fixtureAdjustment() = %.1f, want %.1f", got, tt.want)
			}
		})
	}
}

func TestTableForTeamNormalizesFootballDataNames(t *testing.T) {
	entry, ok := tableForTeam([]types.LeagueTableEntry{{Position: 4, Team: "Manchester City FC"}}, "Manchester City")
	if !ok {
		t.Fatal("tableForTeam() did not match football-data team name")
	}
	if entry.Position != 4 {
		t.Fatalf("position = %d, want 4", entry.Position)
	}
}
