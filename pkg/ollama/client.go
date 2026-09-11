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
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
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

func (c *Client) Explain(prompt string) (string, error) {
	requestBody, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: "You are a football recommendation explainer. Use only the supplied JSON. Preserve the supplied ranking and score. Mention every supplied player exactly once, in order. For each player, state every upcoming fixture opponent, home or away status, gameweek, and the supplied difficulty classification. Explain that favorable/even/difficult is a relative squad-strength proxy, not an official rating. Do not invent fixture results, difficulty, injuries, or statistics. If difficulty is unknown, say so. Return a numbered list and nothing else."},
			{Role: "user", Content: prompt},
		},
		Stream: false,
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
