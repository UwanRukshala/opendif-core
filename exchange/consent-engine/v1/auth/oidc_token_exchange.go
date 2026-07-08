package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenExchangeConfig holds settings for eSignet/SLUDI token exchange.
type TokenExchangeConfig struct {
	ClientID      string
	TokenEndpoint string
	PrivateKeyPEM string
}

// TokenExchangeRequest is sent by the consent portal after eSignet redirects back.
type TokenExchangeRequest struct {
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
}

// TokenExchangeResponse mirrors the OIDC token endpoint response fields used by the portal.
type TokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// TokenExchanger performs authorization-code exchange using private_key_jwt.
type TokenExchanger struct {
	config     TokenExchangeConfig
	privateKey *rsa.PrivateKey
	httpClient *http.Client
}

// NewTokenExchanger creates a token exchanger when private key and token endpoint are configured.
func NewTokenExchanger(config TokenExchangeConfig) (*TokenExchanger, error) {
	if config.ClientID == "" || config.TokenEndpoint == "" || config.PrivateKeyPEM == "" {
		return nil, fmt.Errorf("token exchange is not configured")
	}

	block, _ := pem.Decode([]byte(config.PrivateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid IDP private key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IDP private key: %w", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("IDP private key must be RSA")
	}

	return &TokenExchanger{
		config:     config,
		privateKey: rsaKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (t *TokenExchanger) buildClientAssertion() (string, error) {
	now := time.Now()
	jti, err := randomHex(16)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": t.config.ClientID,
		"sub": t.config.ClientID,
		"aud": t.config.TokenEndpoint,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"jti": jti,
	})

	return token.SignedString(t.privateKey)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

// ExchangeAuthorizationCode exchanges an authorization code for tokens at the eSignet token endpoint.
func (t *TokenExchanger) ExchangeAuthorizationCode(req TokenExchangeRequest) (*TokenExchangeResponse, error) {
	if req.Code == "" || req.RedirectURI == "" || req.CodeVerifier == "" {
		return nil, fmt.Errorf("code, redirect_uri, and code_verifier are required")
	}

	clientAssertion, err := t.buildClientAssertion()
	if err != nil {
		return nil, fmt.Errorf("failed to build client assertion: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", req.Code)
	form.Set("redirect_uri", req.RedirectURI)
	form.Set("client_id", t.config.ClientID)
	form.Set("code_verifier", req.CodeVerifier)
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", clientAssertion)

	httpReq, err := http.NewRequest(http.MethodPost, t.config.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenExchangeResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &tokenResp, nil
}
