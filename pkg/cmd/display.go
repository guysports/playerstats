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
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/jlaffaye/ftp"
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

func formatDate(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.Local().Format("15:04 02-Jan-06")
}

func getGameWeekMatches(data []byte) ([]types.GameWeekMatch, error) {
	gameWeek := types.GameWeek{}
	if err := json.Unmarshal(data, &gameWeek); err != nil {
		return nil, err
	}
	return gameWeek.Data.Items, nil
}

func loadPlayers(source string) ([]types.Player, error) {
	data, err := helper.GetJSON(source)
	if err != nil {
		return nil, err
	}

	players := []types.Player{}
	if err := json.Unmarshal(data, &players); err != nil {
		return nil, err
	}
	return players, nil
}

func (d *Display) Run(globals *Globals) error {
	players, err := loadPlayers(globals.Source)
	if err != nil {
		return err
	}
	var data []byte
	_ = json.Unmarshal(data, &players)

	// Initialize empty match maps for each player (matching by numeric squad id not available)
	for i, player := range players {
		// Obtain the match result data for each player
		data, err = helper.GetJSON(fmt.Sprintf(globals.MatchesSource, player.PlayerId))
		if err != nil {
			return err
		}
		players[i].Results, err = getGameWeekMatches(data)
		if err != nil {
			return err
		}
	}

	filteredPlayers := []types.Player{}

	// Add display-only fields to each player.
	for i, player := range players {
		filter := parseFilters(d.Filter)
		filteredPlayer := checkPlayerValid(&player, filter)
		if filteredPlayer == nil {
			continue
		}
		cost := player.Price
		filteredPlayer.CostDisp = fmt.Sprintf("&pound;%.2fm", cost)
		filteredPlayers = append(filteredPlayers, *filteredPlayer)

		players[i].CostDisp = fmt.Sprintf("&pound;%.2fm", cost)
	}

	if d.Html {
		renderPlayers := []types.RenderedPlayer{}
		for _, player := range players {
			renderedPlayer := types.RenderedPlayer{
				Id:             player.PlayerId,
				Name:           fmt.Sprintf("%s %s", player.FirstName, player.LastName),
				Team:           player.ContestantShortName,
				Position:       player.Position,
				Cost:           player.CostDisp,
				AveragePoints:  player.AveragePoints,
				Last3Average:   player.Last3Average,
				TotalPoints:    player.TotalPoints,
				Goals:          player.Goals,
				Assists:        player.Assists,
				ShotsOnTarget:  player.ShotsOnTarget,
				ChancesCreated: player.ChancesCreated,
				Tackles:        player.Tackles,
			}

			// Generate rendered fixtures
			renderedFixtures := []types.RenderedMatch{}
			for _, match := range player.NextGameweekFixtures {
				competition := "Premier League"
				if match.CompetitionName != nil {
					competition = *match.CompetitionName
				}
				matchFixture := fmt.Sprintf("%s v %s", player.ContestantShortName, match.OpponentShortName)
				if !match.IsHome {
					matchFixture = fmt.Sprintf("%s v %s", match.OpponentShortName, player.ContestantShortName)
				}
				fixture := types.RenderedMatch{
					Gw:          match.GameWeek,
					Competition: competition,
					Fixture:     matchFixture,
					Venue:       match.Venue,
					KickOff:     formatDate(match.KickoffAt),
				}
				renderedFixtures = append(renderedFixtures, fixture)
			}

			sort.Slice(renderedFixtures, func(i, j int) bool {
				return renderedFixtures[i].KickOff > renderedFixtures[j].KickOff
			})

			renderedResults := []types.RenderedResult{}
			for _, result := range player.Results {
				competition := "Premier League"
				if result.CompetitionLabel != "" {
					competition = result.CompetitionLabel
				}
				renderedResult := types.RenderedResult{
					GameWeek:    result.MdLabel,
					Competition: competition,
					HomeTeam:    result.LeftTeam.ShortName,
					AwayTeam:    result.RightTeam.ShortName,
					HomeScore:   result.LeftTeam.Score,
					AwayScore:   result.RightTeam.Score,
					Venue:       result.Venue,
					KickOff:     formatDate(result.KickoffAt),
				}
				renderedResults = append(renderedResults, renderedResult)
			}

			renderedPlayer.TeamFixtures = renderedFixtures
			renderedPlayer.TeamResults = renderedResults
			renderPlayers = append(renderPlayers, renderedPlayer)
		}
		formatAsHtml(renderPlayers, globals.FtpPassword)
		return nil
	}

	if d.PlayerNames == nil || d.PlayerNames[0] == "all" {
		displayPlayerInfo(filteredPlayers, d.Sort)
	} else {
		selectPlayers := []types.Player{}
		for _, name := range d.PlayerNames {
			for _, playerstat := range players {
				if name == playerstat.LastName {
					selectPlayers = append(selectPlayers, playerstat)
				}
			}
		}
		displayPlayerInfo(selectPlayers, d.Sort)
	}
	return nil
}

