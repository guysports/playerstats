package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"

	"guysports/playerstats/pkg/types"
)

type Team struct {
	ID string `name:"id" required:"" help:"Fantasy team UUID to fetch and print"`
}

func (t *Team) Run(globals *Globals) error {
	team, err := loadFantasyTeam(globals.TeamScoringSource, t.ID, globals.DTToken)
	if err != nil {
		return err
	}
	printFantasyTeam(os.Stdout, team)
	return nil
}

func printFantasyTeam(w io.Writer, team *types.FantasyTeam) {
	_, _ = fmt.Fprintf(w, "%s (manager: %s)\n", team.TeamName, team.ManagerName)
	_, _ = fmt.Fprintf(w, "Formation: %s | Total points: %d | Budget remaining: %.1f | Transfers used: %d\n\n",
		team.Formation, team.TotalPoints, team.BudgetRemaining, team.TransfersUsed)

	players := make([]types.FantasyTeamPlayer, len(team.Players))
	copy(players, team.Players)
	sort.SliceStable(players, func(i, j int) bool { return players[i].PositionSlot < players[j].PositionSlot })

	for _, squadPlayer := range players {
		status := "Bench"
		if squadPlayer.IsStarter {
			status = "Start"
		}
		marker := ""
		if squadPlayer.IsCaptain {
			marker = " (C)"
		} else if squadPlayer.IsViceCaptain {
			marker = " (VC)"
		}
		player := squadPlayer.Player
		_, _ = fmt.Fprintf(w, "%2d. [%s] %-4s %-20s %-22s %.1f  %3d pts%s\n",
			squadPlayer.PositionSlot, status, player.Position, player.DisplayName, player.ContestantName, player.Price, squadPlayer.TotalPoints, marker)
	}
}
