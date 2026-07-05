package llmkit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Handler struct {
	Store       Store
	Env         EnvFunc
	ModelLister ModelLister
	NativeAuth  NativeAuthChecker
}

func NewHandler(store Store) *Handler {
	return &Handler{Store: store}
}

func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	prefix = "/" + strings.Trim(strings.TrimSpace(prefix), "/")
	prefix = strings.TrimRight(prefix, "/")
	mux.HandleFunc("GET "+prefix+"/config", h.HandleGetConfig)
	mux.HandleFunc("PUT "+prefix+"/config", h.HandleUpdateConfig)
	mux.HandleFunc("GET "+prefix+"/models", h.HandleListModels)
	mux.HandleFunc("GET "+prefix+"/native-auth", h.HandleNativeAuth)
}

func (h *Handler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store().GetLLMConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load LLM configuration")
		return
	}
	writeJSON(w, http.StatusOK, ResolveWithEnv(cfg, h.env()).Public())
}

func (h *Handler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var patch ConfigPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	patch.BaseURL = strings.TrimSpace(patch.BaseURL)
	patch.APIKey = strings.TrimSpace(patch.APIKey)
	patch.Model = strings.TrimSpace(patch.Model)
	patch.Provider = strings.TrimSpace(patch.Provider)
	if !IsAllowedProvider(patch.Provider) {
		writeError(w, http.StatusBadRequest, "invalid provider")
		return
	}
	if !IsNativeProvider(patch.Provider) && patch.BaseURL != "" && !IsAllowedBaseURL(patch.BaseURL) {
		writeError(w, http.StatusBadRequest, "invalid base URL: must be HTTPS or a local address")
		return
	}

	cfg, err := h.store().GetLLMConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load LLM configuration")
		return
	}
	cfg = applyPatch(cfg, patch)
	cfg, err = h.store().SaveLLMConfig(r.Context(), cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save LLM configuration")
		return
	}
	writeJSON(w, http.StatusOK, ResolveWithEnv(cfg, h.env()).Public())
}

func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store().GetLLMConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load LLM configuration")
		return
	}

	lister := h.ModelLister
	if lister.Env == nil {
		lister.Env = h.env()
	}
	models, err := lister.List(r.Context(), cfg)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "API key required") ||
			strings.Contains(err.Error(), "not configured") ||
			strings.Contains(err.Error(), "invalid LLM base URL") {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *Handler) HandleNativeAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": h.nativeAuth().CheckAll(r.Context()),
	})
}

func applyPatch(cfg Config, patch ConfigPatch) Config {
	cfg = NormalizeConfig(cfg)
	if patch.Provider != "" {
		cfg.Provider = patch.Provider
	}
	if IsNativeProvider(cfg.Provider) {
		cfg.BaseURL = ""
		cfg.APIKey = ""
	} else if patch.BaseURL != "" {
		if cfg.BaseURL != "" && !SameURLHost(cfg.BaseURL, patch.BaseURL) {
			cfg.APIKey = ""
		}
		cfg.BaseURL = patch.BaseURL
	}
	if patch.APIKey != "" && !IsNativeProvider(cfg.Provider) {
		cfg.APIKey = patch.APIKey
	}
	if patch.Model != "" {
		cfg.Model = patch.Model
	}
	if patch.ReadOnly != nil {
		cfg.ReadOnly = *patch.ReadOnly
	}
	return NormalizeConfig(cfg)
}

func (h *Handler) store() Store {
	if h.Store != nil {
		return h.Store
	}
	return NewMemoryStore(DefaultConfig())
}

func (h *Handler) env() EnvFunc {
	if h.Env != nil {
		return h.Env
	}
	return nil
}

func (h *Handler) nativeAuth() NativeAuthChecker {
	return h.NativeAuth
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func MethodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, fmt.Sprintf("method not allowed; use %s", method))
}
