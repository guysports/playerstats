package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const dreamTeamLoginURL = "https://login.thesun.co.uk/oidc/rp/login/dreamteam?gotoUrl=https%3A%2F%2Ffantasy.dreamteamfc.com%2F"
const dreamTeamVerifyURL = "https://engagecraft-fantasy-backend-prod.azurewebsites.net/api/auth/login"

type Login struct {
	Timeout time.Duration `help:"Time allowed for login and any verification in the browser" default:"3m"`
}

func tokenPath() (string, error) {
	if path := os.Getenv("DT_TOKEN_FILE"); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve token directory: %w", err)
	}
	return filepath.Join(dir, "playerstats", "dreamteam-token"), nil
}

func LoadDreamTeamToken() (string, error) {
	path, err := tokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read Dream Team token: %w; set DT_TOKEN or run playerstats login", err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", fmt.Errorf("Dream Team token file is empty or invalid; run playerstats login")
	}
	return token, nil
}

func saveDreamTeamToken(path, token string) error {
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return fmt.Errorf("invalid Dream Team token")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".dreamteam-token-*")
	if err != nil {
		return fmt.Errorf("create token file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(token + "\n"); err != nil {
		f.Close()
		return fmt.Errorf("write token: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close token file: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func verifyDreamTeamToken(ctx context.Context, client *http.Client, url, token string) error {
	body, err := json.Marshal(struct {
		Token string `json:"token"`
	}{token})
	if err != nil {
		return fmt.Errorf("encode token verification: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create token verification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verify Dream Team session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Dream Team session verification failed (HTTP %d)", resp.StatusCode)
	}
	var result struct {
		Data struct {
			UserID string `json:"userId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("decode Dream Team session verification: %w", err)
	}
	if strings.TrimSpace(result.Data.UserID) == "" {
		return fmt.Errorf("Dream Team session verification returned no user ID")
	}
	return nil
}

func dreamTeamCredentials() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve home directory for Dream Team credentials: %w", err)
	}
	path := filepath.Join(home, ".guysports", "guysports.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		username, password := os.Getenv("DT_USERNAME"), os.Getenv("DT_PASSWORD")
		if username == "" || password == "" {
			return "", "", fmt.Errorf("DT_USERNAME and DT_PASSWORD are required when %s is absent", path)
		}
		return username, password, nil
	}
	if err != nil {
		return "", "", fmt.Errorf("read Dream Team credentials file: %w", err)
	}
	var credentials struct {
		Username string `json:"dt_username"`
		Password string `json:"dt_password"`
	}
	if err := json.Unmarshal(data, &credentials); err != nil {
		return "", "", fmt.Errorf("parse Dream Team credentials file: %w", err)
	}
	if credentials.Username == "" || credentials.Password == "" {
		return "", "", fmt.Errorf("Dream Team credentials file must contain dt_username and dt_password")
	}
	password, err := base64.StdEncoding.DecodeString(credentials.Password)
	if err != nil {
		return "", "", fmt.Errorf("decode dt_password in Dream Team credentials file: %w", err)
	}
	if len(password) == 0 {
		return "", "", fmt.Errorf("dt_password in Dream Team credentials file must not be empty")
	}
	return credentials.Username, string(password), nil
}

func (l *Login) Run() error {
	username, password, err := dreamTeamCredentials()
	if err != nil {
		return err
	}
	if l.Timeout <= 0 {
		return fmt.Errorf("login timeout must be positive")
	}
	path, err := tokenPath()
	if err != nil {
		return err
	}

	allocator, cancelAllocator := chromedp.NewExecAllocator(context.Background(),
		chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck)
	defer cancelAllocator()
	browser, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browser, l.Timeout)
	defer cancel()
	if err := chromedp.Run(ctx, chromedp.Navigate(dreamTeamLoginURL)); err != nil {
		return fmt.Errorf("open Dream Team login: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Waiting for Dream Team sign-in. Complete any verification in the browser.")

	submitted := false
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		var state struct {
			Host    string `json:"host"`
			HasForm bool   `json:"hasForm"`
			Token   string `json:"token"`
		}
		err := chromedp.Run(ctx, chromedp.Evaluate(`(() => {
			const visible = selector => {
				const el = document.querySelector(selector);
				return !!el && !el.disabled && el.getClientRects().length > 0;
			};
			return {
				host: location.hostname,
				hasForm: visible('input[type="email"]') && visible('input[type="password"]') && visible('button[type="submit"]'),
				token: location.hostname === "fantasy.dreamteamfc.com" ? (window.ghComponent?.getLoggedCookie?.()?.access_token || "") : ""
			};
		})()`, &state))
		if err != nil {
			return fmt.Errorf("inspect Dream Team login: %w", err)
		}
		if state.Host == "account.thesun.co.uk" && state.HasForm && !submitted {
			if err := chromedp.Run(ctx,
				chromedp.SendKeys(`input[type="email"]`, username, chromedp.ByQuery),
				chromedp.SendKeys(`input[type="password"]`, password, chromedp.ByQuery),
				chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
			); err != nil {
				return fmt.Errorf("submit Dream Team login: %w", err)
			}
			submitted = true
		}
		if state.Token != "" {
			client := &http.Client{
				Timeout: 20 * time.Second,
				CheckRedirect: func(*http.Request, []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			if err := verifyDreamTeamToken(ctx, client, dreamTeamVerifyURL, state.Token); err != nil {
				return err
			}
			if err := saveDreamTeamToken(path, state.Token); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Dream Team token saved.")
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("Dream Team login did not complete: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
