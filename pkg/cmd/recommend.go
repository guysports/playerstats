package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"guysports/playerstats/pkg/betfair"
	"guysports/playerstats/pkg/helper"
	"guysports/playerstats/pkg/ollama"
	"guysports/playerstats/pkg/recommendation"
	"guysports/playerstats/pkg/types"
)

type Recommend struct {
	Limit         int    `help:"Number of recommendations to return" default:"5"`
	DataDir       string `help:"Directory containing players.json and match files" default:"data"`
	ModelOutput   string `help:"Filename to write the recommendation JSON payload to before submitting to Ollama" default:"model.json"`
	ModelOnly     bool   `help:"Write the recommendation JSON payload and exit without sending it to Ollama"`
	Print         bool   `help:"Submit the existing model output file to Ollama without regenerating it"`
	JSONLoginPath string `help:"Path to the Betfair login JSON file" env:"BETFAIR_LOGIN_PATH" default:"" name:"json-login-path"`
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
	Opponent         string            `json:"opponent"`
	Venue            string            `json:"venue"`
	Gameweek         int               `json:"gameweek"`
	Kickoff          string            `json:"kickoff"`
	Difficulty       string            `json:"difficulty"`
	OpponentStrength float64           `json:"opponent_strength"`
	RelativeStrength float64           `json:"relative_strength"`
	OpponentPosition int               `json:"opponent_position"`
	OpponentForm     string            `json:"opponent_form"`
	OpponentFormRate float64           `json:"opponent_form_rate"`
	BetfairOdds      *betfairOddsInput `json:"betfair_odds,omitempty"`
}

type betfairOddsInput struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

