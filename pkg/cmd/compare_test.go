package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameweekPointsSumsAcrossMatches(t *testing.T) {
	dataDir := t.TempDir()
	matches := `{"data":{"items":[
		{"matchId":"m1","mdLabel":"GW 4","mdPoints":"+3"},
		{"matchId":"m2","mdLabel":"GW 4","mdPoints":"-1"},
		{"matchId":"m3","mdLabel":"GW 3","mdPoints":"+10"}
	]}}`
	if err := os.WriteFile(filepath.Join(dataDir, "player-1-matches.json"), []byte(matches), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	points, found, err := gameweekPoints(dataDir, "player-1", 4)
	if err != nil {
		t.Fatalf("gameweekPoints() returned an error: %v", err)
	}
	if !found || points != 2 {
		t.Fatalf("gameweekPoints() = (%d, %v), want (2, true)", points, found)
	}

	if _, found, err := gameweekPoints(dataDir, "player-1", 9); err != nil || found {
		t.Fatalf("gameweekPoints() for unplayed gameweek = (found=%v, err=%v), want (false, nil)", found, err)
	}

	if _, found, err := gameweekPoints(dataDir, "missing-player", 4); err != nil || found {
		t.Fatalf("gameweekPoints() for missing player = (found=%v, err=%v), want (false, nil)", found, err)
	}
}

func TestCompareRunWritesRankedHTML(t *testing.T) {
	dataDir := t.TempDir()
	model := []recommendationInput{
		{PlayerId: "player-1", Name: "Low Scorer", Team: "Test FC", Position: "MID", Score: 90},
		{PlayerId: "player-2", Name: "High Scorer", Team: "Test FC", Position: "STR", Score: 80},
		{PlayerId: "player-3", Name: "No Match File", Team: "Test FC", Position: "DEF", Score: 70},
	}
	modelData, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	modelPath := filepath.Join(dataDir, "model.json")
	if err := os.WriteFile(modelPath, modelData, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "player-1-matches.json"), []byte(`{"data":{"items":[{"matchId":"m1","mdLabel":"GW 4","mdPoints":"+2"}]}}`), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "player-2-matches.json"), []byte(`{"data":{"items":[{"matchId":"m2","mdLabel":"GW 4","mdPoints":"+12"}]}}`), 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}

	outputPath := filepath.Join(dataDir, "compare.html")
	compare := Compare{ModelInput: modelPath, Gameweek: 4, DataDir: dataDir, Output: outputPath}
	if err := compare.Run(&Globals{}); err != nil {
		t.Fatalf("Compare.Run() returned an error: %v", err)
	}

	html, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile() returned an error: %v", err)
	}
	page := string(html)
	if !strings.Contains(page, "Low Scorer") || !strings.Contains(page, "High Scorer") || !strings.Contains(page, "No Match File") {
		t.Fatalf("compare page missing expected players:\n%s", page)
	}
	if !strings.Contains(page, "GW4 Score") {
		t.Fatalf("compare page missing gameweek column header:\n%s", page)
	}
}
