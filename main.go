package main

import (
	"encoding/json"
	"guysports/playerstats/pkg/cmd"
	"guysports/playerstats/pkg/helper"
	"guysports/playerstats/pkg/types"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env"
)

var cli struct {
	Display cmd.Display `cmd:"" help:"Show the player statistics for requested players"`
	Match   cmd.Match   `cmd:"" help:"Display fixture statistics"`
}

func main() {
	globals := cmd.Globals{}
	env.Parse(&globals)
	ctx := kong.Parse(&cli)

	// Obtain the squad information
	data, err := helper.GetJSON(globals.SquadSource)
	ctx.FatalIfErrorf(err)
	squads := []types.Squad{}
	_ = json.Unmarshal(data, &squads)

	// Obtain the competition information
	data, err = helper.GetJSON(globals.CompetitionSource)
	ctx.FatalIfErrorf(err)
	competitions := []types.Competition{}
	_ = json.Unmarshal(data, &competitions)

	// Extract competitions into map
	globals.CompetitionMap = map[int]types.Competition{}
	for _, competition := range competitions {
		globals.CompetitionMap[competition.ID] = competition
	}

	// Extract squads into map
	globals.SquadMap = map[int]types.Squad{}
	for _, squad := range squads {
		globals.SquadMap[squad.ID] = squad
	}

	err = ctx.Run(&globals)
	ctx.FatalIfErrorf(err)
}
