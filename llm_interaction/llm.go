package llm_interaction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var llmURL = "https://opencode.ai/zen/go/v1/responses"

func init() { _ = godotenv.Load() }

func callLLM(prompt string) (string, error) {
	key := os.Getenv("OPENCODE_API_KEY")
	if key == "" {
		key = os.Getenv("OPENCODE_API_TOKEN")
	}
	if key == "" {
		key = os.Getenv("OPENCODE_TOKEN")
	}
	if key == "" {
		return "", fmt.Errorf("OPENCODE_API_KEY not set in .env")
	}
	if v := os.Getenv("OPENCODE_API_URL"); v != "" {
		llmURL = v
	}
	body, _ := json.Marshal(map[string]any{
		"model":     "muse-spark-1.2-contributor",
		"input":     prompt,
		"reasoning": map[string]string{"effort": "minimal"},
	})
	req, _ := http.NewRequest("POST", llmURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("llm %d: %s", resp.StatusCode, string(data))
	}
	var out struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("json %v raw %s", err, string(data))
	}
	if out.Status == "incomplete" {
		return "", fmt.Errorf("incomplete %s", string(data))
	}
	var sb strings.Builder
	for _, o := range out.Output {
		if o.Type == "message" {
			for _, c := range o.Content {
				if c.Type == "output_text" {
					sb.WriteString(c.Text)
				}
			}
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("empty output %s", string(data))
	}
	return sb.String(), nil
}

// extractJSON finds first [ or { and last matching ] or }
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// strip ```json ... ```
	if strings.Contains(s, "```") {
		a := strings.Index(s, "```")
		b := strings.LastIndex(s, "```")
		if a != b {
			inner := s[a+3 : b]
			inner = strings.TrimPrefix(inner, "json")
			s = strings.TrimSpace(inner)
		}
	}
	start := strings.Index(s, "[")
	if start == -1 {
		start = strings.Index(s, "{")
	}
	end := strings.LastIndex(s, "]")
	if end == -1 {
		end = strings.LastIndex(s, "}")
	}
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}
