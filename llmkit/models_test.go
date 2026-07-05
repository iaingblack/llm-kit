package llmkit

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeGatewayModelsFiltersAndPrioritizesToolUseLanguageModels(t *testing.T) {
	body := []byte(`{
		"object":"list",
		"data":[
			{"id":"image/model","name":"Image","type":"image","tags":["tool-use"]},
			{"id":"text/no-tools","name":"No Tools","type":"language","tags":["vision"]},
			{"id":"openai/gpt-5.5","name":"GPT 5.5","type":"language","tags":["tool-use"]},
			{"id":"anthropic/claude-sonnet-5","name":"Claude Sonnet 5","type":"language","tags":["tool-use"]},
			{"id":"z/model","name":"Zed","type":"language","tags":["tool-use"]}
		]
	}`)

	models, err := NormalizeModels(body, true)
	if err != nil {
		t.Fatalf("NormalizeModels: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("len(models) = %d, want 3: %+v", len(models), models)
	}
	if models[0].ID != VercelGatewayDefaultModel {
		t.Fatalf("first model = %q, want default %q", models[0].ID, VercelGatewayDefaultModel)
	}
	if models[1].ID != "openai/gpt-5.5" {
		t.Fatalf("second model = %q, want openai/gpt-5.5", models[1].ID)
	}
}

func TestModelListerDoesNotSendAuthorizationForGateway(t *testing.T) {
	var auth string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		auth = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"data":[{"id":"anthropic/claude-sonnet-5","name":"Claude Sonnet 5","type":"language","tags":["tool-use"]}]}`,
			)),
			Request: req,
		}, nil
	})}

	models, err := (ModelLister{Client: client}).List(context.Background(), Config{
		BaseURL: VercelGatewayBaseURL,
		APIKey:  "stale-or-invalid-key",
		Model:   VercelGatewayDefaultModel,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if auth != "" {
		t.Fatalf("Authorization = %q, want empty for gateway model list", auth)
	}
	if len(models) != 1 || models[0].ID != VercelGatewayDefaultModel {
		t.Fatalf("models = %+v", models)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
