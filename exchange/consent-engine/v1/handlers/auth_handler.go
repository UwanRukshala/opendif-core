package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gov-dx-sandbox/exchange/consent-engine/v1/auth"
	"github.com/gov-dx-sandbox/exchange/consent-engine/v1/models"
	"github.com/gov-dx-sandbox/exchange/consent-engine/v1/utils"
)

// AuthHandlerConfig holds dependencies for SLUDI / eSignet auth flows.
type AuthHandlerConfig struct {
	ClientID          string
	Scope             string
	EsignetBaseURL    string
	CallbackURL       string
	ConsentPortalURL  string
	TokenExchanger    *auth.TokenExchanger
	LoginStateStore   *auth.LoginStateStore
	TokenSessionStore *auth.TokenSessionStore
}

// AuthHandler handles OIDC login redirect and token exchange for the consent portal.
type AuthHandler struct {
	clientID          string
	scope             string
	esignetBaseURL    string
	callbackURL       string
	consentPortalURL  string
	tokenExchanger    *auth.TokenExchanger
	loginStateStore   *auth.LoginStateStore
	tokenSessionStore *auth.TokenSessionStore
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(cfg AuthHandlerConfig) *AuthHandler {
	store := cfg.LoginStateStore
	if store == nil {
		store = auth.NewLoginStateStore()
	}
	tokenSessions := cfg.TokenSessionStore
	if tokenSessions == nil {
		tokenSessions = auth.NewTokenSessionStore()
	}

	return &AuthHandler{
		clientID:          cfg.ClientID,
		scope:             cfg.Scope,
		esignetBaseURL:    cfg.EsignetBaseURL,
		callbackURL:       cfg.CallbackURL,
		consentPortalURL:  cfg.ConsentPortalURL,
		tokenExchanger:    cfg.TokenExchanger,
		loginStateStore:   store,
		tokenSessionStore: tokenSessions,
	}
}

func (h *AuthHandler) authConfigured() bool {
	return h.tokenExchanger != nil &&
		h.clientID != "" &&
		h.esignetBaseURL != "" &&
		h.callbackURL != ""
}

// Login handles GET /api/v1/auth/login and redirects the browser to eSignet.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, models.ErrorCodeMethodNotAllowed, "Method not allowed")
		return
	}

	if !h.authConfigured() {
		utils.RespondWithError(
			w,
			http.StatusServiceUnavailable,
			models.ErrorCodeInternalError,
			"SLUDI login is not configured on the server (missing IDP_PRIVATE_KEY or IDP_TOKEN_ENDPOINT)",
		)
		return
	}

	returnTo := strings.TrimSpace(r.URL.Query().Get("return_to"))
	if returnTo == "" {
		returnTo = h.consentPortalURL
	}
	if !isAllowedReturnTo(returnTo, h.consentPortalURL) {
		utils.RespondWithError(w, http.StatusBadRequest, models.ErrorCodeBadRequest, "Invalid return_to URL")
		return
	}

	state, err := auth.GenerateRandomString(16)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, models.ErrorCodeInternalError, "Failed to start login")
		return
	}

	codeVerifier, err := auth.GenerateRandomString(32)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, models.ErrorCodeInternalError, "Failed to start login")
		return
	}

	nonce, err := auth.GenerateRandomString(16)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, models.ErrorCodeInternalError, "Failed to start login")
		return
	}

	h.loginStateStore.Save(state, codeVerifier, returnTo)

	authorizeURL, err := auth.BuildAuthorizeURL(h.esignetBaseURL, map[string]string{
		"client_id":             h.clientID,
		"redirect_uri":          h.callbackURL,
		"response_type":         "code",
		"scope":                 h.scope,
		"state":                 state,
		"nonce":                 nonce,
		"code_challenge":        auth.GenerateCodeChallenge(codeVerifier),
		"code_challenge_method": "S256",
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, models.ErrorCodeInternalError, "Failed to build authorize URL")
		return
	}

	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

