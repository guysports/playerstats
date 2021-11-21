package types

var (
	Teams = map[int]string{
		11:  "Everton",
		8:   "Chelsea",
		1:   "Manchester United",
		56:  "Sunderland",
		110: "Stoke",
		31:  "Crystal Palace",
		21:  "West Ham",
		35:  "WBA",
		88:  "Hull City",
		25:  "Middlesborough",
		3:   "Arsenal",
		80:  "Swansea City",
		13:  "Leicester City",
		43:  "Manchester City",
		91:  "AFC Bournemouth",
		14:  "Liverpool",
		90:  "Burnley",
		57:  "Watford",
		20:  "Southampton",
		6:   "Tottenham Hotspur",
		36:  "Brighton",
		4:   "Newcastle",
		38:  "Huddersfield",
		97:  "Cardiff City",
		39:  "Wolves",
		7:   "Aston Villa",
		45:  "Norwich",
		49:  "Sheffield United",
		2:   "Leeds United",
		54:  "Fulham",
		94:  "Brentford",
	}

	Position = map[int]string{
		1: "goalkeeper",
		2: "defender",
		3: "midfielder",
		4: "forward",
	}
)

type (
	Player struct {
		Id             int      `json:"id"`
		FirstName      string   `json:"first_name"`
		LastName       string   `json:"last_name"`
		SquadId        int      `json:"squad_id"`
		Cost           int      `json:"cost"`
		Status         string   `json:"status"`
		InPlayStats    Stats    `json:"stats"`
		Positions      []int    `json:"positions"`
		Locked         int      `json:"locked"`
		InPlayEPLStats EPLStats `json:"epl_stats"`
		Team           string
		Job            string
		CostDisp       string
		Matches        map[string]Match
	}

	Stats struct {
		Prices              map[string]int `json:"prices"`
		Scores              map[string]int `json:"scores"`
		MatchScores         map[string]int `json:"match_scores"`
		WeeklyScores        map[string]int `json:"weekly_scores"`
		DraftScores         map[string]int `json:"draft_scores"`
		RoundRank           int            `json:"round_rank"`
		SeasonRank          int            `json:"season_rank"`
		GamesPlayed         int            `json:"games_played"`
		TotalPoints         int            `json:"total_points"`
		AvgPoints           int            `json:"avg_points"`
		HighScore           int            `json:"high_score"`
		LowScore            int            `json:"low_score"`
		Last3Avg            float32        `json:"last_3_avg"`
		Last5Avg            float32        `json:"last_5_avg"`
		Last3ThisSeasonAvg  int            `json:"last_3_this_season_avg"`
		Last5ThisSeasonAvg  int            `json:"last_5_this_season_avg"`
		Selections          int            `json:"selections"`
		MonthlyTransfersIn  int            `json:"monthly_transfers_in"`
		MonthlyTransfersOut int            `json:"monthly_transfers_out"`
		StarManAwards       int            `json:"star_man_awards"`
		SevenPlusRatings    int            `json:"7_plus_ratings"`
		Goals               int            `json:"goals"`
		Assists             int            `json:"assists"`
		Cards               int            `json:"cards"`
		CleanSheets         int            `json:"clean_sheets"`
	}

	EPLStats struct {
		StarManAwards    int `json:"star_man_awards"`
		SevenPlusRatings int `json:"7_plus_ratings"`
		Goals            int `json:"goals"`
		Assists          int `json:"assists"`
		Cards            int `json:"cards"`
		CleanSheets      int `json:"clean_sheets"`
	}

	PlayerFilter struct {
		Team        string
		Job         string
		Cost        int
		Points      int
		Games       int
		Average     int
		ApplyFilter bool
	}

	MatchWeek struct {
		Id            int     `json:"id"`
		Status        string  `json:"status"`
		MatchesInWeek []Match `json:"matches"`
	}

	Match struct {
		Id            int        `json:"id"`
		Gw            int        `json:"gw"`
		CompetitionId int        `json:"competition_id"`
		HomeSquadId   int        `json:"home_squad_id"`
		AwaySquadId   int        `json:"away_squad_id"`
		VenueId       int        `json:"venue_id"`
		Status        string     `json:"status"`
		Date          string     `json:"date"`
		Stats         MatchStats `json:"stats"`
		HomeScore     int        `json:"home_score"`
		AwayScore     int        `json:"away_score"`
		Completed     string     `json:"completed"`
	}

	MatchStats struct {
		GoalScorers []GoalScorer `json:"GS"`
		RedCards    []Card       `json:"RC"`
		YellowCards []Card       `json:"YC"`
	}

	GoalScorer struct {
		ScorerId int    `json:"player_id"`
		AssistId int    `json:"assist_player_id"`
		Min      int    `json:"min"`
		Type     string `json:"type"`
		Period   int    `json:"period"`
	}

	Card struct {
		PlayerId int `json:"player_id"`
		Min      int `json:"min"`
		Period   int `json:"period"`
	}

	Squad struct {
		ID            int        `json:"id"`
		CompetitionID int        `json:"competition_id"`
		Name          string     `json:"full_name"`
		Stats         SquadStats `json:"stats"`
	}

	SquadStats struct {
		Goals       int      `json:"goals"`
		CleanSheets int      `json:"clean_sheets"`
		Cards       int      `json:"cards"`
		ClubForm    []string `json:"club_form"`
		Rank        int      `json:"club_form_rank"`
		Conceded    int      `json:"GC"`
	}

	Competition struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}

	DataMaps struct {
		SquadMap      map[int]Squad
		CompetitonMap map[int]Competition
	}

	RenderedPlayer struct {
		Id           int
		Position     string
		Name         string
		Team         string
		Cost         string
		TotalPoints  int
		GamesPlayed  int
		StarMan      int
		SevenPlus    int
		Goals        int
		Assists      int
		CleanSheets  int
		Cards        int
		Last3Avg     float32
		Last5Avg     float32
		TeamFixtures []RenderedMatch
	}

	RenderedMatch struct {
		Gw          int
		Competition string
		Fixture     string
		Result      string
		Date        string
	}
)
