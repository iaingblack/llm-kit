# llm-kit

Drop-in LLM provider settings for Go backends and Vue frontends.

This repo packages the provider setup work that keeps getting reimplemented:

- Vercel AI Gateway as the recommended multi-provider path.
- OpenAI-compatible custom providers such as OpenRouter.
- Local OpenAI-compatible providers such as Ollama.
- Server-side API key handling with explicit key-source reporting.
- Model discovery and normalization.
- Native CLI login detection for ChatGPT via Codex CLI and Claude Code.
- Optional native CLI structured completion adapter.
- A headless Vue composable plus a plain default settings modal.

The kit deliberately stops at provider access and settings UI. Your app still owns prompts, tools, chat history, streaming, authorization, billing policy, database persistence, and any product-specific confirmation flow.

## Repository Layout

```text
llmkit/                 Go package
  config.go             provider IDs, defaults, public config shape
  resolve.go            env fallback and resolved config
  urls.go               SSRF-aware base URL validation
  models.go             OpenAI-compatible /models listing
  native_auth.go        codex/claude login-status detection
  native_cli.go         optional structured native CLI completion
  handlers.go           net/http handlers for settings endpoints
vue/
  src/useLLMSettings.js headless Vue state/composable
  src/LLMSettingsModal.vue default restylable modal
  src/apiClient.js      small fetch client for the API contract
examples/go-nethttp/    minimal Go server
```

## Go Backend

Install:

```bash
go get github.com/iaingblack/llm-kit
```

Mount the settings endpoints behind your own auth middleware:

```go
package main

import (
	"net/http"

	llmkit "github.com/iaingblack/llm-kit/llmkit"
)

func main() {
	store := llmkit.NewMemoryStore(llmkit.DefaultConfig())
	settings := llmkit.NewHandler(store)

	mux := http.NewServeMux()
	settings.Register(mux, "/api/llm")

	http.ListenAndServe(":8080", authMiddleware(mux))
}
```

For production, implement `llmkit.Store` with your app's config system:

```go
type Store interface {
	GetLLMConfig(context.Context) (llmkit.Config, error)
	SaveLLMConfig(context.Context, llmkit.Config) (llmkit.Config, error)
}
```

`api_key` is accepted on update, but never returned by the config response.

## API Contract

The default Vue client expects:

```http
GET /api/llm/config
PUT /api/llm/config
GET /api/llm/models
GET /api/llm/native-auth
```

Config response:

```json
{
  "provider": "vercel_ai_gateway",
  "base_url": "https://ai-gateway.vercel.sh/v1",
  "model": "anthropic/claude-sonnet-5",
  "has_api_key": true,
  "api_key_source": "environment",
  "read_only": false
}
```

Config update:

```json
{
  "provider": "codex_cli",
  "base_url": "",
  "api_key": "",
  "model": "codex-default",
  "read_only": true
}
```

Model list:

```json
[
  { "id": "anthropic/claude-sonnet-5", "name": "Claude Sonnet 5" }
]
```

Native auth status:

```json
{
  "providers": [
    {
      "id": "codex",
      "label": "ChatGPT (Codex CLI)",
      "command": "codex login status",
      "installed": true,
      "authenticated": true,
      "status": "authenticated"
    }
  ]
}
```

## Vue Frontend

Use the headless composable if you want to build your own UI:

```js
import { createLLMSettingsClient, useLLMSettings } from './llm-kit/vue/src/index.js'

const api = createLLMSettingsClient({ basePath: '/api/llm' })
const settings = useLLMSettings(api)

await settings.loadConfig()
await settings.loadNativeAuthStatus()
```

Use the default modal if you want something ready to drop in:

```vue
<script setup>
import { createLLMSettingsClient, LLMSettingsModal } from './llm-kit/vue/src/index.js'

const api = createLLMSettingsClient({ basePath: '/api/llm' })
</script>

<template>
  <LLMSettingsModal
    :api="api"
    title="Chat Settings"
    @close="showSettings = false"
    @saved="toast.success('Saved')"
    @error="toast.error($event.message)"
  />
</template>
```

The default modal has no Tailwind or icon dependency. Style it with CSS variables:

```css
:root {
  --llmk-surface: #172033;
  --llmk-text: #e5e7eb;
  --llmk-border: #334155;
  --llmk-primary: #3b82f6;
}
```

## Copy/Clone Workflow

For projects where you do not want package dependency management yet:

```bash
git clone https://github.com/iaingblack/llm-kit.git
cp -R llm-kit/llmkit ./internal/llmkit
cp -R llm-kit/vue/src ./frontend/src/llm-kit
```

Then change imports to match your local paths.

## Providers

| Provider | ID | Notes |
| --- | --- | --- |
| Vercel AI Gateway | `vercel_ai_gateway` | Default. Uses `https://ai-gateway.vercel.sh/v1`. Can use saved key or server `AI_GATEWAY_API_KEY`. |
| OpenAI-compatible | `openai_compatible` | For OpenRouter or any `/v1/chat/completions` + `/v1/models` provider. |
| Local | `local` | For loopback providers such as Ollama at `http://localhost:11434/v1`. |
| ChatGPT (Codex CLI) | `codex_cli` | Uses the server user's `codex login status` and `codex exec`. |
| Claude Code | `claude_code_cli` | Uses the server user's `claude auth status` and `claude --print`. |

## Native CLI Notes

Native CLI auth detection only runs fixed commands:

```bash
codex login status
claude auth status
```

The kit does not read token files and does not send CLI tokens to the browser. The optional native completion adapter invokes fixed `codex` or `claude` arguments without a shell, sends the prompt on stdin, and asks for a structured JSON response. The host app must still decide what, if anything, to do with returned tool calls.

## Security Defaults

- API keys are server-side only.
- Config responses return `has_api_key` and `api_key_source`, never the key.
- Switching remote provider hosts clears stored keys in the handler.
- Configurable base URLs allow HTTPS public endpoints and loopback HTTP only.
- Private, link-local, multicast, metadata, userinfo, query, and fragment URL forms are rejected.
- Native auth checks discard stdout/stderr.

## What This Is Not

This is not an agent framework. It does not manage chat history, stream model tokens, define tools, execute tools, implement OAuth, provide BYOK management, or store anything in a database. It is the provider settings and access layer you can reuse before wiring in your app's actual chat behavior.
