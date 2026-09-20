package cmd

import "time"

type Globals struct {
	FtpPassword           string        `env:"GSADMIN_PW" required:"yes"`
	Operation             string        `env:"PLAYER_OPERATION" envDefault:"all"`
	Source                string        `env:"STATS_SOURCE" envDefault:"https://guysports.co.uk/gsadmin/players/1.json"`
	MatchesSource         string        `env:"MATCHES_SOURCE" envDefault:"https://engagecraft-fantasy-backend-prod.azurewebsites.net/api/players/%s/matches?type=gameweek"`
	MatchStatsSource      string        `env:"MATCH_STATS_SOURCE" envDefault:"https://engagecraft-fantasy-backend-prod.azurewebsites.net/api/matches/%s/stats"`
	GameweekMatchesSource string        `env:"GAMEWEEK_MATCHES_SOURCE" envDefault:"https://engagecraft-fantasy-backend-prod.azurewebsites.net/api/matches?type=gameweek&tournamentCalendarId=be2cf2c1-42ad-4aa2-8156-692b2bf03b09&gameweek=%d"`
	TeamScoringSource     string        `env:"TEAM_SCORING_SOURCE" envDefault:"https://engagecraft-fantasy-backend-prod.azurewebsites.net/api/teams/scoring/%s"`
	DTToken               string        `env:"DT_TOKEN" required:"yes"`
	DataFTPDirectory      string        `env:"DATA_FTP_DIRECTORY" envDefault:"/subdomains/playerinsights/data"`
	OllamaURL             string        `env:"OLLAMA_URL" envDefault:"http://localhost:11434"`
	OllamaModel           string        `env:"OLLAMA_MODEL" envDefault:"llama3.1:8b"`
	OllamaTimeout         time.Duration `env:"OLLAMA_TIMEOUT" envDefault:"10m"`
	LeagueTableURL        string        `env:"LEAGUE_TABLE_URL" envDefault:"https://api.football-data.org/v4/competitions/PL"`
	FootballDataAPIToken  string        `env:"FOOTBALL_DATA_API_TOKEN"`
	BetfairAppKey         string        `env:"BETFAIR_APP_KEY"`
}
