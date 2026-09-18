package helper

import (
	"strings"
	"testing"
)

func TestParsePremierLeagueTable(t *testing.T) {
	html := `<table><tr><th>Team</th><th>Played</th><th>Won</th><th>Drawn</th><th>Lost</th><th>Goals For</th><th>Goals Against</th><th>Goal Difference</th><th>Points</th><th>Form</th></tr><tr><td>1</td><td><a>Liverpool</a></td><td>3</td><td>3</td><td>0</td><td>0</td><td>7</td><td>1</td><td>6</td><td>9</td><td>W W D L W W</td></tr><tr><td>2</td><td>Manchester City</td><td>3</td><td>2</td><td>1</td><td>0</td><td>6</td><td>2</td><td>4</td><td>7</td><td>W D W</td></tr></table>`
	entries, err := ParsePremierLeagueTable(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParsePremierLeagueTable() returned an error: %v", err)
	}
	if len(entries) != 2 || entries[0].Team != "Liverpool" || entries[0].Position != 1 || entries[0].Points != 9 || entries[0].RecentForm != "WWDLWW" || entries[0].FormPoints != 13 {
		t.Fatalf("entries = %+v, want Liverpool in first place with 9 points", entries)
	}
}
