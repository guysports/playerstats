package cmd

import "guysports/playerstats/pkg/types"

type Globals struct {
	FtpPassword       string `env:"GSADMIN_PW" required:"yes"`
	Operation         string `env:"PLAYER_OPERATION" envDefault:"all"`
	Source            string `env:"STATS_SOURCE" envDefault:"https://nuk-data.s3-eu-west-1.amazonaws.com/json/players.json"`
	MatchesSource     string `env:"MATCHES_SOURCE" envDefault:"https://nuk-data.s3-eu-west-1.amazonaws.com/json/gameweeks.json"`
	SquadSource       string `env:"SQUAD_SOURCE" envDefault:"https://nuk-data.s3-eu-west-1.amazonaws.com/json/squads.json"`
	CompetitionSource string `env:"COMPETITION_SOURCE" envDefault:"https://nuk-data.s3-eu-west-1.amazonaws.com/json/competitions.json"`
	SquadMap          map[int]types.Squad
	CompetitionMap    map[int]types.Competition
}
