package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// (No custom unmarshal - rely on default JSON decoding into Player)

type (
	// Player represents a single player and their stats as returned by the new JSON schema.
	Player struct {
		PlayerId             string    `json:"playerId"`
		ContestantId         string    `json:"contestantId"`
		FirstName            string    `json:"firstName"`
		LastName             string    `json:"lastName"`
		ShortLastName        string    `json:"shortLastName"`
		MatchName            string    `json:"matchName"`
		DisplayName          string    `json:"displayName"`
		Position             string    `json:"position"`
		ShirtKey             string    `json:"shirtKey"`
		ContestantFlagKey    string    `json:"contestantFlagKey"`
		ContestantName       string    `json:"contestantName"`
		ContestantShortName  string    `json:"contestantShortName"`
		PercentSelected      float64   `json:"percentSelected"`
		AvailabilityDisplay  string    `json:"availabilityDisplay"`
		SuspensionDetails    any       `json:"suspensionDetails"`
		InjuryDetails        any       `json:"injuryDetails"`
		Price                float64   `json:"price"`
		AveragePoints        float64   `json:"averagePoints"`
		Last3Average         float64   `json:"last3Average"`
		PpmPoints            float64   `json:"ppmPoints"`
		BonusPoints          float64   `json:"bonusPoints"`
		TotalPoints          int       `json:"totalPoints"`
		Goals                int       `json:"goals"`
		Assists              int       `json:"assists"`
		ShotsOnTarget        int       `json:"shotsOnTarget"`
		ChancesCreated       int       `json:"chancesCreated"`
		Tackles              int       `json:"tackles"`
		CleanSheet           int       `json:"cleanSheet"`
		Saves                int       `json:"saves"`
		GoalsConceded        int       `json:"goalsConceded"`
		YellowCards          int       `json:"yellowCards"`
		RedCards             int       `json:"redCards"`
		OwnGoals             int       `json:"ownGoals"`
		PenaltyMisses        int       `json:"penaltyMisses"`
		PenaltySaves         int       `json:"penaltySaves"`
		BonusPpm             float64   `json:"bonusPpm"`
		Dribbles             int       `json:"dribbles"`
		Crosses              int       `json:"crosses"`
		Offsides             int       `json:"offsides"`
		PassCompletionRate   float64   `json:"passCompletionRate"`
		Interceptions        int       `json:"interceptions"`
		Blocks               int       `json:"blocks"`
		FoulsWon             int       `json:"foulsWon"`
		FoulsMade            int       `json:"foulsMade"`
		GoalsOutsideArea     int       `json:"goalsOutsideArea"`
		ErrorsLeadingToGoal  int       `json:"errorsLeadingToGoal"`
		Punches              int       `json:"punches"`
		Claims               int       `json:"claims"`
		KeeperSweeps         int       `json:"keeperSweeps"`
		NextGameweekFixtures []Fixture `json:"nextGameweekFixtures"`
		OptaPersonId         string    `json:"optaPersonId"`
		DreamTeamName        *string   `json:"dreamTeamName"`
		DreamTeamPrice       *float64  `json:"dreamTeamPrice"`
		GameweekPoints       int       `json:"gameweekPoints"`

		// Compatibility / rendering fields used elsewhere in the app
		CostDisp string          `json:"-"`
		Results  []GameWeekMatch `json:"-"`
	}

	// Fixture is a lightweight representation of next gameweek fixtures in the player JSON.
	Fixture struct {
		FixtureId         string  `json:"fixtureId"`
		OpponentId        string  `json:"opponentId"`
		OpponentName      string  `json:"opponentName"`
		OpponentShortName string  `json:"opponentShortName"`
		ContestantFlagKey string  `json:"contestantFlagKey"`
		Status            string  `json:"status"`
		KickoffAt         string  `json:"kickoffAt"`
		Venue             string  `json:"venue"`
		CompetitionName   *string `json:"competitionName"`
		IsHome            bool    `json:"isHome"`
		GameWeek          int     `json:"gameweek"`
	}

	PlayerFilter struct {
		Team        string
		Job         string
		Cost        int
		Points      int
		Average     int
		ApplyFilter bool
	}

	GameWeek struct {
		Success bool         `json:"success"`
		Data    GameWeekData `json:"data"`
	}

	GameWeekData struct {
		Items []GameWeekMatch `json:"items"`
	}

	GameWeekMatch struct {
		MatchId          string         `json:"matchId"`
		CompetitionLabel string         `json:"competitionLabel"`
		KickoffAt        string         `json:"kickoffAt"`
		Venue            string         `json:"venue"`
		Status           string         `json:"status"`
		StatusVariant    string         `json:"statusVariant"`
		MdLabel          string         `json:"mdLabel"`
		MdPoints         string         `json:"mdPoints"`
		PeriodId         string         `json:"periodId"`
		LeftTeam         GameWeekTeam   `json:"leftTeam"`
		RightTeam        GameWeekTeam   `json:"rightTeam"`
		Stats            []GameWeekStat `json:"stats"`
	}

	GameWeekTeam struct {
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		FlagKey   string `json:"flagKey"`
		Score     int    `json:"score"`
	}

	GameWeekStat struct {
		Label  string `json:"label"`
		Total  any    `json:"total"`
		Points string `json:"points"`
	}

	MatchWeek struct {
		Id            int     `json:"id"`
		Status        string  `json:"status"`
		MatchesInWeek []Match `json:"matches"`
	}
)

func (g GameWeekStat) TotalAsInt() (int, error) {
	switch v := g.Total.(type) {
	case nil:
		return 0, nil
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, nil
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("invalid GameWeekStat.Total string %q: %w", trimmed, err)
		}
		return parsed, nil
	case json.Number:
		parsed, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, fmt.Errorf("invalid GameWeekStat.Total number %q: %w", v.String(), err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("GameWeekStat.Total is %T, not int or string", g.Total)
	}
}

type (
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
		Id             string
		Position       string
		Name           string
		Team           string
		Cost           string
		AveragePoints  float64
		Last3Average   float64
		TotalPoints    int
		Goals          int
		Assists        int
		ShotsOnTarget  int
		ChancesCreated int
		Tackles        int
		TeamFixtures   []RenderedMatch
		TeamResults    []RenderedResult
	}

	RenderedMatch struct {
		Gw          int
		Competition string
		Fixture     string
		Venue       string
		KickOff     string
	}

	RenderedResult struct {
		MatchId        string
		Competition    string
		KickOff        string
		HomeTeam       string
		AwayTeam       string
		HomeScore      int
		AwayScore      int
		Venue          string
		MinutesPlayed  int
		ShotsOnTarget  int
		GoalsConceded  int
		GameWeek       string
		MatchDayPoints int
	}
)
