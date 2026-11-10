package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDreamTeamTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "token")
	t.Setenv("DT_TOKEN_FILE", path)
	if err := saveDreamTeamToken(path, "test-token"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("token permissions = %o, want 600", info.Mode().Perm())
	}
	token, err := LoadDreamTeamToken()
	if err != nil || token != "test-token" {
		t.Fatalf("load token = %q, %v", token, err)
	}
	if err := saveDreamTeamToken(path, "invalid token"); err == nil {
		t.Fatal("expected invalid token to be rejected")
	}
	token, err = LoadDreamTeamToken()
	if err != nil || token != "test-token" {
		t.Fatalf("invalid write changed token: %q, %v", token, err)
	}
	if err := saveDreamTeamToken(path, "replacement"); err != nil {
		t.Fatal(err)
	}
	token, err = LoadDreamTeamToken()
	if err != nil || token != "replacement" {
		t.Fatalf("replacement token = %q, %v", token, err)
	}
}

func TestLoadDreamTeamTokenErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	t.Setenv("DT_TOKEN_FILE", path)
	if _, err := LoadDreamTeamToken(); err == nil {
		t.Fatal("expected missing file error")
	}
	for _, contents := range []string{"", "a b", "a\nb"} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDreamTeamToken(); err == nil {
			t.Fatalf("expected invalid token error for %q", contents)
		}
	}
}

func TestVerifyDreamTeamToken(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		ok     bool
	}{
		{"valid", http.StatusOK, `{"data":{"userId":"user"}}`, true},
		{"unauthorized", http.StatusUnauthorized, `{"error":"bad token"}`, false},
		{"missing user", http.StatusOK, `{"data":{}}`, false},
		{"invalid JSON", http.StatusOK, `{`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected request: %s %s", r.Method, r.Header.Get("Content-Type"))
				}
				var body struct {
					Token string `json:"token"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token != "secret-token" {
					t.Errorf("unexpected verification request")
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			err := verifyDreamTeamToken(context.Background(), server.Client(), server.URL, "secret-token")
			if (err == nil) != tt.ok {
				t.Fatalf("verification error = %v, want success %t", err, tt.ok)
			}
			if err != nil && strings.Contains(err.Error(), "secret-token") {
				t.Fatal("error disclosed token")
			}
		})
	}
}

func TestLoginRequiresCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DT_USERNAME", "")
	t.Setenv("DT_PASSWORD", "")
	if err := (&Login{}).Run(); err == nil || !strings.Contains(err.Error(), "DT_USERNAME") {
		t.Fatalf("expected credentials error, got %v", err)
	}
}

func TestDreamTeamCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DT_USERNAME", "env-user")
	t.Setenv("DT_PASSWORD", "env-password")
	path := filepath.Join(home, ".guysports", "guysports.json")

	user, password, err := dreamTeamCredentials()
	if err != nil || user != "env-user" || password != "env-password" {
		t.Fatalf("environment fallback failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("file-password"))
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"file overrides environment", `{"dt_username":"file-user","dt_password":"` + encoded + `","other":"ignored"}`, ""},
		{"malformed JSON", `{`, "parse Dream Team credentials file"},
		{"missing password", `{"dt_username":"file-user"}`, "must contain"},
		{"empty username", `{"dt_username":"","dt_password":"` + encoded + `"}`, "must contain"},
		{"invalid base64", `{"dt_username":"file-user","dt_password":"%%%"}`, "decode dt_password"},
		{"empty decoded password", `{"dt_username":"file-user","dt_password":"\n"}`, "must not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			user, password, err := dreamTeamCredentials()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil || user != "file-user" || password != "file-password" {
				t.Fatalf("file credentials not loaded correctly: %v", err)
			}
		})
	}
}

func TestDreamTeamCredentialsReadError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DT_USERNAME", "env-user")
	t.Setenv("DT_PASSWORD", "env-password")
	if err := os.MkdirAll(filepath.Join(home, ".guysports", "guysports.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := dreamTeamCredentials(); err == nil || !strings.Contains(err.Error(), "read Dream Team credentials file") {
		t.Fatalf("expected file read error, got %v", err)
	}
}