func (r *Recommend) Run(globals *Globals) error {
	if r.DataDir == "" {
		r.DataDir = "data"
	}
	if r.ModelOutput == "" {
		r.ModelOutput = "model.json"
	}

	if r.Print {
		return r.printExisting(globals)
	}

	if r.JSONLoginPath != "" {
		if _, err := os.Stat(r.JSONLoginPath); err != nil {
			return fmt.Errorf("betfair login json path %q is not accessible: %w", r.JSONLoginPath, err)
		}
		if globals.BetfairAppKey == "" {
			fmt.Printf("warning: BETFAIR_APP_KEY is not set; the login JSON path is configured but the app key is still required for Betfair auth\n")
		}
	}

	players, err := readPlayers(r.DataDir)
	if err != nil {
		return err
	}
	results, err := readMatchResults(r.DataDir, players)
	if err != nil {
		return err
	}
	currentGameweekMatches, err := loadCurrentGameweekMatches(r.DataDir)
	if err != nil {
		fmt.Printf("warning: current gameweek fixtures unavailable (%v); falling back to nextGameweekFixtures\n", err)
		currentGameweekMatches = nil
	}
	currentFixturesByTeam := currentGameweekFixturesByTeam(currentGameweekMatches)
	currentOddsByTeams := currentGameweekOddsByTeams(currentGameweekMatches)
	for i := range players {
		if fixtures, ok := currentFixturesByTeam[players[i].ContestantName]; ok {
			players[i].NextGameweekFixtures = fixtures
		}
	}
	var table []types.LeagueTableEntry
	if globals.LeagueTableURL != "" {
		table, err = loadLeagueTable(globals.LeagueTableURL, globals.FootballDataAPIToken)
		if err != nil {
			fmt.Printf("warning: league table unavailable (%v); continuing without table context\n", err)
			table = nil
		}
	}

	var betfairClient *betfair.Client
	if r.JSONLoginPath != "" {
		betfairClient, err = betfair.NewClient(r.JSONLoginPath, globals.BetfairAppKey)
		if err != nil {
			fmt.Printf("warning: betfair client unavailable (%v); continuing without odds\n", err)
			betfairClient = nil
		}
	}

	recommendations := recommendation.RankWithTable(players, results, table, r.Limit)
	for index, item := range recommendations {
		fmt.Printf("%d. %s (%s, %s) score %.1f: %s | fixtures: %s\n", index+1, item.Player.DisplayName, item.Player.Position, item.Player.ContestantName, item.Score, strings.Join(item.Reasons, "; "), formatRecommendationFixtures(item.Fixtures))
	}

	promptData := make([]recommendationInput, 0, len(recommendations))
	for _, item := range recommendations {
		fixtures := make([]fixtureInput, 0, len(item.Player.NextGameweekFixtures))
		for _, fixture := range item.Fixtures {
			fixtureInputData := fixtureInput{Opponent: fixture.Opponent, Venue: fixture.Venue, Gameweek: fixture.Gameweek, Difficulty: fixture.Difficulty, OpponentStrength: fixture.OpponentStrength, RelativeStrength: fixture.RelativeStrength, OpponentPosition: fixture.OpponentPosition, OpponentForm: fixture.OpponentForm, OpponentFormRate: fixture.OpponentFormRate}
			homeTeam, awayTeam := item.Player.ContestantName, fixture.Opponent
			if fixture.Venue == "away" {
				homeTeam, awayTeam = fixture.Opponent, item.Player.ContestantName
			}
			if odds, ok := currentOddsByTeams[oddsKey(homeTeam, awayTeam)]; ok {
				fixtureInputData.BetfairOdds = &odds
			} else if betfairClient != nil {
				if odds, err := betfairClient.MatchOdds(homeTeam, awayTeam); err == nil {
					fixtureInputData.BetfairOdds = &betfairOddsInput{Home: odds.Home, Draw: odds.Draw, Away: odds.Away}
				} else {
					fmt.Printf("warning: betfair odds unavailable for %s vs %s (%v)\n", homeTeam, awayTeam, err)
				}
			}
			fixtures = append(fixtures, fixtureInputData)
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

	modelPath := filepath.Join(r.DataDir, r.ModelOutput)
	if err := os.WriteFile(modelPath, prompt, 0644); err != nil {
		return fmt.Errorf("write model payload to %s: %w", modelPath, err)
	}
	fmt.Printf("Model payload written to %s\n", modelPath)
	if r.ModelOnly {
		return nil
	}

	explanation, err := ollama.NewClient(globals.OllamaURL, globals.OllamaModel, globals.OllamaTimeout).Explain(string(prompt))
	if err != nil {
		return fmt.Errorf("deterministic recommendations succeeded but Ollama explanation failed: %w", err)
	}
	fmt.Printf("\nOllama explanation:\n%s\n", explanation)
	return nil
}

// printExisting submits an already-generated model output file to Ollama
// without recomputing recommendations, odds, etc.
func (r *Recommend) printExisting(globals *Globals) error {
	modelPath := filepath.Join(r.DataDir, r.ModelOutput)
	prompt, err := os.ReadFile(modelPath)
	if err != nil {
		return fmt.Errorf("read model payload from %s: %w", modelPath, err)
	}

	explanation, err := ollama.NewClient(globals.OllamaURL, globals.OllamaModel, globals.OllamaTimeout).Explain(string(prompt))
	if err != nil {
		return fmt.Errorf("ollama explanation failed: %w", err)
	}
	fmt.Printf("\nOllama explanation:\n%s\n", explanation)
	return nil
}

// loadLeagueTable fetches the Premier League table and recent form from the
// football-data.org API. The standings endpoint doesn't report form on the
// free tier, so recent form is derived separately from finished matches.
func loadLeagueTable(baseURL, apiToken string) ([]types.LeagueTableEntry, error) {
	headers := map[string]string{}
	if apiToken != "" {
		headers["X-Auth-Token"] = apiToken
	}

	standingsData, err := helper.GetJSONWithHeaders(baseURL+"/standings", headers)
	if err != nil {
		return nil, fmt.Errorf("fetch standings: %w", err)
	}
	entries, err := helper.ParseFootballDataStandings(standingsData)
	if err != nil {
		return nil, err
	}

	matchesData, err := helper.GetJSONWithHeaders(baseURL+"/matches?status=FINISHED", headers)
	if err != nil {
		fmt.Printf("warning: recent form unavailable (%v); continuing without it\n", err)
		return entries, nil
	}
	if err := helper.ApplyRecentForm(entries, matchesData); err != nil {
		fmt.Printf("warning: recent form unavailable (%v); continuing without it\n", err)
	}
	return entries, nil
}

// loadCurrentGameweekMatches reads data/currentgameweek.json; a missing file
// is not an error since --gw is only dumped when a caller opts in.
func loadCurrentGameweekMatches(dataDir string) ([]types.CurrentGameweekMatch, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, "currentgameweek.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload types.CurrentGameweekPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.Data.Matches, nil
}

// currentGameweekFixturesByTeam converts current gameweek matches into
// Fixture lists keyed by team name, to override the potentially stale
// NextGameweekFixtures on the player record.
func currentGameweekFixturesByTeam(matches []types.CurrentGameweekMatch) map[string][]types.Fixture {
	fixturesByTeam := make(map[string][]types.Fixture, len(matches)*2)
	for _, match := range matches {
		home := types.Fixture{
			OpponentId: match.AwayContestant.ID, OpponentName: match.AwayContestant.Name, OpponentShortName: match.AwayContestant.ShortName,
			ContestantFlagKey: match.AwayContestant.ContestantFlagKey, Status: match.Status, KickoffAt: match.KickoffAt,
			Venue: "home", IsHome: true, GameWeek: match.Gameweek,
		}
		away := types.Fixture{
			OpponentId: match.HomeContestant.ID, OpponentName: match.HomeContestant.Name, OpponentShortName: match.HomeContestant.ShortName,
			ContestantFlagKey: match.HomeContestant.ContestantFlagKey, Status: match.Status, KickoffAt: match.KickoffAt,
			Venue: "away", IsHome: false, GameWeek: match.Gameweek,
		}
		fixturesByTeam[match.HomeContestant.Name] = append(fixturesByTeam[match.HomeContestant.Name], home)
		fixturesByTeam[match.AwayContestant.Name] = append(fixturesByTeam[match.AwayContestant.Name], away)
	}
	return fixturesByTeam
}

// currentGameweekOddsByTeams extracts the odds already embedded in
// currentgameweek.json, keyed by home/away team pair, so the recommend
// command doesn't need to call the Betfair API when they're available.
func currentGameweekOddsByTeams(matches []types.CurrentGameweekMatch) map[string]betfairOddsInput {
	odds := make(map[string]betfairOddsInput, len(matches))
	for _, match := range matches {
		if match.Odds == nil {
			continue
		}
		home, homeErr := strconv.ParseFloat(match.Odds.HomePrice, 64)
		draw, drawErr := strconv.ParseFloat(match.Odds.DrawPrice, 64)
		away, awayErr := strconv.ParseFloat(match.Odds.AwayPrice, 64)
		if homeErr != nil || drawErr != nil || awayErr != nil {
			continue
		}
		odds[oddsKey(match.HomeContestant.Name, match.AwayContestant.Name)] = betfairOddsInput{Home: home, Draw: draw, Away: away}
	}
	return odds
}

func oddsKey(homeTeam, awayTeam string) string {
	return strings.ToLower(strings.TrimSpace(homeTeam)) + "|" + strings.ToLower(strings.TrimSpace(awayTeam))
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
