package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"guysports/playerstats/pkg/types"
)

func TestReadPlayers(t *testing.T) {
	dataDir := t.TempDir()
	players := []types.Player{{PlayerId: "player-1", DisplayName: "Test Player", Position: "MID"}}
	data, err := json.Marshal(players)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "players.json"), data, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	got, err := readPlayers(dataDir)
	if err != nil {
		t.Fatalf("readPlayers() returned an error: %v", err)
	}
	if len(got) != 1 || got[0].PlayerId != "player-1" {
		t.Fatalf("readPlayers() = %+v, want one player", got)
	}
}

func TestLoadFantasyTeam(t *testing.T) {
	teamJSON := `{"success":true,"data":{"id":"team-1","teamName":"Test Team","formation":"3-5-2","totalPoints":505,"managerName":"G. Barden","players":[{"id":"squad-1","fantasyPlayerId":"player-1","isStarter":true,"isCaptain":true,"positionSlot":9,"totalPoints":39,"player":{"id":"player-1","displayName":"E. Haaland","position":"STR","price":14.5,"contestantName":"Manchester City"}}]}}`

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(teamJSON))
	}))
	defer server.Close()

	team, err := loadFantasyTeam(server.URL+"/api/teams/scoring/%s", "team-1", "test-token")
	if err != nil {
		t.Fatalf("loadFantasyTeam() returned an error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
	if team.TeamName != "Test Team" || team.Formation != "3-5-2" || len(team.Players) != 1 {
		t.Fatalf("loadFantasyTeam() = %+v, want one player in Test Team", team)
	}
	if team.Players[0].Player.DisplayName != "E. Haaland" || !team.Players[0].IsCaptain {
		t.Fatalf("loadFantasyTeam() player = %+v, want captain E. Haaland", team.Players[0])
	}
}

func TestLoadFantasyTeamUnsuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"data":{}}`))
	}))
	defer server.Close()

	if _, err := loadFantasyTeam(server.URL+"/%s", "team-1", "test-token"); err == nil {
		t.Fatal("loadFantasyTeam() expected an error for an unsuccessful response")
	}
}

func TestReadMatchResultsAllowsMissingFiles(t *testing.T) {
	dataDir := t.TempDir()
	players := []types.Player{{PlayerId: "player-1"}, {PlayerId: "missing"}}
	matches := `{"data":{"items":[{"matchId":"match-1","stats":[{"label":"Minutes played","total":90}]}]}}`
	if err := os.WriteFile(filepath.Join(dataDir, "player-1-matches.json"), []byte(matches), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	got, err := readMatchResults(dataDir, players)
	if err != nil {
		t.Fatalf("readMatchResults() returned an error: %v", err)
	}
	if len(got["player-1"]) != 1 || got["player-1"][0].Stats[0].Label != "Minutes played" {
		t.Fatalf("readMatchResults() = %+v, want player match stats", got)
	}
	if _, ok := got["missing"]; ok {
		t.Fatal("readMatchResults() added an entry for a missing match file")
	}
}

func TestRecommendRunUsesOllama(t *testing.T) {
	dataDir := t.TempDir()
	players := []types.Player{
		{
			PlayerId: "player-1", DisplayName: "Top Player", Position: "MID", ContestantName: "Test FC",
			Last3Average: 8, Goals: 2, NextGameweekFixtures: []types.Fixture{{OpponentName: "Other FC", IsHome: true}},
		},
	}
	playersData, err := json.Marshal(players)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "players.json"), playersData, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	matches := `{"data":{"items":[{"matchId":"match-1","stats":[{"label":"Minutes played","total":90}]}]}}`
	if err := os.WriteFile(filepath.Join(dataDir, "player-1-matches.json"), []byte(matches), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	var requestBody string
	var requestJSON []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("request path = %q, want /api/chat", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("request body is invalid JSON: %v", err)
		}
		requestJSON, _ = json.Marshal(body)
		requestBody = fmt.Sprint(body["model"])
		_, _ = fmt.Fprint(w, `{"message":{"content":"Top Player is the leading recommendation."}}`)
	}))
	defer server.Close()

	err = (&Recommend{Limit: 1, DataDir: dataDir}).Run(&Globals{OllamaURL: server.URL, OllamaModel: "test-model"})
	if err != nil {
		t.Fatalf("Recommend.Run() returned an error: %v", err)
	}
	if requestBody != "test-model" {
		t.Fatalf("Ollama model = %q, want test-model", requestBody)
	}
	if !strings.Contains(string(requestJSON), "Other FC") || !strings.Contains(string(requestJSON), "home") {
		t.Fatalf("Ollama request does not contain structured fixture context: %s", requestJSON)
	}
}

func TestRecommendRunWritesModelJSON(t *testing.T) {
	dataDir := t.TempDir()
	players := []types.Player{
		{
			PlayerId: "player-1", DisplayName: "Top Player", Position: "MID", ContestantName: "Test FC",
			Last3Average: 8, Goals: 2, NextGameweekFixtures: []types.Fixture{{OpponentName: "Other FC", IsHome: true}},
		},
	}
	playersData, err := json.Marshal(players)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "players.json"), playersData, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	matches := `{"data":{"items":[{"matchId":"match-1","stats":[{"label":"Minutes played","total":90}]}]}}`
	if err := os.WriteFile(filepath.Join(dataDir, "player-1-matches.json"), []byte(matches), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"message":{"content":"Top Player is the leading recommendation."}}`)
	}))
	defer server.Close()

	err = (&Recommend{Limit: 1, DataDir: dataDir}).Run(&Globals{OllamaURL: server.URL, OllamaModel: "test-model"})
	if err != nil {
		t.Fatalf("Recommend.Run() returned an error: %v", err)
	}

	modelPath := filepath.Join(dataDir, "model.json")
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned an error: %v", modelPath, err)
	}
	if !strings.Contains(string(content), "Top Player") || !strings.Contains(string(content), "Other FC") {
		t.Fatalf("model.json does not contain expected recommendation data: %s", string(content))
	}
}

