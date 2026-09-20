package cmd

import (
	"strings"
	"testing"

	"guysports/playerstats/pkg/types"
)

func TestPrintFantasyTeamOrdersBySlot(t *testing.T) {
	team := &types.FantasyTeam{
		TeamName: "Test Team", ManagerName: "G. Barden", Formation: "3-5-2", TotalPoints: 505,
		Players: []types.FantasyTeamPlayer{
			{PositionSlot: 9, IsStarter: true, TotalPoints: 59, Player: types.FantasyTeamPlayerDetail{DisplayName: "E. Haaland", Position: "STR", ContestantName: "Manchester City", Price: 14.5}},
			{PositionSlot: 0, IsStarter: true, IsCaptain: true, TotalPoints: 39, Player: types.FantasyTeamPlayerDetail{DisplayName: "D. Raya", Position: "GK", ContestantName: "Arsenal", Price: 5.1}},
		},
	}
	var buf strings.Builder
	printFantasyTeam(&buf, team)
	output := buf.String()

	if strings.Index(output, "D. Raya") > strings.Index(output, "E. Haaland") {
		t.Fatalf("expected D. Raya (slot 0) before E. Haaland (slot 9), got:\n%s", output)
	}
	if !strings.Contains(output, "(C)") {
		t.Fatalf("expected captain marker in output:\n%s", output)
	}
	if !strings.Contains(output, "Test Team") || !strings.Contains(output, "3-5-2") {
		t.Fatalf("expected team header details in output:\n%s", output)
	}
}
