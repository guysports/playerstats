package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"guysports/playerstats/pkg/ollama"
	"guysports/playerstats/pkg/recommendation"
	"guysports/playerstats/pkg/types"
)

type Recommend struct {
	Limit   int    `help:"Number of recommendations to return" default:"5"`
	DataDir string `help:"Directory containing players.json and match files" default:"data"`
}

type recommendationInput struct {
	Name        string         `json:"name"`
	Position    string         `json:"position"`
	Team        string         `json:"team"`
	Score       float64        `json:"score"`
	Opportunity string         `json:"opportunity"`
	Reasons     []string       `json:"reasons"`
	Fixtures    []fixtureInput `json:"fixtures"`
}

type fixtureInput struct {
	Opponent         string  `json:"opponent"`
	Venue            string  `json:"venue"`
	Gameweek         int     `json:"gameweek"`
	Kickoff          string  `json:"kickoff"`
	Difficulty       string  `json:"difficulty"`
	OpponentStrength float64 `json:"opponent_strength"`
	RelativeStrength float64 `json:"relative_strength"`
}

func (r *Recommend) Run(globals *Globals) error {
	players, err := readPlayers(r.DataDir)
	if err != nil {
		return err
	}
	results, err := readMatchResults(r.DataDir, players)
	if err != nil {
		return err
	}

	recommendations := recommendation.RankWithResults(players, results, r.Limit)
	for index, item := range recommendations {
		fmt.Printf("%d. %s (%s, %s) score %.1f: %s | fixtures: %s\n", index+1, item.Player.DisplayName, item.Player.Position, item.Player.ContestantName, item.Score, strings.Join(item.Reasons, "; "), formatRecommendationFixtures(item.Fixtures))
	}

	promptData := make([]recommendationInput, 0, len(recommendations))
	for _, item := range recommendations {
		fixtures := make([]fixtureInput, 0, len(item.Player.NextGameweekFixtures))
		for _, fixture := range item.Fixtures {
			fixtures = append(fixtures, fixtureInput{Opponent: fixture.Opponent, Venue: fixture.Venue, Gameweek: fixture.Gameweek, Difficulty: fixture.Difficulty, OpponentStrength: fixture.OpponentStrength, RelativeStrength: fixture.RelativeStrength})
		}
		promptData = append(promptData, recommendationInput{
			Name: item.Player.DisplayName, Position: item.Player.Position, Team: item.Player.ContestantName,
			Score: item.Score, Opportunity: item.Opportunity, Reasons: item.Reasons, Fixtures: fixtures,
		})
	}
	prompt, err := json.MarshalIndent(promptData, "", "  ")
	if err != nil {
		return err
	}

	explanation, err := ollama.NewClient(globals.OllamaURL, globals.OllamaModel, globals.OllamaTimeout).Explain(string(prompt))
	if err != nil {
		return fmt.Errorf("deterministic recommendations succeeded but Ollama explanation failed: %w", err)
	}
	fmt.Printf("\nOllama explanation:\n%s\n", explanation)
	return nil
}

func formatFixtures(fixtures []types.Fixture) string {
	if len(fixtures) == 0 {
		return "fixture data unavailable"
	}
	formatted := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		venue := "away"
		if fixture.IsHome {
			venue = "home"
		}
		formatted = append(formatted, fmt.Sprintf("%s (%s, GW%d)", fixture.OpponentName, venue, fixture.GameWeek))
	}
	return strings.Join(formatted, ", ")
}

func formatRecommendationFixtures(fixtures []recommendation.FixtureAssessment) string {
	if len(fixtures) == 0 {
		return "fixture context unavailable"
	}
	formatted := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		formatted = append(formatted, fmt.Sprintf("%s (%s, GW%d, %s)", fixture.Opponent, fixture.Venue, fixture.Gameweek, fixture.Difficulty))
	}
	return strings.Join(formatted, ", ")
}

func readPlayers(dataDir string) ([]types.Player, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, "players.json"))
	if err != nil {
		return nil, err
	}
	var players []types.Player
	if err := json.Unmarshal(data, &players); err != nil {
		return nil, err
	}
	return players, nil
}

func readMatchResults(dataDir string, players []types.Player) (map[string][]types.GameWeekMatch, error) {
	results := make(map[string][]types.GameWeekMatch, len(players))
	for _, player := range players {
		data, err := os.ReadFile(filepath.Join(dataDir, fmt.Sprintf("%s-matches.json", player.PlayerId)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var matches types.GameWeek
		if err := json.Unmarshal(data, &matches); err != nil {
			return nil, err
		}
		results[player.PlayerId] = matches.Data.Items
	}
	return results, nil
}