func TestRecommendRunReportsOllamaFailure(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "players.json"), []byte(`[]`), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "offline", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := (&Recommend{Limit: 1, DataDir: dataDir}).Run(&Globals{OllamaURL: server.URL, OllamaModel: "test-model"})
	if err == nil || !strings.Contains(err.Error(), "Ollama explanation failed") {
		t.Fatalf("Recommend.Run() error = %v, want Ollama explanation failure", err)
	}
}

func TestRecommendRunUsesCurrentGameweekFixturesAndOdds(t *testing.T) {
	dataDir := t.TempDir()
	players := []types.Player{
		{
			PlayerId: "player-1", DisplayName: "Top Player", Position: "MID", ContestantName: "Test FC",
			Last3Average: 8, Goals: 2,
			// Stale: players.json already rolled over to a fixture that isn't
			// the imminent one; currentgameweek.json should take priority.
			NextGameweekFixtures: []types.Fixture{{OpponentName: "Stale FC", IsHome: false}},
		},
	}
	playersData, err := json.Marshal(players)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "players.json"), playersData, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	matches := `{"data":{"items":[{"matchId":"match-1","stats":[{"label":"Minutes played","total":90}]}]}}`
	if err := os.WriteFile(filepath.Join(dataDir, "player-1-matches.json"), []byte(matches), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	currentGameweek := `{"success":true,"data":{"matches":[{
		"id":"match-cur","homeContestant":{"id":"team-1","name":"Test FC","shortName":"Test"},
		"awayContestant":{"id":"team-2","name":"Other FC","shortName":"Other"},
		"kickoffAt":"2026-09-19T14:00:00.000Z","status":"fixture","gameweek":5,
		"odds":{"homePrice":"1.80","drawPrice":"3.50","awayPrice":"4.20"}
	}]}}`
	if err := os.WriteFile(filepath.Join(dataDir, "currentgameweek.json"), []byte(currentGameweek), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"message":{"content":"Top Player is the leading recommendation."}}`)
	}))
	defer server.Close()

	err = (&Recommend{Limit: 1, DataDir: dataDir}).Run(&Globals{OllamaURL: server.URL, OllamaModel: "test-model"})
	if err != nil {
		t.Fatalf("Recommend.Run() returned an error: %v", err)
	}

	modelPath := filepath.Join(dataDir, "model.json")
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned an error: %v", modelPath, err)
	}
	if strings.Contains(string(content), "Stale FC") {
		t.Fatalf("model.json used the stale nextGameweekFixtures entry: %s", string(content))
	}
	if !strings.Contains(string(content), "Other FC") {
		t.Fatalf("model.json missing current gameweek opponent: %s", string(content))
	}
	if !strings.Contains(string(content), `"home": 1.8`) {
		t.Fatalf("model.json missing embedded betfair odds from currentgameweek.json: %s", string(content))
	}
}
