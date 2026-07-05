package llmkit

import "testing"

func TestResolveUsesGatewayEnvFallback(t *testing.T) {
	resolved := ResolveWithEnv(Config{
		BaseURL: VercelGatewayBaseURL,
		Model:   VercelGatewayDefaultModel,
	}, func(name string) string {
		if name == VercelGatewayAPIKeyEnv {
			return "env-key"
		}
		return ""
	})

	if resolved.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want env-key", resolved.APIKey)
	}
	if resolved.KeySource != KeySourceEnvironment {
		t.Fatalf("KeySource = %q, want %q", resolved.KeySource, KeySourceEnvironment)
	}
	if !resolved.IsGateway || resolved.Provider != ProviderVercelGateway {
		t.Fatalf("gateway not detected: %+v", resolved)
	}
}

func TestResolveStoredKeyWinsOverGatewayEnv(t *testing.T) {
	resolved := ResolveWithEnv(Config{
		BaseURL: VercelGatewayBaseURL,
		APIKey:  "stored-key",
		Model:   VercelGatewayDefaultModel,
	}, func(string) string {
		return "env-key"
	})

	if resolved.APIKey != "stored-key" {
		t.Fatalf("APIKey = %q, want stored-key", resolved.APIKey)
	}
	if resolved.KeySource != KeySourceStored {
		t.Fatalf("KeySource = %q, want %q", resolved.KeySource, KeySourceStored)
	}
}

func TestIsAllowedBaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		ok   bool
	}{
		{"gateway", VercelGatewayBaseURL, true},
		{"https domain", "https://openrouter.ai/api/v1", true},
		{"local ollama", "http://localhost:11434/v1", true},
		{"local loopback", "http://127.0.0.1:11434/v1", true},
		{"local ipv6 loopback", "http://[::1]:11434/v1", true},
		{"http remote", "http://example.com/v1", false},
		{"http query containing localhost", "http://example.com/v1?host=localhost", false},
		{"https private ip", "https://192.168.1.10/v1", false},
		{"https metadata ip", "https://169.254.169.254/v1", false},
		{"userinfo", "https://key@example.com/v1", false},
		{"fragment", "https://example.com/v1#models", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllowedBaseURL(tt.url); got != tt.ok {
				t.Fatalf("IsAllowedBaseURL(%q) = %v, want %v", tt.url, got, tt.ok)
			}
		})
	}
}
