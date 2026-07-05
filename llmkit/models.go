package llmkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const gatewayModelLimit = 120

var gatewayModelPriority = []string{
	VercelGatewayDefaultModel,
	"anthropic/claude-opus-4.8",
	"openai/gpt-5.5",
	"google/gemini-3.5-flash",
	"google/gemini-3.1-flash-lite",
	"xai/grok-4.3",
	"alibaba/qwen3-coder",
}

type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ModelLister struct {
	Client *http.Client
	Env    EnvFunc
}

func (l ModelLister) List(ctx context.Context, cfg Config) ([]Model, error) {
	resolved := ResolveWithEnv(cfg, l.env())
	if resolved.IsNative {
		return NativeProviderModels(resolved.Provider), nil
	}
	if resolved.BaseURL == "" {
		return nil, fmt.Errorf("LLM provider not configured")
	}
	if !IsAllowedBaseURL(resolved.BaseURL) {
		return nil, fmt.Errorf("invalid LLM base URL")
	}
	if resolved.NeedsAPIKey && !resolved.HasAPIKey() && !resolved.IsGateway {
		return nil, fmt.Errorf("API key required to fetch models from remote provider")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolved.ModelsURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}
	if !resolved.IsGateway {
		resolved.Authorize(req)
	}

	resp, err := l.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach provider: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read provider response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ProviderError("provider", resp.StatusCode, body)
	}
	return NormalizeModels(body, resolved.IsGateway)
}

func (l ModelLister) client() *http.Client {
	if l.Client != nil {
		return l.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (l ModelLister) env() EnvFunc {
	if l.Env != nil {
		return l.Env
	}
	return nil
}

func NormalizeModels(body []byte, gateway bool) ([]Model, error) {
	var modelsResp struct {
		Data []struct {
			ID     string   `json:"id"`
			Name   string   `json:"name"`
			Type   string   `json:"type"`
			Tags   []string `json:"tags"`
			Object string   `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("parse models response: %w", err)
	}

	models := make([]Model, 0, len(modelsResp.Data))
	seen := make(map[string]bool, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || seen[id] {
			continue
		}
		if gateway && !isUsefulGatewayChatModel(m.Type, m.Tags) {
			continue
		}
		name := strings.TrimSpace(m.Name)
		if name == "" {
			name = id
		}
		models = append(models, Model{ID: id, Name: name})
		seen[id] = true
	}

	sortModels(models, gateway)
	if gateway && len(models) > gatewayModelLimit {
		models = models[:gatewayModelLimit]
	}
	return models, nil
}

func ProviderError(prefix string, statusCode int, body []byte) error {
	msg := extractProviderError(body)
	if msg == "" {
		msg = http.StatusText(statusCode)
	}
	return fmt.Errorf("%s returned status %d: %s", prefix, statusCode, truncate(msg, 300))
}

func isUsefulGatewayChatModel(modelType string, tags []string) bool {
	if modelType != "" && modelType != "language" {
		return false
	}
	if len(tags) == 0 {
		return true
	}
	for _, tag := range tags {
		if tag == "tool-use" {
			return true
		}
	}
	return false
}

func sortModels(models []Model, gateway bool) {
	priority := map[string]int{}
	if gateway {
		for i, id := range gatewayModelPriority {
			priority[id] = i + 1
		}
	}
	sort.Slice(models, func(i, j int) bool {
		pi, iPriority := priority[models[i].ID]
		pj, jPriority := priority[models[j].ID]
		if iPriority || jPriority {
			if !iPriority {
				return false
			}
			if !jPriority {
				return true
			}
			return pi < pj
		}
		return strings.ToLower(models[i].Name) < strings.ToLower(models[j].Name)
	})
}

func extractProviderError(body []byte) string {
	var openAIError struct {
		Error any `json:"error"`
	}
	if err := json.Unmarshal(body, &openAIError); err == nil {
		switch e := openAIError.Error.(type) {
		case string:
			return e
		case map[string]any:
			if msg, ok := e["message"].(string); ok {
				return msg
			}
			if code, ok := e["code"].(string); ok {
				return code
			}
		}
	}
	return strings.TrimSpace(string(body))
}
