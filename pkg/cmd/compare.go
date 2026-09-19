package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"guysports/playerstats/pkg/types"
)

type Compare struct {
	ModelInput string `arg:"" name:"model" help:"Path to the model-only JSON produced by 'recommend --model-only'"`
	Gameweek   int    `name:"gw" required:"" help:"Gameweek number to compare actual results against"`
	DataDir    string `help:"Directory containing <playerId>-matches.json files" default:"data"`
	Output     string `help:"Path to write the generated comparison HTML page" default:"data/compare.html"`
}

type compareRow struct {
	PlayerId    string
	Name        string
	Team        string
	Position    string
	ModelScore  float64
	ModelRank   int
	ActualScore int
	ActualRank  int
	HasActual   bool
	RankDelta   int
}

func (c *Compare) Run(globals *Globals) error {
	if c.DataDir == "" {
		c.DataDir = "data"
	}
	if c.Output == "" {
		c.Output = "data/compare.html"
	}
	if c.Gameweek <= 0 {
		return fmt.Errorf("--gw must be a positive gameweek number")
	}

	data, err := os.ReadFile(c.ModelInput)
	if err != nil {
		return fmt.Errorf("read model input %s: %w", c.ModelInput, err)
	}
	var entries []recommendationInput
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse model input %s: %w", c.ModelInput, err)
	}

	rows := make([]compareRow, 0, len(entries))
	for i, entry := range entries {
		row := compareRow{PlayerId: entry.PlayerId, Name: entry.Name, Team: entry.Team, Position: entry.Position, ModelScore: entry.Score, ModelRank: i + 1}
		if entry.PlayerId == "" {
			fmt.Printf("warning: %s has no player_id in the model input; cannot look up gameweek %d results\n", entry.Name, c.Gameweek)
			rows = append(rows, row)
			continue
		}
		points, found, err := gameweekPoints(c.DataDir, entry.PlayerId, c.Gameweek)
		if err != nil {
			return fmt.Errorf("read gameweek %d results for %s: %w", c.Gameweek, entry.Name, err)
		}
		row.ActualScore = points
		row.HasActual = found
		rows = append(rows, row)
	}

	rankByActualScore(rows)

	if err := writeCompareHTML(c.Output, c.Gameweek, rows); err != nil {
		return err
	}
	fmt.Printf("Comparison page written to %s\n", c.Output)
	return nil
}

// gameweekPoints sums the fantasy points a player scored across every match
// (league, cup, etc.) tagged with the requested gameweek label.
func gameweekPoints(dataDir, playerId string, gameweek int) (int, bool, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, fmt.Sprintf("%s-matches.json", playerId)))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	var matches types.GameWeek
	if err := json.Unmarshal(data, &matches); err != nil {
		return 0, false, err
	}

	label := fmt.Sprintf("GW %d", gameweek)
	total := 0
	found := false
	for _, item := range matches.Data.Items {
		if item.MdLabel != label {
			continue
		}
		found = true
		points, err := parseMdPoints(item.MdPoints)
		if err != nil {
			return 0, false, fmt.Errorf("parse mdPoints %q for match %s: %w", item.MdPoints, item.MatchId, err)
		}
		total += points
	}
	return total, found, nil
}

func parseMdPoints(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimPrefix(value, "+"))
}

// rankByActualScore assigns ActualRank and RankDelta (model rank minus
// actual rank) to every row that has an actual gameweek score, leaving rows
// without a played match unranked.
func rankByActualScore(rows []compareRow) {
	played := make([]int, 0, len(rows))
	for i, row := range rows {
		if row.HasActual {
			played = append(played, i)
		}
	}
	sort.SliceStable(played, func(a, b int) bool { return rows[played[a]].ActualScore > rows[played[b]].ActualScore })
	for rank, index := range played {
		rows[index].ActualRank = rank + 1
		rows[index].RankDelta = rows[index].ModelRank - rows[index].ActualRank
	}
}

const compareTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<title>Model vs Gameweek {{.Gameweek}}</title>
<style>
body { font-family: system-ui, sans-serif; margin: 2rem; background: #0f1115; color: #e6e6e6; }
h1 { font-size: 1.4rem; }
table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
th, td { padding: 0.5rem 0.75rem; text-align: left; border-bottom: 1px solid #333; }
th { cursor: pointer; position: sticky; top: 0; background: #1a1d24; user-select: none; }
tr:hover { background: #1a1d24; }
.delta-pos { color: #4caf50; }
.delta-neg { color: #f44336; }
.no-data { color: #888; }
</style>
</head>
<body>
<h1>Model score vs gameweek {{.Gameweek}} results</h1>
<p>{{len .Rows}} players compared. Click a column header to sort.</p>
<table id="compare-table">
<thead>
<tr>
<th>Model Rank</th>
<th>Player</th>
<th>Team</th>
<th>Position</th>
<th>Model Score</th>
<th>GW{{.Gameweek}} Score</th>
<th>GW Rank</th>
<th>Rank &Delta;</th>
</tr>
</thead>
<tbody>
{{range .Rows}}
<tr>
<td>{{.ModelRank}}</td>
<td>{{.Name}}</td>
<td>{{.Team}}</td>
<td>{{.Position}}</td>
<td>{{printf "%.1f" .ModelScore}}</td>
{{if .HasActual}}
<td>{{.ActualScore}}</td>
<td>{{.ActualRank}}</td>
<td class="{{if gt .RankDelta 0}}delta-pos{{else if lt .RankDelta 0}}delta-neg{{end}}">{{.RankDelta}}</td>
{{else}}
<td class="no-data">-</td>
<td class="no-data">-</td>
<td class="no-data">-</td>
{{end}}
</tr>
{{end}}
</tbody>
</table>
<script>
const table = document.getElementById('compare-table');
const headers = table.querySelectorAll('th');
const sortState = {};
headers.forEach((th, index) => {
  th.addEventListener('click', () => {
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));
    const asc = sortState[index] = !sortState[index];
    rows.sort((a, b) => {
      const av = a.children[index].textContent.trim();
      const bv = b.children[index].textContent.trim();
      const an = parseFloat(av), bn = parseFloat(bv);
      const cmp = (!isNaN(an) && !isNaN(bn)) ? an - bn : av.localeCompare(bv);
      return asc ? cmp : -cmp;
    });
    rows.forEach(row => tbody.appendChild(row));
  });
});
</script>
</body>
</html>
`

func writeCompareHTML(outputPath string, gameweek int, rows []compareRow) error {
	tmpl, err := template.New("compare").Parse(compareTemplate)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(outputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, struct {
		Gameweek int
		Rows     []compareRow
	}{Gameweek: gameweek, Rows: rows})
}