func displayPlayerInfo(players []types.Player, criteria string) {
	if criteria != "" {
		switch criteria {
		case "position":
			sort.Slice(players, func(i, j int) bool {
				if players[i].Position < players[j].Position {
					return true
				}
				if players[i].Position > players[j].Position {
					return false
				}
				return players[i].ContestantShortName < players[j].ContestantShortName
			})
		case "team":
			sort.Slice(players, func(i, j int) bool {
				return players[i].ContestantShortName < players[j].ContestantShortName
			})
		}
	}

	for _, player := range players {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Position", "Player", "Team", "Cost", "Total Points"})
		cost := player.Price
		t.AppendRow(table.Row{player.Position, fmt.Sprintf("%s %s", player.FirstName, player.LastName), player.ContestantShortName, fmt.Sprintf("£%.2fm", cost), player.TotalPoints})
		t.AppendRow(table.Row{"Average Points", "Last 3 Average", "Goals", "Assists", "Shots on Target"})
		t.AppendRow(table.Row{player.AveragePoints,
			player.Last3Average,
			player.Goals,
			player.Assists,
			player.ShotsOnTarget})
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
		f, err := os.Create(fmt.Sprintf("players/%s.php", player.Id))
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
	ftpClient, err := ftp.Dial("ftp.guysports.co.uk:21")
	if err != nil {
		return err
	}

	defer ftpClient.Quit()
	// Username / password authentication
	if err = ftpClient.Login("guysports@guysports.co.uk", password); err != nil {
		return err
	}

	if err = ftpClient.ChangeDir("/public_html/guysports/players"); err != nil {
		return err
	}

	for _, player := range players {
		localFilename := fmt.Sprintf("players/%s.php", player.Id)
		remoteFilename := fmt.Sprintf("%s.php", player.Id)

		// Upload player stats
		file, err := os.Open(localFilename)
		if err != nil {
			return err
		}

		if err := ftpClient.Stor(remoteFilename, file); err != nil {
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
		case "cost":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Cost = intValue
			playerFilter.ApplyFilter = true
		case "points":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Points = intValue
			playerFilter.ApplyFilter = true
		case "average":
			intValue, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			playerFilter.Average = intValue
			playerFilter.ApplyFilter = true
		case "position":
			playerFilter.Job = value
			playerFilter.ApplyFilter = true
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
		if player.AveragePoints < float64(filter.Average) {
			return nil
		}
	}
	if filter.Cost > 0 {
		if int(player.Price) > filter.Cost {
			return nil
		}
	}
	if filter.Points > 0 {
		if player.TotalPoints < filter.Points {
			return nil
		}
	}
	if filter.Team != "" {
		if player.ContestantName != filter.Team && player.ContestantShortName != filter.Team {
			return nil
		}
	}
	if filter.Job != "" {
		if player.Position != filter.Job {
			return nil
		}
	}
	return player
}
