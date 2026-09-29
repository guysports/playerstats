package betfair

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/guysports/go-betfair-api/pkg/access"
	"github.com/guysports/go-betfair-api/pkg/betting"
	"github.com/guysports/go-betfair-api/pkg/transport"
	betfairtypes "github.com/guysports/go-betfair-api/pkg/types"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultBaseURL  = "https://api.betfair.com/exchange/betting"
	defaultLoginURL = "https://identitysso.betfair.com/api/login"
)

type LoginConfig struct {
	Username string `json:"user"`
	Password string `json:"password"`
	AppKey   string `json:"appKey"`
	CertPath string `json:"certpath"`
	KeyPath  string `json:"keypath"`
}

type Odds struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

type Client struct {
	BaseURL  string
	LoginURL string
	API      *betting.API

	loginPath string
	appKey    string

	oddsCacheMu sync.Mutex
	oddsCache   map[string]oddsCacheEntry
}

type oddsCacheEntry struct {
	odds Odds
	err  error
}

// oddsCacheKey normalizes team names so players from the same fixture (e.g.
// teammates) share a single cached lookup rather than each making their own
// Betfair request.
func oddsCacheKey(homeTeam, awayTeam string) string {
	return strings.ToLower(strings.TrimSpace(homeTeam)) + "|" + strings.ToLower(strings.TrimSpace(awayTeam))
}

func NewClient(loginPath string, appKey string) (*Client, error) {
	apiClient, err := access.NewLogin(loginPath)
	if err != nil {
		return nil, err
	}

	// NB: the go-betfair-api client stores this context on the transport and
	// reuses it for every subsequent request (not just login), so it must not
	// carry a deadline/cancel that fires once NewClient returns - doing so
	// caused every later API call to fail with "context canceled".
	ctx := context.Background()

	// Login to Betfair
	bettingClient, err := apiClient.BetfairAuthenticate(ctx, appKey, nil)
	if err != nil {
		return nil, err
	}
	client := &Client{
		BaseURL:   defaultBaseURL,
		LoginURL:  defaultLoginURL,
		API:       bettingClient,
		loginPath: loginPath,
		appKey:    appKey,
	}
	enableDebugLogging(client)

	return client, nil
}

// isSessionExpiredErr reports whether err is Betfair's ANGX-0003 gateway
// error, which the go-betfair-api access package can return even for a
// cached session it still considers valid (its expiry check has an off-by-
// one-window bug that lets it reuse an already-expired session token).
func isSessionExpiredErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ANGX-0003")
}

// reauthenticate discards the cached session token and logs in again.
func (c *Client) reauthenticate() error {
	sessionHome := os.Getenv("UNIT_TEST_HOME")
	if sessionHome == "" {
		sessionHome = filepath.Join(os.Getenv("HOME"), ".betfair")
	}
	_ = os.Remove(filepath.Join(sessionHome, "session.json"))

	login, err := access.NewLogin(c.loginPath)
	if err != nil {
		return err
	}
	api, err := login.BetfairAuthenticate(context.Background(), c.appKey, nil)
	if err != nil {
		return err
	}
	c.API = api
	enableDebugLogging(c)
	return nil
}

// withSessionRetry runs fn, and if it fails with a session-expired error,
// re-authenticates and retries fn once.
func (c *Client) withSessionRetry(fn func() error) error {
	err := fn()
	if isSessionExpiredErr(err) {
		if reauthErr := c.reauthenticate(); reauthErr == nil {
			err = fn()
		}
	}
	return err
}

// enableDebugLogging wires up request/response logging on the underlying
// retryablehttp client when BETFAIR_DEBUG is set, printing headers and
// payloads for every Betfair API call.
func enableDebugLogging(client *Client) {
	if os.Getenv("BETFAIR_DEBUG") == "" {
		return
	}
	jsonRPCClient, ok := client.API.Client.(*transport.JsonRPCClient)
	if !ok || jsonRPCClient.Client == nil {
		return
	}
	jsonRPCClient.Client.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, attempt int) {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			fmt.Printf("betfair debug: failed to dump request (attempt %d): %v\n", attempt, err)
			return
		}
		fmt.Printf("betfair debug: request (attempt %d):\n%s\n", attempt, dump)
	}
	jsonRPCClient.Client.ResponseLogHook = func(_ retryablehttp.Logger, resp *http.Response) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("betfair debug: failed to read response body: %v\n", err)
			return
		}
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))
		fmt.Printf("betfair debug: response %s\nheaders: %v\nbody: %s\n", resp.Status, resp.Header, body)
	}
}