// Callback handles GET /api/v1/auth/callback from eSignet and redirects back to the portal with tokens.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, models.ErrorCodeMethodNotAllowed, "Method not allowed")
		return
	}

	returnTo := h.consentPortalURL
	if oauthError := r.URL.Query().Get("error"); oauthError != "" {
		description := r.URL.Query().Get("error_description")
		if description == "" {
			description = oauthError
		}
		http.Redirect(w, r, buildPortalErrorRedirect(returnTo, description), http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Redirect(w, r, buildPortalErrorRedirect(returnTo, "Missing authorization code or state"), http.StatusFound)
		return
	}

	pending, ok := h.loginStateStore.Consume(state)
	if !ok {
		http.Redirect(w, r, buildPortalErrorRedirect(returnTo, "Invalid or expired login state. Please try again."), http.StatusFound)
		return
	}
	returnTo = pending.ReturnTo

	if !h.authConfigured() {
		http.Redirect(w, r, buildPortalErrorRedirect(returnTo, "SLUDI login is not configured on the server"), http.StatusFound)
		return
	}

	tokens, err := h.tokenExchanger.ExchangeAuthorizationCode(auth.TokenExchangeRequest{
		Code:         code,
		RedirectURI:  h.callbackURL,
		CodeVerifier: pending.CodeVerifier,
	})
	if err != nil {
		slog.Error("SLUDI token exchange failed", "error", err)
		http.Redirect(w, r, buildPortalErrorRedirect(returnTo, err.Error()), http.StatusFound)
		return
	}

	slog.Info("SLUDI token exchange succeeded",
		"has_id_token", tokens.IDToken != "",
		"expires_in", tokens.ExpiresIn,
	)

	sessionID, err := auth.GenerateRandomString(24)
	if err != nil {
		http.Redirect(w, r, buildPortalErrorRedirect(returnTo, "Failed to create login session"), http.StatusFound)
		return
	}

	h.tokenSessionStore.Save(sessionID, auth.TokenSession{
		AccessToken: tokens.AccessToken,
		IDToken:     tokens.IDToken,
		TokenType:   tokens.TokenType,
		ExpiresIn:   tokens.ExpiresIn,
	})

	http.Redirect(w, r, buildPortalSessionRedirect(returnTo, sessionID), http.StatusFound)
}

// FetchSession handles GET /api/v1/auth/session and returns tokens for a one-time session id.
func (h *AuthHandler) FetchSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, models.ErrorCodeMethodNotAllowed, "Method not allowed")
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("session"))
	if sessionID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, models.ErrorCodeBadRequest, "session is required")
		return
	}

	session, ok := h.tokenSessionStore.Get(sessionID)
	if !ok {
		utils.RespondWithError(w, http.StatusNotFound, models.ErrorCodeBadRequest, "Invalid or expired login session")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, auth.TokenExchangeResponse{
		AccessToken: session.AccessToken,
		IDToken:     session.IDToken,
		TokenType:   session.TokenType,
		ExpiresIn:   session.ExpiresIn,
	})
}

// ExchangeToken handles POST /api/v1/auth/token for direct portal-side code exchange.
func (h *AuthHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, models.ErrorCodeMethodNotAllowed, "Method not allowed")
		return
	}

	if h.tokenExchanger == nil {
		utils.RespondWithError(
			w,
			http.StatusServiceUnavailable,
			models.ErrorCodeInternalError,
			"SLUDI token exchange is not configured on the server (missing IDP_PRIVATE_KEY or IDP_TOKEN_ENDPOINT)",
		)
		return
	}

	var req auth.TokenExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, models.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	tokens, err := h.tokenExchanger.ExchangeAuthorizationCode(req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadGateway, models.ErrorCodeInternalError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, tokens)
}

func isAllowedReturnTo(returnTo, consentPortalURL string) bool {
	target, err := url.Parse(returnTo)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return false
	}

	allowed, err := url.Parse(consentPortalURL)
	if err != nil || allowed.Scheme == "" || allowed.Host == "" {
		return false
	}

	return strings.EqualFold(target.Scheme, allowed.Scheme) &&
		strings.EqualFold(target.Host, allowed.Host)
}

func buildPortalErrorRedirect(returnTo, message string) string {
	u, err := url.Parse(returnTo)
	if err != nil {
		u, _ = url.Parse("/")
	}
	q := url.Values{}
	q.Set("error", "login_failed")
	q.Set("error_description", message)
	u.RawQuery = q.Encode()
	return u.String()
}

func buildPortalSessionRedirect(returnTo, sessionID string) string {
	u, err := url.Parse(returnTo)
	if err != nil {
		u, _ = url.Parse("/")
	}
	q := u.Query()
	q.Set("auth_session", sessionID)
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String()
}
