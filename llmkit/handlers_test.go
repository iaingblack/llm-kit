package llmkit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGetConfigReportsEnvKeyWithoutRevealingIt(t *testing.T) {
	h := &Handler{
		Store: NewMemoryStore(Config{
			BaseURL: VercelGatewayBaseURL,
			Model:   VercelGatewayDefaultModel,
		}),
		Env: func(name string) string {
			if name == VercelGatewayAPIKeyEnv {
				return "env-secret"
			}
			return ""
		},
	}

	rr := httptest.NewRecorder()
	h.HandleGetConfig(rr, httptest.NewRequest(http.MethodGet, "/llm/config", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "env-secret") {
		t.Fatal("config response exposed the environment API key")
	}
	var resp PublicConfig
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.HasAPIKey || resp.APIKeySource != KeySourceEnvironment {
		t.Fatalf("response = %+v", resp)
	}
}

func TestHandleUpdateConfigNativeProviderDoesNotStoreAPIKey(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	h := &Handler{Store: store}
	body := bytes.NewBufferString(`{
		"provider":"codex_cli",
		"api_key":"should-not-be-stored",
		"model":"codex-default"
	}`)

	rr := httptest.NewRecorder()
	h.HandleUpdateConfig(rr, httptest.NewRequest(http.MethodPut, "/llm/config", body))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	cfg, err := store.GetLLMConfig(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderCodexCLI {
		t.Fatalf("Provider = %q", cfg.Provider)
	}
	if cfg.APIKey != "" || cfg.BaseURL != "" {
		t.Fatalf("native provider stored URL/key: %+v", cfg)
	}
	if strings.Contains(rr.Body.String(), "should-not-be-stored") {
		t.Fatal("response exposed native provider API key")
	}
}
