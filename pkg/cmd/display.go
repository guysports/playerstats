package cmd

import (
	"encoding/json"
	"fmt"
	"guysports/playerstats/pkg/helper"
	"guysports/playerstats/pkg/types"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/jedib0t/go-pretty/v6/table"

	goftp "gopkg.in/dutchcoders/goftp.v1"
)

type (
	Display struct {
		PlayerNames []string `help:"name of players whose stats to be viewed"`
		Sort        string   `help:"what to sort the lists by, supported are position, team"`
		Filter      []string `help:"apply criteria to players to display, with separated filter=value,filter=value list"`
		Matches     bool     `help:"temporary option to display match info"`
		Html        bool     `help:"format player info into html pages"`
	}
)

func (d *Display) Run(globals *Globals) error {
	// Load in the statistics data from source
	players := []types.Player{}
	matchweeks := []types.MatchWeek{}
	squadMap := globals.SquadMap
	competitionMap := globals.CompetitionMap
	var data []byte
	var err error

	// Obtain the player data
	data, err = helper.GetJSON(globals.Source)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(data, &players)

	// Obtain the game week data
	data, err = helper.GetJSON(globals.MatchesSource)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(data, &matchweeks)

	// Extract matches from each week
	matchesMap := map[string]types.Match{}
	for _, matchweek := range matchweeks {
		for _, match := range matchweek.MatchesInWeek {
			key := fmt.Sprintf("%d", match.Id)
			matchesMap[key] = match
		}
	}

	// The match Ids in the player struct are unreliable so get the matches by player squad id
	for i, player := range players {
		players[i].Matches = map[string]types.Match{}
		for _, match := range matchesMap {
			if player.SquadId == match.HomeSquadId || player.SquadId == match.AwaySquadId {
				players[i].Matches[fmt.Sprintf("%d", match.Id)] = match
			}
		}
	}

	filteredPlayers := []types.Player{}

	// Add team and position to each player
	for i, player := range players {
		// Check any filters and only add player if filter is met
		filter := parseFilters(d.Filter)

		// Check player against filter
		filteredPlayer := checkPlayerValid(&player, filter)
		if filteredPlayer == nil {
			continue
		}
		cost := float64(player.Cost) / 1000000
		filteredPlayer.Team = types.Teams[player.SquadId]
		filteredPlayer.Job = types.Position[player.Positions[0]]
		filteredPlayer.CostDisp = fmt.Sprintf("&pound;%.2fm", cost)
		filteredPlayers = append(filteredPlayers, *filteredPlayer)

		players[i].Team = types.Teams[player.SquadId]
		players[i].Job = types.Position[player.Positions[0]]
		players[i].CostDisp = fmt.Sprintf("&pound;%.2fm", cost)
	}

	if d.Html {
		// Generate rendered player and match information
		renderPlayers := []types.RenderedPlayer{}
		for _, player := range players {
			renderedPlayer := types.RenderedPlayer{
				Id:          player.Id,
				Name:        fmt.Sprintf("%s %s", player.FirstName, player.LastName),
				Team:        player.Team,
				Position:    player.Job,
				Cost:        player.CostDisp,
				TotalPoints: player.InPlayStats.TotalPoints,
				GamesPlayed: player.InPlayStats.GamesPlayed,
				StarMan:     player.InPlayStats.StarManAwards,
				SevenPlus:   player.InPlayStats.SevenPlusRatings,
				Goals:       player.InPlayStats.Goals,
				Assists:     player.InPlayStats.Assists,
				CleanSheets: player.InPlayStats.CleanSheets,
				Cards:       player.InPlayStats.Cards,
				Last3Avg:    player.InPlayStats.Last3Avg,
				Last5Avg:    player.InPlayStats.Last5Avg,
			}
			// Generate rendered fixtures
			renderedFixtures := []types.RenderedMatch{}
			for _, match := range player.Matches {
				if match.Status == "complete" {
					fixture := types.RenderedMatch{
						Gw:          match.Gw,
						Competition: competitionMap[match.CompetitionId].Name,
						Fixture:     fmt.Sprintf("%s v %s", squadMap[match.HomeSquadId].Name, squadMap[match.AwaySquadId].Name),
						Result:      fmt.Sprintf("%d v %d", match.HomeScore, match.AwayScore),
						Date:        strings.Split(match.Date, "T")[0],
					}
					renderedFixtures = append(renderedFixtures, fixture)
				}
			}

			sort.Slice(renderedFixtures, func(i, j int) bool {
				if renderedFixtures[i].Date > renderedFixtures[j].Date {
					return true
				}
				return false
			})

			renderedPlayer.TeamFixtures = renderedFixtures
			renderPlayers = append(renderPlayers, renderedPlayer)
		}
		formatAsHtml(renderPlayers, globals.FtpPassword)
		return nil
	}

	if d.PlayerNames == nil || d.PlayerNames[0] == "all" {
		displayPlayerInfo(filteredPlayers, matchesMap, competitionMap, squadMap, d.Sort)
	} else {
		selectPlayers := []types.Player{}
		for _, player := range d.PlayerNames {
			for _, playerstat := range players {
				if player == playerstat.LastName {
					selectPlayers = append(selectPlayers, playerstat)
				}
			}
		}
		displayPlayerInfo(selectPlayers, matchesMap, competitionMap, squadMap, d.Sort)
	}
	return nil
}

