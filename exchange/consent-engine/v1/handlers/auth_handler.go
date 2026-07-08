package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gov-dx-sandbox/exchange/consent-engine/v1/auth"
	"github.com/gov-dx-sandbox/exchange/consent-engine/v1/models"
	"github.com/gov-dx-sandbox/exchange/consent-engine/v1/utils"
)

// AuthHandler handles OIDC token exchange for the consent portal.
type AuthHandler struct {
	tokenExchanger *auth.TokenExchanger
}

// NewAuthHandler creates a new auth handler. tokenExchanger may be nil if not configured.
func NewAuthHandler(tokenExchanger *auth.TokenExchanger) *AuthHandler {
	return &AuthHandler{tokenExchanger: tokenExchanger}
}

// ExchangeToken handles POST /api/v1/auth/token
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
