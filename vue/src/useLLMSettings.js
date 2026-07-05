import { computed, ref } from 'vue'
import {
  defaultNativeModel,
  isGatewayProvider,
  isLocalBaseUrl,
  isNativeProvider,
  nativeProviderValue,
  normalizeBaseUrl,
  providerPresets,
  PROVIDER_GATEWAY,
} from './providerPresets.js'
import { fromWireConfig } from './apiClient.js'

export function useLLMSettings(api, options = {}) {
  if (!api) {
    throw new Error('useLLMSettings requires API functions')
  }

  const config = ref({
    provider: PROVIDER_GATEWAY,
    baseUrl: '',
    model: '',
    hasApiKey: false,
    apiKeySource: 'none',
    readOnly: false,
  })
  const draft = ref({
    provider: PROVIDER_GATEWAY,
    baseUrl: '',
    apiKey: '',
    model: '',
    readOnly: false,
  })
  const models = ref([])
  const modelSearch = ref('')
  const nativeAuthProviders = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const testing = ref(false)
  const nativeAuthLoading = ref(false)
  const error = ref(null)
  const modelsError = ref(null)
  const nativeAuthError = ref(null)
  const connectionTested = ref(false)

  const native = computed(() => isNativeProvider(draft.value.provider))
  const gateway = computed(() => isGatewayProvider(draft.value.provider, draft.value.baseUrl))
  const local = computed(() => isLocalBaseUrl(draft.value.baseUrl))
  const savedBaseUrlMatches = computed(() => {
    return normalizeBaseUrl(draft.value.baseUrl) === normalizeBaseUrl(config.value.baseUrl)
  })
  const canConnect = computed(() => {
    if (native.value) return true
    if (!draft.value.baseUrl) return false
    if (gateway.value || local.value) return true
    return !!(draft.value.apiKey || (savedBaseUrlMatches.value && config.value.hasApiKey))
  })
  const apiKeyBadge = computed(() => {
    if (draft.value.apiKey || !savedBaseUrlMatches.value) return ''
    if (config.value.apiKeySource === 'environment' && gateway.value) return 'server env'
    if (config.value.apiKeySource === 'stored' || config.value.hasApiKey) return 'configured'
    return ''
  })
  const apiKeyHint = computed(() => {
    if (local.value) return 'Not required for local providers'
    if (gateway.value && savedBaseUrlMatches.value && config.value.apiKeySource === 'environment' && !draft.value.apiKey) {
      return 'Using server AI_GATEWAY_API_KEY'
    }
    if (gateway.value && !(savedBaseUrlMatches.value && config.value.hasApiKey) && !draft.value.apiKey) {
      return 'Models can be browsed without a key; chat needs a saved key or server AI_GATEWAY_API_KEY'
    }
    return 'Enter key and connect to browse models'
  })
  const filteredModels = computed(() => {
    const q = modelSearch.value.toLowerCase()
    if (!q) return models.value
    return models.value.filter((model) =>
      model.id.toLowerCase().includes(q) || model.name.toLowerCase().includes(q)
    )
  })

  async function loadConfig() {
    loading.value = true
    error.value = null
    try {
      const loaded = fromWireConfig(await api.getConfig())
      config.value = loaded
      draft.value = {
        provider: loaded.provider,
        baseUrl: loaded.baseUrl,
        apiKey: '',
        model: loaded.model,
        readOnly: loaded.readOnly,
      }
      modelSearch.value = loaded.model
      return loaded
    } catch (err) {
      error.value = err.message || 'Failed to load LLM settings'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function loadNativeAuthStatus() {
    nativeAuthLoading.value = true
    nativeAuthError.value = null
    try {
      const result = await api.getNativeAuthStatus()
      nativeAuthProviders.value = result.providers || []
      return nativeAuthProviders.value
    } catch (err) {
      nativeAuthError.value = err.message || 'Failed to check native auth'
      nativeAuthProviders.value = []
      return []
    } finally {
      nativeAuthLoading.value = false
    }
  }

  async function save() {
    saving.value = true
    error.value = null
    try {
      const saved = fromWireConfig(await api.saveConfig(draft.value))
      config.value = saved
      draft.value.apiKey = ''
      return saved
    } catch (err) {
      error.value = err.message || 'Failed to save LLM settings'
      throw err
    } finally {
      saving.value = false
    }
  }

  async function connectAndFetchModels() {
    if (!draft.value.baseUrl && !native.value) return []
    testing.value = true
    modelsError.value = null
    connectionTested.value = false
    try {
      await save()
      const result = await api.listModels()
      models.value = result || []
      connectionTested.value = true
      return models.value
    } catch (err) {
      modelsError.value = err.message || 'Failed to connect'
      models.value = []
      return []
    } finally {
      testing.value = false
    }
  }

  function setPreset(name) {
    const preset = providerPresets[name]
    if (!preset) return
    connectionTested.value = false
    models.value = []
    modelsError.value = null
    draft.value.provider = preset.provider
    draft.value.baseUrl = preset.baseUrl
    draft.value.model = preset.model
    draft.value.apiKey = ''
    modelSearch.value = preset.model
    if (options.autoConnectPresets?.includes(name)) {
      connectAndFetchModels()
    }
  }

  function setNativeProvider(provider) {
    draft.value.provider = provider
    draft.value.baseUrl = ''
    draft.value.apiKey = ''
    draft.value.model = defaultNativeModel(provider)
    modelSearch.value = draft.value.model
    return connectAndFetchModels()
  }

  function useNativeAuthProvider(nativeAuthProvider) {
    const provider = nativeProviderValue(nativeAuthProvider)
    if (provider) {
      return setNativeProvider(provider)
    }
    return Promise.resolve([])
  }

  function selectModel(model) {
    draft.value.model = model.id
    modelSearch.value = model.id
  }

  return {
    apiKeyBadge,
    apiKeyHint,
    canConnect,
    config,
    connectAndFetchModels,
    connectionTested,
    draft,
    error,
    filteredModels,
    gateway,
    loadConfig,
    loadNativeAuthStatus,
    loading,
    local,
    modelSearch,
    models,
    modelsError,
    native,
    nativeAuthError,
    nativeAuthLoading,
    nativeAuthProviders,
    save,
    saving,
    selectModel,
    setNativeProvider,
    setPreset,
    testing,
    useNativeAuthProvider,
  }
}