func displayPlayerInfo(players []types.Player, matches map[string]types.Match, competitions map[int]types.Competition, squads map[int]types.Squad, criteria string) {
	if criteria != "" {
		switch criteria {
		case "position":
			sort.Slice(players, func(i, j int) bool {
				if players[i].Positions[0] < players[j].Positions[0] {
					return true
				}
				if players[i].Positions[0] > players[j].Positions[0] {
					return false
				}
				return players[i].SquadId < players[j].SquadId
			})
			break
		case "team":
			sort.Slice(players, func(i, j int) bool {
				return players[i].SquadId < players[j].SquadId
			})
			break
		}
	}

	var pageSize int
	for _, player := range players {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Position", "Player", "Team", "Cost", "Total Points"})
		cost := float64(player.Cost) / 1000000
		t.AppendRow(table.Row{player.Job, fmt.Sprintf("%s %s", player.FirstName, player.LastName), player.Team, fmt.Sprintf("£%.2fm", cost), player.InPlayStats.TotalPoints})
		t.AppendRow(table.Row{"Games Played", "Star Man Awards", "+7 Ratings", "Goals", "Assists", "Clean Sheets", "Cards"})
		t.AppendRow(table.Row{player.InPlayStats.GamesPlayed,
			player.InPlayStats.StarManAwards,
			player.InPlayStats.SevenPlusRatings,
			player.InPlayStats.Goals,
			player.InPlayStats.Assists,
			player.InPlayStats.CleanSheets,
			player.InPlayStats.Cards})
		t.AppendRow(table.Row{"Date", "Competition", "Fixture", "Score", "Points Scored"})
		gameRows := []table.Row{}
		completedMatches := []types.Match{}
		for _, match := range player.Matches {
			if match.Status == "complete" {
				completedMatches = append(completedMatches, match)
			}
		}

		sort.Slice(completedMatches, func(i, j int) bool {
			if completedMatches[i].Date > completedMatches[j].Date {
				return false
			}
			return true
		})

		for _, match := range completedMatches {
			gameRows = append(gameRows, table.Row{
				strings.Split(match.Date, "T")[0],
				competitions[match.CompetitionId].Name,
				fmt.Sprintf("%s v %s", squads[match.HomeSquadId].Name, squads[match.AwaySquadId].Name),
				fmt.Sprintf("%d v %d", match.HomeScore, match.AwayScore),
				fmt.Sprintf("%d", player.InPlayStats.MatchScores[fmt.Sprintf("%d", match.Id)]),
			})
		}
		t.AppendRows(gameRows)
		t.SetPageSize(pageSize)
		t.Render()
		t = nil
	}

}

func formatAsHtml(players []types.RenderedPlayer, password string) {
	tmpl, err := template.ParseFiles("pkg/cmd/player.template")
	if err != nil {
		fmt.Printf("Error templating %v\n", err)
		return
	}
	for _, player := range players {
		f, err := os.Create(fmt.Sprintf("players/%d.php", player.Id))
		if err != nil {
			fmt.Printf("Error opening file %v\n", err)
			return
		}
		tmpl.Execute(f, player)
		f.Close()
	}
	err = uploadPlayerStats(players, password)
	if err != nil {
		fmt.Printf("error uploading playerstats %v\n", err)
	}
}

func uploadPlayerStats(players []types.RenderedPlayer, password string) error {
	// FTP file to guysports
	ftp, err := goftp.Connect("ftp.guysports.co.uk:21")
	if err != nil {
		return err
	}

	defer ftp.Close()
	// Username / password authentication
	if err = ftp.Login("guysports@guysports.co.uk", password); err != nil {
		return err
	}

	if err = ftp.Cwd("/public_html/guysports/players"); err != nil {
		return err
	}

	for _, player := range players {
		localFilename := fmt.Sprintf("players/%d.php", player.Id)
		remoteFilename := fmt.Sprintf("%d.php", player.Id)

		// Upload player stats
		file, err := os.Open(localFilename)
		if err != nil {
			return err
		}

		if err := ftp.Stor(remoteFilename, file); err != nil {
			return err
		}
		fmt.Printf("uploaded %s for %s\n", remoteFilename, player.Name)
	}
	return nil
}

func parseFilters(filters []string) *types.PlayerFilter {
	playerFilter := types.PlayerFilter{
		ApplyFilter: false,
	}
	if filters == nil {
		return &playerFilter
	}
	for _, filterValuePair := range filters {
		pair := strings.Split(filterValuePair, "=")
		if len(pair) != 2 {
			continue
		}
		filter := pair[0]
		value := pair[1]
		switch filter {
		case "team":
			playerFilter.Team = value
			playerFilter.ApplyFilter = true
			break
		case "cost":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Cost = intValue
			playerFilter.ApplyFilter = true
			break
		case "points":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Points = intValue
			playerFilter.ApplyFilter = true
			break
		case "games":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Games = intValue
			playerFilter.ApplyFilter = true
			break
		case "average":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Cost = intValue
			playerFilter.ApplyFilter = true
			break
		case "position":
			playerFilter.Job = value
			playerFilter.ApplyFilter = true
			break
		}
	}
	return &playerFilter
}

func checkPlayerValid(player *types.Player, filter *types.PlayerFilter) *types.Player {
	// Check each filter value in turn, and progress if not used or matches
	if !filter.ApplyFilter {
		return player
	}
	if filter.Average > 0 {
		if player.InPlayStats.AvgPoints < filter.Average {
			return nil
		}
	}
	if filter.Cost > 0 {
		if player.Cost > filter.Cost {
			return nil
		}
	}
	if filter.Points > 0 {
		if player.InPlayStats.TotalPoints < filter.Points {
			return nil
		}
	}
	if filter.Games > 0 {
		if player.InPlayStats.GamesPlayed < filter.Games {
			return nil
		}
	}
	if filter.Team != "" {
		if types.Teams[player.SquadId] != filter.Team {
			return nil
		}
	}
	if filter.Job != "" {
		if types.Position[player.Positions[0]] != filter.Job {
			return nil
		}
	}
	return player
}
