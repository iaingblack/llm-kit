export function createLLMSettingsClient({ basePath = '/api/llm', request } = {}) {
  const send = request || defaultRequest
  const root = basePath.replace(/\/+$/, '')

  return {
    getConfig: () => send(`${root}/config`),
    saveConfig: (config) => send(`${root}/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(toWireConfig(config)),
    }),
    listModels: () => send(`${root}/models`),
    getNativeAuthStatus: () => send(`${root}/native-auth`),
  }
}

async function defaultRequest(url, options = {}) {
  const response = await fetch(url, {
    credentials: 'include',
    ...options,
  })
  const text = await response.text()
  const data = text ? JSON.parse(text) : null
  if (!response.ok) {
    throw new Error(data?.error || data?.message || `Request failed with status ${response.status}`)
  }
  return data
}

export function toWireConfig(config) {
  return {
    provider: config.provider || '',
    base_url: config.baseUrl || '',
    api_key: config.apiKey || '',
    model: config.model || '',
    read_only: config.readOnly,
  }
}

export function fromWireConfig(config = {}) {
  return {
    provider: config.provider || 'openai_compatible',
    baseUrl: config.base_url || '',
    model: config.model || '',
    hasApiKey: !!config.has_api_key,
    apiKeySource: config.api_key_source || 'none',
    readOnly: !!config.read_only,
  }
}
