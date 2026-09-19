package helper

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"guysports/playerstats/pkg/types"
)

const recentFormMatches = 6

// footballDataStandingsResponse mirrors the payload returned by the
// football-data.org /competitions/{id}/standings endpoint.
type footballDataStandingsResponse struct {
	Standings []struct {
		Type  string `json:"type"`
		Table []struct {
			Position int `json:"position"`
			Team     struct {
				Name string `json:"name"`
			} `json:"team"`
			PlayedGames    int `json:"playedGames"`
			Won            int `json:"won"`
			Draw           int `json:"draw"`
			Lost           int `json:"lost"`
			Points         int `json:"points"`
			GoalsFor       int `json:"goalsFor"`
			GoalsAgainst   int `json:"goalsAgainst"`
			GoalDifference int `json:"goalDifference"`
		} `json:"table"`
	} `json:"standings"`
}

// ParseFootballDataStandings converts the football-data.org standings
// payload into typed league table rows. The free API tier always returns a
// null "form" field on this endpoint, so recent form is left empty here;
// call ApplyRecentForm with a finished-matches payload to populate it.
func ParseFootballDataStandings(data []byte) ([]types.LeagueTableEntry, error) {
	var payload footballDataStandingsResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse football-data standings: %w", err)
	}
	for _, standing := range payload.Standings {
		if standing.Type != "TOTAL" {
			continue
		}
		entries := make([]types.LeagueTableEntry, 0, len(standing.Table))
		for _, row := range standing.Table {
			entries = append(entries, types.LeagueTableEntry{
				Position: row.Position, Team: row.Team.Name,
				Played: row.PlayedGames, Won: row.Won, Drawn: row.Draw, Lost: row.Lost,
				GoalsFor: row.GoalsFor, GoalsAgainst: row.GoalsAgainst,
				GoalDifference: row.GoalDifference, Points: row.Points,
			})
		}
		return entries, nil
	}
	return nil, fmt.Errorf("no TOTAL standings found in football-data response")
}

// footballDataMatchesResponse mirrors the payload returned by the
// football-data.org /competitions/{id}/matches endpoint.
type footballDataMatchesResponse struct {
	Matches []struct {
		UtcDate  string `json:"utcDate"`
		Status   string `json:"status"`
		HomeTeam struct {
			Name string `json:"name"`
		} `json:"homeTeam"`
		AwayTeam struct {
			Name string `json:"name"`
		} `json:"awayTeam"`
		Score struct {
			Winner string `json:"winner"`
		} `json:"score"`
	} `json:"matches"`
}

// ApplyRecentForm derives each team's last six results (oldest first) from a
// football-data.org finished-matches payload and fills in RecentForm,
// FormPoints and FormRate on the matching entries, since the standings
// endpoint itself does not report form on the free API tier.
func ApplyRecentForm(entries []types.LeagueTableEntry, matchesData []byte) error {
	var payload footballDataMatchesResponse
	if err := json.Unmarshal(matchesData, &payload); err != nil {
		return fmt.Errorf("parse football-data matches: %w", err)
	}

	type result struct {
		date   time.Time
		letter string
	}
	resultsByTeam := make(map[string][]result)
	for _, match := range payload.Matches {
		if match.Status != "FINISHED" {
			continue
		}
		date, err := time.Parse(time.RFC3339, match.UtcDate)
		if err != nil {
			continue
		}
		homeLetter, awayLetter := "D", "D"
		switch match.Score.Winner {
		case "HOME_TEAM":
			homeLetter, awayLetter = "W", "L"
		case "AWAY_TEAM":
			homeLetter, awayLetter = "L", "W"
		}
		resultsByTeam[match.HomeTeam.Name] = append(resultsByTeam[match.HomeTeam.Name], result{date, homeLetter})
		resultsByTeam[match.AwayTeam.Name] = append(resultsByTeam[match.AwayTeam.Name], result{date, awayLetter})
	}

	for i := range entries {
		results, ok := resultsByTeam[entries[i].Team]
		if !ok {
			continue
		}
		sort.Slice(results, func(a, b int) bool { return results[a].date.Before(results[b].date) })
		if len(results) > recentFormMatches {
			results = results[len(results)-recentFormMatches:]
		}
		letters := make([]string, 0, len(results))
		points := 0
		for _, r := range results {
			letters = append(letters, r.letter)
			switch r.letter {
			case "W":
				points += 3
			case "D":
				points++
			}
		}
		entries[i].RecentForm = strings.Join(letters, "")
		entries[i].FormPoints = points
		if len(letters) > 0 {
			entries[i].FormRate = float64(points) / float64(len(letters)*3)
		}
	}
	return nil
}
