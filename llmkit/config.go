package llmkit

import "strings"

const (
	VercelGatewayBaseURL      = "https://ai-gateway.vercel.sh/v1"
	VercelGatewayDefaultModel = "anthropic/claude-sonnet-5"
	VercelGatewayAPIKeyEnv    = "AI_GATEWAY_API_KEY"
)

const (
	ProviderOpenAICompatible = "openai_compatible"
	ProviderVercelGateway    = "vercel_ai_gateway"
	ProviderLocal            = "local"
	ProviderCodexCLI         = "codex_cli"
	ProviderClaudeCodeCLI    = "claude_code_cli"
)

const (
	KeySourceNone        = "none"
	KeySourceStored      = "stored"
	KeySourceEnvironment = "environment"
)

type Config struct {
	BaseURL  string `json:"base_url,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	Model    string `json:"model,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty"`
	Provider string `json:"provider,omitempty"`
}

type ConfigPatch struct {
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	ReadOnly *bool  `json:"read_only,omitempty"`
	Provider string `json:"provider"`
}

type PublicConfig struct {
	BaseURL      string `json:"base_url"`
	Model        string `json:"model"`
	HasAPIKey    bool   `json:"has_api_key"`
	APIKeySource string `json:"api_key_source"`
	ReadOnly     bool   `json:"read_only"`
	Provider     string `json:"provider"`
}

type ResolvedConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	KeySource   string
	ReadOnly    bool
	Provider    string
	Local       bool
	IsGateway   bool
	IsNative    bool
	NeedsAPIKey bool
}

type EnvFunc func(string) string

func DefaultConfig() Config {
	return Config{
		BaseURL:  VercelGatewayBaseURL,
		Model:    VercelGatewayDefaultModel,
		Provider: ProviderVercelGateway,
	}
}

func NormalizeConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.Provider) == "" && strings.TrimSpace(cfg.BaseURL) == "" && strings.TrimSpace(cfg.Model) == "" {
		def := DefaultConfig()
		def.APIKey = cfg.APIKey
		def.ReadOnly = cfg.ReadOnly
		return def
	}
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	if cfg.Provider == "" && IsVercelGateway(cfg.BaseURL) {
		cfg.Provider = ProviderVercelGateway
	}
	if cfg.Provider == "" && IsLocalProvider(cfg.BaseURL) {
		cfg.Provider = ProviderLocal
	}
	if cfg.Provider == "" {
		cfg.Provider = ProviderOpenAICompatible
	}
	return cfg
}

func IsNativeProvider(provider string) bool {
	return provider == ProviderCodexCLI || provider == ProviderClaudeCodeCLI
}

func IsAllowedProvider(provider string) bool {
	switch provider {
	case "", ProviderOpenAICompatible, ProviderVercelGateway, ProviderLocal, ProviderCodexCLI, ProviderClaudeCodeCLI:
		return true
	default:
		return false
	}
}

func NativeProviderLabel(provider string) string {
	switch provider {
	case ProviderCodexCLI:
		return "ChatGPT (Codex CLI)"
	case ProviderClaudeCodeCLI:
		return "Claude Code"
	default:
		return provider
	}
}

func NativeProviderModels(provider string) []Model {
	switch provider {
	case ProviderCodexCLI:
		return []Model{
			{ID: "codex-default", Name: "Codex CLI default"},
			{ID: "gpt-5.5", Name: "GPT 5.5"},
		}
	case ProviderClaudeCodeCLI:
		return []Model{
			{ID: "claude-default", Name: "Claude Code default"},
			{ID: "sonnet", Name: "Claude Sonnet"},
			{ID: "opus", Name: "Claude Opus"},
		}
	default:
		return []Model{}
	}
}
