package cmd

import (
	"encoding/json"
	"fmt"
	"guysports/playerstats/pkg/helper"
	"guysports/playerstats/pkg/types"
)

type (
	Match struct {
		Gw     int      `help:"List the fixtures for the game week"`
		Filter []string `help:"apply criteria to fixtures to display, with separated filter=value,filter=value list"`
	}
)

func (m *Match) Run(globals *Globals) error {
	// Obtain the game week data
	data, err := helper.GetJSON(globals.MatchesSource)
	if err != nil {
		return err
	}
	matchweeks := []types.MatchWeek{}
	_ = json.Unmarshal(data, &matchweeks)

	gwFixtures := matchweeks[m.Gw-1]
	for _, fixture := range gwFixtures.MatchesInWeek {
		fmt.Printf("%s %d v %d %s\n", globals.SquadMap[fixture.HomeSquadId].Name, fixture.HomeScore, fixture.AwayScore, globals.SquadMap[fixture.AwaySquadId].Name)
	}

	//homeForm, awayForm := loadForm(matchweeks)

	return nil
}

func loadForm(mw []types.MatchWeek) (map[int][]types.Match, map[int][]types.Match) {
	homeForm := map[int][]types.Match{}
	awayForm := map[int][]types.Match{}
	for _, week := range mw {
		if week.Status != "complete" {
			continue
		}

		// For each complete week, extract each match and add the matches to the maps based on a squadID key
		for _, match := range week.MatchesInWeek {
			_, ok := homeForm[match.HomeSquadId]
			if !ok {
				homeForm[match.HomeSquadId] = []types.Match{
					match,
				}
			} else {
				homeForm[match.HomeSquadId] = append(homeForm[match.HomeSquadId], match)
			}
			_, ok = awayForm[match.AwaySquadId]
			if !ok {
				awayForm[match.AwaySquadId] = []types.Match{
					match,
				}
			} else {
				awayForm[match.AwaySquadId] = append(awayForm[match.AwaySquadId], match)
			}
		}
	}
	return homeForm, awayForm
}
