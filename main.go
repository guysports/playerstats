package main

import (
	"fmt"
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
	Login     cmd.Login     `cmd:"" help:"Sign in to Dream Team and save its API token"`
}

func main() {
	ctx := kong.Parse(&cli)
	globals := cmd.Globals{}
	if err := env.Parse(&globals); err != nil {
		ctx.FatalIfErrorf(err)
	}
	if ctx.Command() != "login" {
		if globals.FtpPassword == "" {
			ctx.FatalIfErrorf(fmt.Errorf("GSADMIN_PW is required"))
		}
		if globals.DTToken == "" {
			var err error
			globals.DTToken, err = cmd.LoadDreamTeamToken()
			ctx.FatalIfErrorf(err)
		}
	}

	err := ctx.Run(&globals)
	ctx.FatalIfErrorf(err)
}
