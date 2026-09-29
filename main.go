package main

import (
	"guysports/playerstats/pkg/cmd"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env"
)

var cli struct {
	Display   cmd.Display   `cmd:"" help:"Show the player statistics for requested players"`
	Player    cmd.Player    `cmd:"" help:"Display player scores from last season"`
	Dump      cmd.Dump      `cmd:"" help:"Dump all player data to data/players.json"`
	Recommend cmd.Recommend `cmd:"" help:"Rank upcoming player opportunities and explain them with Ollama"`
	Compare   cmd.Compare   `cmd:"" help:"Compare a model-only recommendation file against actual gameweek results"`
	Team      cmd.Team      `cmd:"" help:"Print a fantasy team's current squad"`
	FTP       cmd.FTP       `cmd:"" help:"Test the FTP connection and list the players directory"`
}

func main() {
	globals := cmd.Globals{}
	if err := env.Parse(&globals); err != nil {
		panic(err)
	}
	ctx := kong.Parse(&cli)

	err := ctx.Run(&globals)
	ctx.FatalIfErrorf(err)
}
