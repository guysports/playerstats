package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"guysports/playerstats/pkg/helper"
	"io"
	"os"
	"path/filepath"

	"github.com/jlaffaye/ftp"
)

type ftpClient interface {
	Login(user, password string) error
	ChangeDir(path string) error
	List(path string) ([]*ftp.Entry, error)
	Stor(path string, r io.Reader) error
	Quit() error
}

type Dump struct {
	NoSync       bool                                 `help:"Only download data locally; skip uploading the data directory to FTP" name:"no-sync"`
	GW           int                                  `help:"Current gameweek number; fetches its fixtures to data/gameweek_<gw>.json so they aren't lost from nextGameweekFixtures once the gameweek rolls over" name:"gw"`
	NewFTPClient func(addr string) (ftpClient, error) `kong:"-"`
}

func (d *Dump) ftpClientFactory() func(string) (ftpClient, error) {
	if d != nil && d.NewFTPClient != nil {
		return d.NewFTPClient
	}
	return func(addr string) (ftpClient, error) {
		return ftp.Dial(addr)
	}
}

func (d *Dump) dumpGameweekFixtures(globals *Globals) error {
	fixtures, err := helper.GetJSONWithHeaders(
		fmt.Sprintf(globals.GameweekMatchesSource, d.GW),
		map[string]string{"Authorization": "Bearer " + globals.DTToken},
	)
	if err != nil {
		return err
	}
	fixtures, err = filterGameweekMatches(fixtures, d.GW)
	if err != nil {
		return err
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, fixtures, "", "  "); err != nil {
		return err
	}
	formatted.WriteByte('\n')
	return os.WriteFile(filepath.Join("data", "currentgameweek.json"), formatted.Bytes(), 0644)
}

// filterGameweekMatches keeps only matches for gw because the endpoint can return every gameweek.
func filterGameweekMatches(payload []byte, gw int) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, err
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(root["data"], &data); err != nil {
		return nil, err
	}
	var matches []json.RawMessage
	if err := json.Unmarshal(data["matches"], &matches); err != nil {
		return nil, err
	}
	kept := make([]json.RawMessage, 0, len(matches))
	for _, raw := range matches {
		var m struct {
			Gameweek int `json:"gameweek"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		if m.Gameweek == gw {
			kept = append(kept, raw)
		}
	}
	var err error
	if data["matches"], err = json.Marshal(kept); err != nil {
		return nil, err
	}
	if root["data"], err = json.Marshal(data); err != nil {
		return nil, err
	}
	return json.Marshal(root)
}

// dumpPlayerStats writes the player's stats from PlayerStatsSource to data/<playerID>-player.json.
func (d *Dump) dumpPlayerStats(globals *Globals, playerID string) error {
	stats, err := helper.GetJSON(fmt.Sprintf(globals.PlayerStatsSource, playerID))
	if err != nil {
		return err
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, stats, "", "  "); err != nil {
		return err
	}
	formatted.WriteByte('\n')
	filename := fmt.Sprintf("%s-player.json", playerID)
	return os.WriteFile(filepath.Join("data", filename), formatted.Bytes(), 0644)
}

func (d *Dump) Run(globals *Globals) error {
	players, err := loadPlayers(globals.Source)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("data", "players.json"), append(data, '\n'), 0644); err != nil {
		return err
	}

	if d.GW > 0 {
		if err := d.dumpGameweekFixtures(globals); err != nil {
			return err
		}
	}

	for _, player := range players {
		matches, err := helper.GetJSON(fmt.Sprintf(globals.MatchesSource, player.PlayerId))
		if err != nil {
			return err
		}
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, matches, "", "  "); err != nil {
			return err
		}
		formatted.WriteByte('\n')
		filename := fmt.Sprintf("%s-matches.json", player.PlayerId)
		if err := os.WriteFile(filepath.Join("data", filename), formatted.Bytes(), 0644); err != nil {
			return err
		}
		if err := d.dumpPlayerStats(globals, player.PlayerId); err != nil {
			return err
		}
	}

	matchIDs := map[string]struct{}{}
	for _, player := range players {
		matches, err := os.ReadFile(filepath.Join("data", fmt.Sprintf("%s-matches.json", player.PlayerId)))
		if err != nil {
			return err
		}
		var matchData struct {
			Data struct {
				Items []struct {
					MatchID string `json:"matchId"`
				} `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(matches, &matchData); err != nil {
			return err
		}
		for _, match := range matchData.Data.Items {
			if match.MatchID != "" {
				matchIDs[match.MatchID] = struct{}{}
			}
		}
	}

	for matchID := range matchIDs {
		filename := fmt.Sprintf("%s-result-stats.json", matchID)
		path := filepath.Join("data", filename)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		stats, err := helper.GetJSONWithHeaders(
			fmt.Sprintf(globals.MatchStatsSource, matchID),
			map[string]string{"Authorization": "Bearer " + globals.DTToken},
		)
		if err != nil {
			return err
		}
		var formattedStats bytes.Buffer
		if err := json.Indent(&formattedStats, stats, "", "  "); err != nil {
			return err
		}
		formattedStats.WriteByte('\n')
		if err := os.WriteFile(path, formattedStats.Bytes(), 0644); err != nil {
			return err
		}
	}

	if d.NoSync {
		return nil
	}
	return d.syncDataDirectory(globals)
}

func (d *Dump) syncDataDirectory(globals *Globals) error {
	if globals == nil || globals.FtpPassword == "" {
		return nil
	}

	ftpConn, err := d.ftpClientFactory()("ftp.guysports.co.uk:21")
	if err != nil {
		return err
	}
	defer func() {
		if err := ftpConn.Quit(); err != nil {
			fmt.Printf("error closing FTP connection %v\n", err)
		}
	}()

	if err := ftpConn.Login("guysports@guysports.co.uk", globals.FtpPassword); err != nil {
		return err
	}
	if err := ftpConn.ChangeDir(globals.DataFTPDirectory); err != nil {
		return err
	}

	files, err := os.ReadDir("data")
	if err != nil {
		return err
	}
	remoteEntries, err := ftpConn.List(".")
	if err != nil {
		return err
	}
	remoteSizes := make(map[string]uint64, len(remoteEntries))
	for _, entry := range remoteEntries {
		if entry.Type == ftp.EntryTypeFile {
			remoteSizes[entry.Name] = entry.Size
		}
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}
		localPath := filepath.Join("data", entry.Name())
		localInfo, err := entry.Info()
		if err != nil {
			return err
		}
		remoteSize, existsRemotely := remoteSizes[entry.Name()]
		if existsRemotely && uint64(localInfo.Size()) == remoteSize {
			fmt.Printf("skipped data/%s (unchanged)\n", entry.Name())
			continue
		}

		localData, err := os.ReadFile(localPath)
		if err != nil {
			return err
		}
		if err := ftpConn.Stor(entry.Name(), bytes.NewReader(localData)); err != nil {
			return err
		}
		fmt.Printf("uploaded data/%s to %s\n", entry.Name(), globals.DataFTPDirectory)
	}
	return nil
}
