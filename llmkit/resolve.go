package llmkit

import (
	"net/http"
	"os"
	"strings"
)

func Resolve(cfg Config) ResolvedConfig {
	return ResolveWithEnv(cfg, os.Getenv)
}

func ResolveWithEnv(cfg Config, getenv EnvFunc) ResolvedConfig {
	cfg = NormalizeConfig(cfg)
	keySource := KeySourceNone
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey != "" {
		keySource = KeySourceStored
	}

	if IsNativeProvider(cfg.Provider) {
		return ResolvedConfig{
			Model:       cfg.Model,
			KeySource:   KeySourceNone,
			ReadOnly:    cfg.ReadOnly,
			Provider:    cfg.Provider,
			IsNative:    true,
			NeedsAPIKey: false,
		}
	}

	isGateway := IsVercelGateway(cfg.BaseURL)
	if apiKey == "" && isGateway && getenv != nil {
		if envKey := strings.TrimSpace(getenv(VercelGatewayAPIKeyEnv)); envKey != "" {
			apiKey = envKey
			keySource = KeySourceEnvironment
		}
	}

	local := IsLocalProvider(cfg.BaseURL)
	provider := cfg.Provider
	if isGateway {
		provider = ProviderVercelGateway
	} else if local {
		provider = ProviderLocal
	} else if provider == "" {
		provider = ProviderOpenAICompatible
	}

	return ResolvedConfig{
		BaseURL:     cfg.BaseURL,
		APIKey:      apiKey,
		Model:       cfg.Model,
		KeySource:   keySource,
		ReadOnly:    cfg.ReadOnly,
		Provider:    provider,
		Local:       local,
		IsGateway:   isGateway,
		IsNative:    false,
		NeedsAPIKey: !local,
	}
}

func (r ResolvedConfig) HasAPIKey() bool {
	return r.APIKey != ""
}

func (r ResolvedConfig) Public() PublicConfig {
	return PublicConfig{
		BaseURL:      r.BaseURL,
		Model:        r.Model,
		HasAPIKey:    r.HasAPIKey(),
		APIKeySource: r.KeySource,
		ReadOnly:     r.ReadOnly,
		Provider:     r.Provider,
	}
}

func (r ResolvedConfig) ChatCompletionsURL() string {
	return endpointURL(r.BaseURL, "chat/completions")
}

func (r ResolvedConfig) ModelsURL() string {
	return endpointURL(r.BaseURL, "models")
}

func (r ResolvedConfig) Authorize(req *http.Request) {
	if r.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.APIKey)
	}
}
