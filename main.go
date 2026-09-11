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
}

func main() {
	globals := cmd.Globals{}
	env.Parse(&globals)
	ctx := kong.Parse(&cli)

	err := ctx.Run(&globals)
	ctx.FatalIfErrorf(err)
}
