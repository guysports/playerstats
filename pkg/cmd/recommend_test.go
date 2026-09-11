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
