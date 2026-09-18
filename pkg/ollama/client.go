package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

type chatRequest struct {
	Model    string                 `json:"model"`
	Messages []message              `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message message `json:"message"`
}

func NewClient(baseURL, model string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// contextSizeFor picks a num_ctx large enough to hold the prompt plus the
// system message and response, since Ollama otherwise silently truncates
// large payloads to its default (small) context window.
func contextSizeFor(promptLen int) int {
	const minContext = 4096
	// rough estimate of ~4 chars per token, plus headroom for the system
	// prompt and the model's response.
	estimated := promptLen/4 + 2048
	size := minContext
	for size < estimated {
		size *= 2
	}
	return size
}

func (c *Client) Explain(prompt string) (string, error) {
	requestBody, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: "You are a football recommendation explainer. Use only the supplied JSON. Preserve the supplied ranking and score. Mention every supplied player exactly once, in order. For each player, state every upcoming fixture opponent, home or away status, gameweek, and the supplied difficulty classification. Explain that favorable/even/difficult is a relative squad-strength proxy, not an official rating. Do not invent fixture results, difficulty, injuries, or statistics. If difficulty is unknown, say so. Return a numbered list and nothing else."},
			{Role: "user", Content: "Here is the player JSON data:\n\n" + prompt + "\n\nNow write the numbered list of recommendations for the players above, one entry per player, following the system instructions exactly. Do not describe the JSON schema or structure - only summarize the players and their fixtures."},
		},
		Stream: false,
		Options: map[string]interface{}{
			"num_ctx":     contextSizeFor(len(prompt)),
			"temperature": 0.2,
		},
	})
	if err != nil {
		return "", err
	}

	response, err := c.HTTP.Post(c.BaseURL+"/api/chat", "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("ollama returned HTTP %s", response.Status)
	}

	var payload chatResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Message.Content, nil
}
