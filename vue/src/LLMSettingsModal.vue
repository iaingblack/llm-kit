<script setup>
import { computed, onMounted, ref } from 'vue'
import { nativeProviderValue } from './providerPresets.js'
import { useLLMSettings } from './useLLMSettings.js'

const props = defineProps({
  api: {
    type: Object,
    required: true,
  },
  title: {
    type: String,
    default: 'LLM Settings',
  },
  showReadOnly: {
    type: Boolean,
    default: true,
  },
  closeOnSave: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits(['close', 'saved', 'error'])

const showModelDropdown = ref(false)
const settings = useLLMSettings(props.api, {
  autoConnectPresets: ['gateway', 'ollama'],
})

const modelPlaceholder = computed(() => {
  return settings.models.value.length ? 'Search models...' : 'Type model name or connect to browse'
})

onMounted(async () => {
  try {
    await settings.loadConfig()
    settings.loadNativeAuthStatus()
    if (settings.config.value.hasApiKey || settings.local.value || settings.gateway.value) {
      settings.connectAndFetchModels()
    }
  } catch (err) {
    emit('error', err)
  }
})

async function save() {
  try {
    const saved = await settings.save()
    emit('saved', saved)
    if (props.closeOnSave) {
      emit('close')
    }
  } catch (err) {
    emit('error', err)
  }
}

function nativeAuthLabel(provider) {
  if (provider.status === 'authenticated') return 'Detected'
  if (provider.status === 'not_installed') return 'Not installed'
  if (provider.status === 'timeout') return 'Timed out'
  return 'Not detected'
}

function nativeAuthClass(provider) {
  if (provider.status === 'authenticated') return 'llmk-status-ok'
  if (provider.status === 'not_installed') return 'llmk-status-muted'
  return 'llmk-status-warn'
}

function nativeProviderSelected(provider) {
  return settings.draft.value.provider === nativeProviderValue(provider)
}

function onModelFocus() {
  if (settings.models.value.length > 0) {
    showModelDropdown.value = true
    settings.modelSearch.value = ''
  }
}

function onModelBlur() {
  window.setTimeout(() => {
    showModelDropdown.value = false
    settings.modelSearch.value = settings.draft.value.model || ''
  }, 150)
}
</script>

<template>
  <div class="llmk-backdrop" @mousedown.self="emit('close')">
    <section class="llmk-modal" role="dialog" aria-modal="true" :aria-label="title">
      <header class="llmk-header">
        <h2>{{ title }}</h2>
        <button class="llmk-icon-button" type="button" @click="emit('close')" aria-label="Close">
          x
        </button>
      </header>

      <div class="llmk-body">
        <slot name="before" :settings="settings" />

        <div class="llmk-field">
          <label>Quick Setup</label>
          <div class="llmk-button-row">
            <button type="button" class="llmk-button" @click="settings.setPreset('gateway')">
              Vercel AI Gateway
            </button>
            <button type="button" class="llmk-button" @click="settings.setPreset('openrouter')">
              OpenRouter
            </button>
            <button type="button" class="llmk-button" @click="settings.setPreset('ollama')">
              Ollama (local)
            </button>
          </div>
        </div>

        <div class="llmk-field">
          <div class="llmk-label-row">
            <label>Detected CLI Logins</label>
            <button
              type="button"
              class="llmk-link-button"
              :disabled="settings.nativeAuthLoading.value"
              @click="settings.loadNativeAuthStatus"
            >
              {{ settings.nativeAuthLoading.value ? 'Checking...' : 'Refresh' }}
            </button>
          </div>
          <div class="llmk-native-list">
            <div
              v-for="provider in settings.nativeAuthProviders.value"
              :key="provider.id"
              class="llmk-native-row"
            >
              <div class="llmk-native-main">
                <strong>{{ provider.label }}</strong>
                <code>{{ provider.command }}</code>
              </div>
              <div class="llmk-native-actions">
                <span :class="nativeAuthClass(provider)">{{ nativeAuthLabel(provider) }}</span>
                <button
                  v-if="provider.status === 'authenticated'"
                  type="button"
                  class="llmk-button llmk-button-small"
                  :class="{ 'llmk-button-selected': nativeProviderSelected(provider) }"
                  @click="settings.useNativeAuthProvider(provider)"
                >
                  {{ nativeProviderSelected(provider) ? 'Selected' : 'Use' }}
                </button>
              </div>
            </div>
            <div
              v-if="settings.nativeAuthLoading.value && settings.nativeAuthProviders.value.length === 0"
              class="llmk-empty"
            >
              Checking...
            </div>
          </div>
          <p v-if="settings.nativeAuthError.value" class="llmk-error">
            {{ settings.nativeAuthError.value }}
          </p>
        </div>

        <div v-if="!settings.native.value" class="llmk-field">
          <label>Base URL</label>
          <input v-model="settings.draft.value.baseUrl" class="llmk-input" type="text" />
        </div>

        <div v-if="!settings.native.value" class="llmk-field">
          <label>
            API Key
            <span v-if="settings.apiKeyBadge.value" class="llmk-badge">
              {{ settings.apiKeyBadge.value }}
            </span>
          </label>
          <div class="llmk-input-row">
            <input
              v-model="settings.draft.value.apiKey"
              class="llmk-input"
              type="password"
              :placeholder="settings.config.value.hasApiKey ? 'Configured' : 'Not set'"
            />
            <button
              type="button"
              class="llmk-button"
              :disabled="!settings.canConnect.value || settings.testing.value"
              @click="settings.connectAndFetchModels"
            >
              {{ settings.testing.value ? 'Testing...' : settings.connectionTested.value ? 'Connected' : 'Connect' }}
            </button>
          </div>
          <p :class="settings.modelsError.value ? 'llmk-error' : 'llmk-help'">
            {{ settings.modelsError.value || settings.apiKeyHint.value }}
          </p>
        </div>

        <div class="llmk-field llmk-model-field">
          <label>
            Model
            <span v-if="settings.models.value.length" class="llmk-help-inline">
              ({{ settings.models.value.length }} available)
            </span>
          </label>
          <input
            v-model="settings.modelSearch.value"
            class="llmk-input"
            type="text"
            :placeholder="modelPlaceholder"
            @focus="onModelFocus"
            @blur="onModelBlur"
            @input="settings.draft.value.model = settings.modelSearch.value"
          />
          <div
            v-if="showModelDropdown && settings.filteredModels.value.length"
            class="llmk-model-menu"
          >
            <button
              v-for="model in settings.filteredModels.value"
              :key="model.id"
              type="button"
              class="llmk-model-option"
              @mousedown.prevent="settings.selectModel(model)"
            >
              <span>{{ model.id }}</span>
              <small v-if="model.name !== model.id">{{ model.name }}</small>
            </button>
          </div>
        </div>

        <div v-if="showReadOnly" class="llmk-toggle-row">
          <div>
            <label>Read-only mode</label>
            <p class="llmk-help">Only allow informational queries.</p>
          </div>
          <input v-model="settings.draft.value.readOnly" type="checkbox" />
        </div>

        <slot name="after" :settings="settings" />
      </div>

      <footer class="llmk-footer">
        <button type="button" class="llmk-button" @click="emit('close')">
          Cancel
        </button>
        <button
          type="button"
          class="llmk-button llmk-button-primary"
          :disabled="settings.saving.value"
          @click="save"
        >
          {{ settings.saving.value ? 'Saving...' : 'Save' }}
        </button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.llmk-backdrop {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--llmk-backdrop, rgb(0 0 0 / 0.48));
  z-index: var(--llmk-z-index, 1000);
}

.llmk-modal {
  width: min(92vw, var(--llmk-modal-width, 460px));
  max-height: min(92vh, 760px);
  overflow: hidden;
  color: var(--llmk-text, #e5e7eb);
  background: var(--llmk-surface, #172033);
  border: 1px solid var(--llmk-border, #334155);
  border-radius: var(--llmk-radius, 10px);
  box-shadow: var(--llmk-shadow, 0 22px 70px rgb(0 0 0 / 0.36));
}

.llmk-header,
.llmk-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-color: var(--llmk-border, #334155);
}

.llmk-header {
  border-bottom: 1px solid var(--llmk-border, #334155);
}

.llmk-footer {
  justify-content: flex-end;
  border-top: 1px solid var(--llmk-border, #334155);
}

.llmk-header h2 {
  margin: 0;
  font-size: 18px;
}

.llmk-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 64vh;
  overflow: auto;
  padding: 16px;
}

.llmk-field {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.llmk-field label,
.llmk-toggle-row label {
  font-size: 13px;
  color: var(--llmk-muted, #9ca3af);
}

.llmk-label-row,
.llmk-input-row,
.llmk-button-row,
.llmk-toggle-row,
.llmk-native-row,
.llmk-native-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.llmk-label-row,
.llmk-toggle-row,
.llmk-native-row {
  justify-content: space-between;
}

.llmk-button-row {
  flex-wrap: wrap;
}

.llmk-input,
.llmk-button,
.llmk-icon-button,
.llmk-link-button {
  font: inherit;
}

.llmk-input {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  color: var(--llmk-text, #e5e7eb);
  background: var(--llmk-input-bg, #111827);
  border: 1px solid var(--llmk-border, #334155);
  border-radius: 7px;
  padding: 9px 10px;
}

.llmk-input-row .llmk-input {
  flex: 1;
}

.llmk-button,
.llmk-icon-button {
  color: var(--llmk-text, #e5e7eb);
  background: var(--llmk-button-bg, transparent);
  border: 1px solid var(--llmk-border, #334155);
  border-radius: 7px;
  padding: 8px 11px;
  cursor: pointer;
}

.llmk-button:hover,
.llmk-icon-button:hover {
  background: var(--llmk-hover, #243047);
}

.llmk-button:disabled,
.llmk-link-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.llmk-button-primary {
  color: var(--llmk-primary-text, #ffffff);
  background: var(--llmk-primary, #3b82f6);
  border-color: var(--llmk-primary, #3b82f6);
}

.llmk-button-small {
  padding: 5px 8px;
  font-size: 12px;
}

.llmk-button-selected {
  color: var(--llmk-ok, #38bdf8);
  border-color: var(--llmk-ok, #38bdf8);
}

.llmk-icon-button {
  width: 32px;
  height: 32px;
  padding: 0;
}

.llmk-link-button {
  color: var(--llmk-link, #93c5fd);
  background: transparent;
  border: 0;
  cursor: pointer;
}

.llmk-native-list {
  border: 1px solid var(--llmk-border, #334155);
  border-radius: 8px;
  overflow: hidden;
}

.llmk-native-row {
  padding: 10px;
  border-bottom: 1px solid var(--llmk-border, #334155);
}

.llmk-native-row:last-child {
  border-bottom: 0;
}

.llmk-native-main {
  min-width: 0;
}

.llmk-native-main strong,
.llmk-native-main code {
  display: block;
}

.llmk-native-main code {
  margin-top: 2px;
  overflow: hidden;
  color: var(--llmk-muted, #9ca3af);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.llmk-model-field {
  position: relative;
}

.llmk-model-menu {
  position: absolute;
  z-index: 2;
  top: 100%;
  left: 0;
  right: 0;
  max-height: 220px;
  overflow: auto;
  background: var(--llmk-surface, #172033);
  border: 1px solid var(--llmk-border, #334155);
  border-radius: 8px;
}

.llmk-model-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  padding: 8px 10px;
  color: var(--llmk-text, #e5e7eb);
  text-align: left;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.llmk-model-option:hover {
  background: var(--llmk-hover, #243047);
}

.llmk-model-option small,
.llmk-help,
.llmk-help-inline,
.llmk-empty {
  color: var(--llmk-muted, #9ca3af);
}

.llmk-help,
.llmk-error {
  margin: 0;
  font-size: 12px;
}

.llmk-error,
.llmk-status-warn {
  color: var(--llmk-error, #f87171);
}

.llmk-status-ok {
  color: var(--llmk-ok, #38bdf8);
}

.llmk-status-muted {
  color: var(--llmk-muted, #9ca3af);
}

.llmk-badge {
  margin-left: 6px;
  color: var(--llmk-ok, #38bdf8);
}
</style>
