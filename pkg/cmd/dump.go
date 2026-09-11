package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"guysports/playerstats/pkg/helper"
	"os"
	"path/filepath"

	"github.com/jlaffaye/ftp"
)

type Dump struct{}

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

	return syncDataDirectory(globals)
}

func syncDataDirectory(globals *Globals) error {
	ftpClient, err := ftp.Dial("ftp.guysports.co.uk:21")
	if err != nil {
		return err
	}
	defer ftpClient.Quit()

	if err := ftpClient.Login("guysports@guysports.co.uk", globals.FtpPassword); err != nil {
		return err
	}
	if err := ftpClient.ChangeDir(globals.DataFTPDirectory); err != nil {
		return err
	}

	files, err := os.ReadDir("data")
	if err != nil {
		return err
	}
	remoteEntries, err := ftpClient.List(".")
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
		if err := ftpClient.Stor(entry.Name(), bytes.NewReader(localData)); err != nil {
			return err
		}
		fmt.Printf("uploaded data/%s to %s\n", entry.Name(), globals.DataFTPDirectory)
	}
	return nil
}
