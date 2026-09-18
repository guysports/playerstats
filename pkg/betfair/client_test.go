package betfair

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatchOdds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/v1.0/listEvents":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{"eventName": "Arsenal v Bournemouth", "eventId": "evt-1"}})
		case "/rest/v1.0/listMarketCatalogue":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{"marketId": "market-1"}})
		case "/rest/v1.0/listMarketBook":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"runners": []any{
					map[string]any{"ex": map[string]any{"availableToBack": []any{map[string]any{"price": 1.9}}}},
					map[string]any{"ex": map[string]any{"availableToBack": []any{map[string]any{"price": 3.2}}}},
					map[string]any{"ex": map[string]any{"availableToBack": []any{map[string]any{"price": 4.5}}}},
				},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		LoginURL: server.URL,
		API:      nil,
	}

	odds, err := client.MatchOdds("Arsenal", "Bournemouth")
	if err != nil {
		t.Fatalf("MatchOdds() error = %v", err)
	}
	if odds.Home != 1.9 || odds.Draw != 3.2 || odds.Away != 4.5 {
		t.Fatalf("MatchOdds() = %+v, want {Home:1.9 Draw:3.2 Away:4.5}", odds)
	}
}
