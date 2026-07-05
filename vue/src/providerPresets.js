export const VERCEL_GATEWAY_BASE_URL = 'https://ai-gateway.vercel.sh/v1'
export const VERCEL_GATEWAY_DEFAULT_MODEL = 'anthropic/claude-sonnet-5'
export const OPENROUTER_BASE_URL = 'https://openrouter.ai/api/v1'
export const OPENROUTER_DEFAULT_MODEL = 'anthropic/claude-sonnet-4'
export const OLLAMA_BASE_URL = 'http://localhost:11434/v1'
export const OLLAMA_DEFAULT_MODEL = 'llama3.2'

export const PROVIDER_GATEWAY = 'vercel_ai_gateway'
export const PROVIDER_OPENAI_COMPATIBLE = 'openai_compatible'
export const PROVIDER_LOCAL = 'local'
export const PROVIDER_CODEX = 'codex_cli'
export const PROVIDER_CLAUDE = 'claude_code_cli'

export const providerPresets = {
  gateway: {
    provider: PROVIDER_GATEWAY,
    baseUrl: VERCEL_GATEWAY_BASE_URL,
    model: VERCEL_GATEWAY_DEFAULT_MODEL,
  },
  openrouter: {
    provider: PROVIDER_OPENAI_COMPATIBLE,
    baseUrl: OPENROUTER_BASE_URL,
    model: OPENROUTER_DEFAULT_MODEL,
  },
  ollama: {
    provider: PROVIDER_LOCAL,
    baseUrl: OLLAMA_BASE_URL,
    model: OLLAMA_DEFAULT_MODEL,
  },
}

export function nativeProviderValue(nativeAuthProvider) {
  if (nativeAuthProvider?.id === 'codex') return PROVIDER_CODEX
  if (nativeAuthProvider?.id === 'claude') return PROVIDER_CLAUDE
  return ''
}

export function defaultNativeModel(provider) {
  if (provider === PROVIDER_CODEX) return 'codex-default'
  if (provider === PROVIDER_CLAUDE) return 'claude-default'
  return ''
}

export function normalizeBaseUrl(value) {
  return (value || '').trim().replace(/\/+$/, '')
}

export function isNativeProvider(provider) {
  return provider === PROVIDER_CODEX || provider === PROVIDER_CLAUDE
}

export function isGatewayProvider(provider, baseUrl) {
  return provider === PROVIDER_GATEWAY || normalizeBaseUrl(baseUrl) === VERCEL_GATEWAY_BASE_URL
}

export function isLocalBaseUrl(baseUrl) {
  const value = (baseUrl || '').toLowerCase()
  return value.includes('localhost') || value.includes('127.0.0.1') || value.includes('[::1]')
}
