package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"guysports/playerstats/pkg/types"

	"github.com/jlaffaye/ftp"
)

type fakeFTPClient struct {
	loginUser     string
	loginPassword string
	cwd           string
	stored        map[string]string
	remoteFiles   map[string]uint64
}

func (f *fakeFTPClient) Login(user, password string) error {
	f.loginUser = user
	f.loginPassword = password
	return nil
}

func (f *fakeFTPClient) ChangeDir(path string) error {
	f.cwd = path
	return nil
}

func (f *fakeFTPClient) List(path string) ([]*ftp.Entry, error) {
	entries := make([]*ftp.Entry, 0, len(f.remoteFiles))
	for name, size := range f.remoteFiles {
		entries = append(entries, &ftp.Entry{Name: name, Size: size, Type: ftp.EntryTypeFile})
	}
	return entries, nil
}

func (f *fakeFTPClient) Stor(path string, r io.Reader) error {
	if f.stored == nil {
		f.stored = map[string]string{}
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.stored[path] = string(data)
	return nil
}

func (f *fakeFTPClient) Quit() error { return nil }

func TestDumpRun(t *testing.T) {
	t.Chdir(t.TempDir())
	fixtureDir := t.TempDir()
	source := filepath.Join(fixtureDir, "players.json")
	input := []types.Player{
		{PlayerId: "player-1", FirstName: "Ada", LastName: "Lovelace"},
		{PlayerId: "player-2", FirstName: "Grace", LastName: "Hopper"},
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	if err := os.WriteFile(source, data, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	for _, player := range input {
		matches := fmt.Appendf(nil, `{"data":{"items":[{"matchId":"%s-match"}]}}`, player.PlayerId)
		if err := os.WriteFile(filepath.Join(fixtureDir, player.PlayerId+".json"), matches, 0644); err != nil {
			t.Fatalf("os.WriteFile() returned an error: %v", err)
		}
	}
	if err := os.MkdirAll("data", 0755); err != nil {
		t.Fatalf("os.MkdirAll() returned an error: %v", err)
	}
	existingStats := []byte(`{"existing":true}`)
	if err := os.WriteFile(filepath.Join("data", "player-1-match-result-stats.json"), existingStats, 0644); err != nil {
		t.Fatalf("os.WriteFile() returned an error: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer test-token")
		}
		if r.URL.Path != "/matches/player-2-match/stats" {
			t.Errorf("request path = %q, want %q", r.URL.Path, "/matches/player-2-match/stats")
		}
		_, _ = fmt.Fprint(w, `{"stats":[{"label":"Minutes played","total":90}]}`)
	}))
	defer server.Close()

	fakeFTP := &fakeFTPClient{remoteFiles: map[string]uint64{}, stored: map[string]string{}}

	if err := (&Dump{NewFTPClient: func(addr string) (ftpClient, error) {
		if addr != "ftp.guysports.co.uk:21" {
			return nil, fmt.Errorf("unexpected ftp address: %s", addr)
		}
		return fakeFTP, nil
	}}).Run(&Globals{
		Source:           source,
		MatchesSource:    filepath.Join(fixtureDir, "%s.json"),
		MatchStatsSource: server.URL + "/matches/%s/stats",
		DTToken:          "test-token",
		FtpPassword:      "secret",
		DataFTPDirectory: "/subdomains/playerinsights/data",
	}); err != nil {
		t.Fatalf("Dump.Run() returned an error: %v", err)
	}

	if fakeFTP.loginUser != "guysports@guysports.co.uk" {
		t.Fatalf("FTP login user = %q, want %q", fakeFTP.loginUser, "guysports@guysports.co.uk")
	}
	if fakeFTP.loginPassword != "secret" {
		t.Fatalf("FTP login password = %q, want %q", fakeFTP.loginPassword, "secret")
	}
	if fakeFTP.cwd != "/subdomains/playerinsights/data" {
		t.Fatalf("FTP cwd = %q, want %q", fakeFTP.cwd, "/subdomains/playerinsights/data")
	}
	if _, ok := fakeFTP.stored["player-2-match-result-stats.json"]; !ok {
		t.Fatalf("uploaded stats were not sent to FTP: %+v", fakeFTP.stored)
	}

	output, err := os.ReadFile(filepath.Join("data", "players.json"))
	if err != nil {
		t.Fatalf("os.ReadFile() returned an error: %v", err)
	}
	var got []types.Player
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("dump output is invalid JSON: %v", err)
	}
	if len(got) != len(input) || got[0].PlayerId != "player-1" {
		t.Fatalf("dump output = %+v, want the input players", got)
	}

	for _, player := range input {
		matches, err := os.ReadFile(filepath.Join("data", player.PlayerId+"-matches.json"))
		if err != nil {
			t.Fatalf("os.ReadFile() returned an error for %s: %v", player.PlayerId, err)
		}
		want := fmt.Sprintf("{\n  \"data\": {\n    \"items\": [\n      {\n        \"matchId\": \"%s-match\"\n      }\n    ]\n  }\n}\n", player.PlayerId)
		if string(matches) != want {
			t.Fatalf("matches output for %s = %s", player.PlayerId, matches)
		}
	}

	if got, err := os.ReadFile(filepath.Join("data", "player-1-match-result-stats.json")); err != nil {
		t.Fatalf("os.ReadFile() returned an error for existing stats: %v", err)
	} else if string(got) != string(existingStats) {
		t.Fatalf("existing stats were changed to %s", got)
	}
	statsOutput, err := os.ReadFile(filepath.Join("data", "player-2-match-result-stats.json"))
	if err != nil {
		t.Fatalf("os.ReadFile() returned an error for downloaded stats: %v", err)
	}
	wantStats := "{\n  \"stats\": [\n    {\n      \"label\": \"Minutes played\",\n      \"total\": 90\n    }\n  ]\n}\n"
	if string(statsOutput) != wantStats {
		t.Fatalf("downloaded stats = %s, want formatted JSON", statsOutput)
	}
}
