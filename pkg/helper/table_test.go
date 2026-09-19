package helper

import "testing"

const standingsFixture = `{
  "standings": [
    {
      "type": "TOTAL",
      "table": [
        {"position": 1, "team": {"name": "Liverpool FC"}, "playedGames": 3, "won": 3, "draw": 0, "lost": 0, "points": 9, "goalsFor": 7, "goalsAgainst": 1, "goalDifference": 6},
        {"position": 2, "team": {"name": "Manchester City FC"}, "playedGames": 3, "won": 2, "draw": 1, "lost": 0, "points": 7, "goalsFor": 6, "goalsAgainst": 2, "goalDifference": 4}
      ]
    }
  ]
}`

const matchesFixture = `{
  "matches": [
    {"utcDate": "2026-08-01T15:00:00Z", "status": "FINISHED", "homeTeam": {"name": "Liverpool FC"}, "awayTeam": {"name": "Manchester City FC"}, "score": {"winner": "HOME_TEAM"}},
    {"utcDate": "2026-08-08T15:00:00Z", "status": "FINISHED", "homeTeam": {"name": "Manchester City FC"}, "awayTeam": {"name": "Liverpool FC"}, "score": {"winner": "DRAW"}},
    {"utcDate": "2026-08-15T15:00:00Z", "status": "SCHEDULED", "homeTeam": {"name": "Liverpool FC"}, "awayTeam": {"name": "Manchester City FC"}, "score": {"winner": null}}
  ]
}`

func TestParseFootballDataStandings(t *testing.T) {
	entries, err := ParseFootballDataStandings([]byte(standingsFixture))
	if err != nil {
		t.Fatalf("ParseFootballDataStandings() returned an error: %v", err)
	}
	if len(entries) != 2 || entries[0].Team != "Liverpool FC" || entries[0].Position != 1 || entries[0].Points != 9 {
		t.Fatalf("entries = %+v, want Liverpool FC in first place with 9 points", entries)
	}
}

func TestApplyRecentForm(t *testing.T) {
	entries, err := ParseFootballDataStandings([]byte(standingsFixture))
	if err != nil {
		t.Fatalf("ParseFootballDataStandings() returned an error: %v", err)
	}
	if err := ApplyRecentForm(entries, []byte(matchesFixture)); err != nil {
		t.Fatalf("ApplyRecentForm() returned an error: %v", err)
	}
	if entries[0].RecentForm != "WD" || entries[0].FormPoints != 4 {
		t.Fatalf("entries[0] = %+v, want Liverpool FC form WD with 4 points", entries[0])
	}
	if entries[1].RecentForm != "LD" || entries[1].FormPoints != 1 {
		t.Fatalf("entries[1] = %+v, want Manchester City FC form LD with 1 point", entries[1])
	}
}