func (c *Client) MatchOdds(homeTeam, awayTeam string) (Odds, error) {
	if c == nil {
		return Odds{}, fmt.Errorf("betfair client is nil")
	}
	key := oddsCacheKey(homeTeam, awayTeam)
	c.oddsCacheMu.Lock()
	if entry, ok := c.oddsCache[key]; ok {
		c.oddsCacheMu.Unlock()
		return entry.odds, entry.err
	}
	c.oddsCacheMu.Unlock()

	odds, err := c.fetchMatchOdds(homeTeam, awayTeam)

	c.oddsCacheMu.Lock()
	if c.oddsCache == nil {
		c.oddsCache = make(map[string]oddsCacheEntry)
	}
	c.oddsCache[key] = oddsCacheEntry{odds: odds, err: err}
	c.oddsCacheMu.Unlock()

	return odds, err
}

func (c *Client) fetchMatchOdds(homeTeam, awayTeam string) (Odds, error) {
	filter := &betfairtypes.MarketFilter{
		TextQuery:       strings.TrimSpace(homeTeam + " " + awayTeam),
		MarketTypeCodes: []string{"MATCH_ODDS"},
	}
	var events []betfairtypes.EventWrapper
	err := c.withSessionRetry(func() error {
		var err error
		events, err = c.API.ListEvents(filter)
		return err
	})
	if err != nil {
		return Odds{}, fmt.Errorf("query betfair events: %w", err)
	}
	if len(events) == 0 {
		return Odds{}, fmt.Errorf("betfair found no matching event for %s vs %s", homeTeam, awayTeam)
	}

	var eventID string
	for _, event := range events {
		if event.Event == nil {
			continue
		}
		name := event.Event.Name
		if strings.EqualFold(name, homeTeam) || strings.Contains(strings.ToLower(name), strings.ToLower(homeTeam)) || strings.Contains(strings.ToLower(name), strings.ToLower(awayTeam)) {
			eventID = event.Event.ID
			break
		}
	}
	if eventID == "" {
		return Odds{}, fmt.Errorf("betfair did not resolve event id for %s vs %s", homeTeam, awayTeam)
	}

	var catalogues []betfairtypes.MarketCatalogueWrapper
	err = c.withSessionRetry(func() error {
		var err error
		catalogues, err = c.API.ListMarketCatalogue(&betfairtypes.MarketFilter{EventIds: []string{eventID}, MarketTypeCodes: []string{"MATCH_ODDS"}}, 20, []string{"RUNNER_DESCRIPTION"})
		return err
	})
	if err != nil {
		return Odds{}, fmt.Errorf("query betfair markets: %w", err)
	}
	if len(catalogues) == 0 {
		return Odds{}, fmt.Errorf("betfair returned no market catalogue for %s vs %s", homeTeam, awayTeam)
	}
	prices := make([]float64, 0, 3)
	marketIDs := make([]string, 0, len(catalogues))
	for _, catalogue := range catalogues {
		marketIDs = append(marketIDs, catalogue.MarketId)
	}
	var books []betfairtypes.MarketBookWrapper
	err = c.withSessionRetry(func() error {
		var err error
		books, err = c.API.ListMarketBook(marketIDs, &betfairtypes.PriceProjection{PriceData: []string{"EX_ALL_OFFERS"}}, "", "")
		return err
	})
	if err != nil {
		return Odds{}, fmt.Errorf("query betfair odds: %w", err)
	}
	if len(books) == 0 {
		return Odds{}, fmt.Errorf("betfair market book empty for %s vs %s", homeTeam, awayTeam)
	}
	for _, book := range books {
		for _, runner := range book.Runners {
			if len(runner.Exchange.AvailableToBack) == 0 {
				continue
			}
			prices = append(prices, float64(runner.Exchange.AvailableToBack[0].Price))
		}
	}
	if len(prices) == 0 {
		return Odds{}, fmt.Errorf("betfair odds unavailable for %s vs %s", homeTeam, awayTeam)
	}
	sort.Float64s(prices)
	if len(prices) >= 3 {
		return Odds{Home: prices[0], Draw: prices[1], Away: prices[2]}, nil
	}
	return Odds{Home: prices[0], Draw: prices[0], Away: prices[0]}, nil
}
