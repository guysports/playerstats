package helper

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"guysports/playerstats/pkg/types"

	"golang.org/x/net/html"
)

// ParsePremierLeagueTable converts the BBC Premier League HTML table into
// stable typed rows for the recommendation engine.
func ParsePremierLeagueTable(reader io.Reader) ([]types.LeagueTableEntry, error) {
	document, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	var rows []types.LeagueTableEntry
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			if entry, ok := parseTableRow(node); ok {
				rows = append(rows, entry)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	if len(rows) == 0 {
		return nil, fmt.Errorf("premier league table not found")
	}
	for i := range rows {
		if rows[i].Position == 0 {
			rows[i].Position = i + 1
		}
	}
	return rows, nil
}

func parseTableRow(row *html.Node) (types.LeagueTableEntry, bool) {
	cells := make([]*html.Node, 0, 12)
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
			cells = append(cells, child)
		}
	}
	if len(cells) < 9 {
		return types.LeagueTableEntry{}, false
	}

	teamIndex := 0
	valueStart := 1
	position := 0
	if positionCandidate, err := parseTableInt(nodeText(cells[0])); err == nil && positionCandidate >= 1 && positionCandidate <= 20 {
		position = positionCandidate
		teamIndex = 1
		valueStart = 2
	}
	if teamIndex >= len(cells) {
		return types.LeagueTableEntry{}, false
	}
	team := nodeText(cells[teamIndex])
	if team == "" {
		return types.LeagueTableEntry{}, false
	}

	statCells := cells[valueStart:]
	if len(statCells) < 8 {
		return types.LeagueTableEntry{}, false
	}
	values := make([]int, 0, 8)
	for _, cell := range statCells[:8] {
		value, err := parseTableInt(nodeText(cell))
		if err != nil {
			return types.LeagueTableEntry{}, false
		}
		values = append(values, value)
	}

	entry := types.LeagueTableEntry{
		Position: position,
		Team:     team,
		Played:   values[0], Won: values[1], Drawn: values[2], Lost: values[3],
		GoalsFor: values[4], GoalsAgainst: values[5], GoalDifference: values[6], Points: values[7],
	}
	if len(statCells) > 8 {
		entry.RecentForm, entry.FormPoints, entry.FormRate = parseForm(nodeText(statCells[8]))
	}
	return entry, true
}

func parseForm(value string) (string, int, float64) {
	results := make([]string, 0, 6)
	for _, token := range strings.Fields(strings.ToUpper(value)) {
		token = strings.Trim(token, "[](),|/")
		if token == "W" || token == "D" || token == "L" {
			results = append(results, token)
		}
	}
	points := 0
	for _, result := range results {
		switch result {
		case "W":
			points += 3
		case "D":
			points++
		}
	}
	rate := 0.0
	if len(results) > 0 {
		rate = float64(points) / float64(len(results)*3)
	}
	return strings.Join(results, ""), points, rate
}

func parseTableInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty table value")
	}
	if value == "-" {
		return 0, nil
	}
	for index, character := range value {
		if (character == '-' && index == 0) || (character >= '0' && character <= '9') {
			continue
		}
		return 0, fmt.Errorf("invalid table value %q", value)
	}
	return strconv.Atoi(value)
}

func nodeText(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}
	parts := make([]string, 0, 2)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		parts = append(parts, nodeText(child))
	}
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}
