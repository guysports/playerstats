package cmd

import (
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	input := "2026-09-12T19:00:00.000Z"
	parsed, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		t.Fatalf("time.Parse() returned an error: %v", err)
	}
	want := parsed.Local().Format("15:04 02-Jan-06")
	if got := formatDate(input); got != want {
		t.Fatalf("formatDate() = %q, want local time %q", got, want)
	}
}

func TestFormatDateInvalidInput(t *testing.T) {
	input := "not-a-date"
	if got := formatDate(input); got != input {
		t.Fatalf("formatDate() = %q, want unchanged input %q", got, input)
	}
}

func TestGetGameWeekMatches(t *testing.T) {
	data := []byte(`{
		"success": true,
		"data": {
			"items": [{
				"matchId": "fda2af01-0770-4e50-896c-a88a507c21de",
				"competitionLabel": "Premier League",
				"kickoffAt": "2026-08-23T13:00:00.000Z",
				"venue": "Etihad Stadium",
				"status": "FT",
				"statusVariant": "finished",
				"mdLabel": "GW 1",
				"mdPoints": "+4",
				"periodId": "85eb8200-6ee9-4b08-8ce5-27585c7839f2",
				"leftTeam": {
					"name": "Manchester City",
					"shortName": "Man City",
					"flagKey": "MCI",
					"score": 2
				},
				"rightTeam": {
					"name": "Bournemouth",
					"shortName": "Bournemouth",
					"flagKey": "BOU",
					"score": 1
				},
				"stats": [{
					"label": "Minutes played",
					"total": 90,
					"points": "+2"
				}]
			}]
		}
	}`)

	matches, err := getGameWeekMatches(data)
	if err != nil {
		t.Fatalf("getGameWeekMatches returned an error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("getGameWeekMatches returned %d matches, want 1", len(matches))
	}

	match := matches[0]
	if match.MatchId != "fda2af01-0770-4e50-896c-a88a507c21de" {
		t.Errorf("MatchId = %q, want the supplied match ID", match.MatchId)
	}
	if match.LeftTeam.ShortName != "Man City" || match.LeftTeam.Score != 2 {
		t.Errorf("LeftTeam = %+v, want Man City with score 2", match.LeftTeam)
	}
	total, err := match.Stats[0].TotalAsInt()
	if err != nil {
		t.Fatalf("TotalAsInt returned an error: %v", err)
	}
	if len(match.Stats) != 1 || total != 90 || match.Stats[0].Points != "+2" {
		t.Errorf("Stats = %+v, want one stat with total 90 and points +2", match.Stats)
	}
}

func TestGetGameWeekMatchesStringTotal(t *testing.T) {
	data := []byte(`{
		"success": true,
		"data": {
			"items": [{
				"matchId": "fda2af01-0770-4e50-896c-a88a507c21de",
				"competitionLabel": "Premier League",
				"kickoffAt": "2026-08-23T13:00:00.000Z",
				"venue": "Etihad Stadium",
				"status": "FT",
				"statusVariant": "finished",
				"mdLabel": "GW 1",
				"mdPoints": "+4",
				"periodId": "85eb8200-6ee9-4b08-8ce5-27585c7839f2",
				"leftTeam": {
					"name": "Manchester City",
					"shortName": "Man City",
					"flagKey": "MCI",
					"score": 2
				},
				"rightTeam": {
					"name": "Bournemouth",
					"shortName": "Bournemouth",
					"flagKey": "BOU",
					"score": 1
				},
				"stats": [{
					"label": "Minutes played",
					"total": "90",
					"points": "+2"
				}]
			}]
		}
	}`)

	matches, err := getGameWeekMatches(data)
	if err != nil {
		t.Fatalf("getGameWeekMatches returned an error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("getGameWeekMatches returned %d matches, want 1", len(matches))
	}
	total, err := matches[0].Stats[0].TotalAsInt()
	if err != nil {
		t.Fatalf("TotalAsInt returned an error: %v", err)
	}
	if total != 90 {
		t.Fatalf("Stats[0].TotalAsInt() = %d, want 90", total)
	}
}

func TestGetGameWeekMatchesInvalidJSON(t *testing.T) {
	matches, err := getGameWeekMatches([]byte(`{"data":`))
	if err == nil {
		t.Fatal("getGameWeekMatches returned nil error for invalid JSON")
	}
	if matches != nil {
		t.Errorf("getGameWeekMatches returned %v matches, want nil", matches)
	}
}
