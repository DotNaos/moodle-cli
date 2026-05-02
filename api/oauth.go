package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

const (
	oauthCodePrefix         = "moodle_code_"
	oauthRefreshTokenPrefix = "moodle_refresh_"
	oauthClientIDPrefix     = "moodle_client_"
	oauthDefaultScope       = "moodle:read pdf:read calendar:read offline_access"
	oauthAccessTokenTTL     = time.Hour
	oauthCodeTTL            = 10 * time.Minute
	oauthRefreshTokenTTL    = 90 * 24 * time.Hour
)

func Oauth(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("route") {
	case "protected-resource":
		oauthProtectedResource(w, r)
	case "authorization-server":
		oauthAuthorizationServer(w, r)
	case "register":
		oauthRegister(w, r)
	case "authorize":
		oauthAuthorize(w, r)
	case "authorize-complete":
		oauthAuthorizeComplete(w, r)
	case "token":
		oauthToken(w, r)
	default:
		svc.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "OAuth route not found"})
	}
}

func oauthProtectedResource(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodGet) {
		return
	}
	baseURL := oauthBaseURL(r)
	svc.WriteJSON(w, http.StatusOK, map[string]any{
		"resource":                 oauthResource(r),
		"authorization_servers":    []string{baseURL},
		"scopes_supported":         oauthScopes(),
		"resource_documentation":   baseURL + "/api/docs",
		"bearer_methods_supported": []string{"header"},
	})
}

func oauthAuthorizationServer(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodGet) {
		return
	}
	baseURL := oauthBaseURL(r)
	svc.WriteJSON(w, http.StatusOK, map[string]any{
		"issuer":                                baseURL,
		"authorization_endpoint":                baseURL + "/oauth/authorize",
		"token_endpoint":                        baseURL + "/oauth/token",
		"registration_endpoint":                 baseURL + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      oauthScopes(),
	})
}

func oauthRegister(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodPost) {
		return
	}
	cfg := svc.LoadServerEnv()
	store, err := svc.OpenStoreFromEnv(cfg)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	defer store.Close()

	var input struct {
		ClientName              string   `json:"client_name"`
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		Scope                   string   `json:"scope"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if len(input.RedirectURIs) == 0 {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "redirect_uris is required"})
		return
	}
	if err := validateRedirectURIs(input.RedirectURIs); err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	grantTypes := defaultStrings(input.GrantTypes, []string{"authorization_code", "refresh_token"})
	responseTypes := defaultStrings(input.ResponseTypes, []string{"code"})
	scope := strings.TrimSpace(input.Scope)
	if scope == "" {
		scope = oauthDefaultScope
	}
	clientID, err := randomToken(oauthClientIDPrefix)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	client, err := store.CreateOAuthClient(r.Context(), svc.CreateOAuthClientInput{
		ClientID:      clientID,
		ClientName:    strings.TrimSpace(input.ClientName),
		RedirectURIs:  input.RedirectURIs,
		GrantTypes:    grantTypes,
		ResponseTypes: responseTypes,
		Scope:         scope,
	})
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	svc.WriteJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ClientID,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"client_name":                client.ClientName,
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                client.GrantTypes,
		"response_types":             client.ResponseTypes,
		"scope":                      client.Scope,
		"token_endpoint_auth_method": "none",
	})
}

func oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodGet) {
		return
	}
	if _, err := validateAuthorizeQuery(r); err != nil {
		redirectOAuthError(w, r, "invalid_request", err.Error())
		return
	}
	webURL := strings.TrimRight(strings.TrimSpace(os.Getenv("MOODLE_WEB_PUBLIC_URL")), "/")
	if webURL == "" {
		webURL = "https://moodle.os-home.net"
	}
	target := webURL + "/oauth/authorize"
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func oauthAuthorizeComplete(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodPost) {
		return
	}
	expectedSecret := strings.TrimSpace(os.Getenv("MOODLE_WEB_INTERNAL_SECRET"))
	if expectedSecret == "" {
		svc.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "MOODLE_WEB_INTERNAL_SECRET is not configured"})
		return
	}
	if !svc.ConstantTimeEqual(strings.TrimSpace(r.Header.Get("X-Moodle-Internal-Secret")), expectedSecret) {
		svc.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	clerkUserID := strings.TrimSpace(r.Header.Get("X-Clerk-User-Id"))
	if clerkUserID == "" {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing Clerk user id"})
		return
	}

	var input authorizeCompleteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	requestURL, err := authorizeRequestURL(r, input)
	if err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	params, err := validateAuthorizeQuery(requestURL)
	if err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cfg := svc.LoadServerEnv()
	store, err := svc.OpenStoreFromEnv(cfg)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	defer store.Close()
	client, err := store.OAuthClient(r.Context(), params.ClientID)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	if !contains(client.RedirectURIs, params.RedirectURI) {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "redirect_uri is not registered for this client"})
		return
	}
	user, err := store.UserForClerkID(r.Context(), clerkUserID)
	if errors.Is(err, svc.ErrNotFound) {
		svc.WriteJSON(w, http.StatusConflict, map[string]string{"error": "Connect Moodle before authorizing ChatGPT."})
		return
	}
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	if _, err := store.MoodleCredentialsForUserID(r.Context(), user.ID); err != nil {
		svc.WriteJSON(w, http.StatusConflict, map[string]string{"error": "Connect Moodle before authorizing ChatGPT."})
		return
	}
	code, err := randomToken(oauthCodePrefix)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	scope := strings.TrimSpace(params.Scope)
	if scope == "" {
		scope = oauthDefaultScope
	}
	err = store.CreateOAuthAuthorizationCode(r.Context(), svc.CreateOAuthAuthorizationCodeInput{
		Code:                code,
		ClientID:            params.ClientID,
		UserID:              user.ID,
		RedirectURI:         params.RedirectURI,
		CodeChallenge:       params.CodeChallenge,
		CodeChallengeMethod: params.CodeChallengeMethod,
		Resource:            defaultResource(params.Resource, r),
		Scope:               scope,
		ExpiresAt:           time.Now().Add(oauthCodeTTL),
		HashSecret:          cfg.HashSecret,
	})
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	redirectURL, err := callbackURL(params.RedirectURI, code, params.State)
	if err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	svc.WriteJSON(w, http.StatusOK, map[string]string{"redirectUrl": redirectURL})
}

func oauthToken(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodPost) {
		return
	}
	if err := r.ParseForm(); err != nil {
		oauthTokenError(w, "invalid_request", "invalid form body", http.StatusBadRequest)
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		oauthAuthorizationCodeToken(w, r)
	case "refresh_token":
		oauthRefreshToken(w, r)
	default:
		oauthTokenError(w, "unsupported_grant_type", "unsupported grant_type", http.StatusBadRequest)
	}
}

func oauthAuthorizationCodeToken(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.Form.Get("code"))
	clientID := strings.TrimSpace(r.Form.Get("client_id"))
	redirectURI := strings.TrimSpace(r.Form.Get("redirect_uri"))
	codeVerifier := strings.TrimSpace(r.Form.Get("code_verifier"))
	if code == "" || clientID == "" || redirectURI == "" || codeVerifier == "" {
		oauthTokenError(w, "invalid_request", "code, client_id, redirect_uri, and code_verifier are required", http.StatusBadRequest)
		return
	}
	cfg := svc.LoadServerEnv()
	store, err := svc.OpenStoreFromEnv(cfg)
	if err != nil {
		oauthTokenError(w, "server_error", err.Error(), http.StatusInternalServerError)
		return
	}
	defer store.Close()
	codeRecord, err := store.ConsumeOAuthAuthorizationCode(r.Context(), code, cfg.HashSecret)
	if err != nil {
		oauthTokenError(w, "invalid_grant", "authorization code is invalid or expired", http.StatusBadRequest)
		return
	}
	if codeRecord.ClientID != clientID || codeRecord.RedirectURI != redirectURI {
		oauthTokenError(w, "invalid_grant", "authorization code does not match this client", http.StatusBadRequest)
		return
	}
	if !verifyPKCE(codeVerifier, codeRecord.CodeChallenge) {
		oauthTokenError(w, "invalid_grant", "PKCE verification failed", http.StatusBadRequest)
		return
	}
	writeIssuedTokens(w, r, store, cfg, codeRecord.UserID, codeRecord.ClientID, codeRecord.Resource, codeRecord.Scope)
}

func oauthRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := strings.TrimSpace(r.Form.Get("refresh_token"))
	clientID := strings.TrimSpace(r.Form.Get("client_id"))
	if refreshToken == "" || clientID == "" {
		oauthTokenError(w, "invalid_request", "refresh_token and client_id are required", http.StatusBadRequest)
		return
	}
	cfg := svc.LoadServerEnv()
	store, err := svc.OpenStoreFromEnv(cfg)
	if err != nil {
		oauthTokenError(w, "server_error", err.Error(), http.StatusInternalServerError)
		return
	}
	defer store.Close()
	refreshRecord, err := store.OAuthRefreshToken(r.Context(), refreshToken, cfg.HashSecret)
	if err != nil {
		oauthTokenError(w, "invalid_grant", "refresh token is invalid or expired", http.StatusBadRequest)
		return
	}
	if refreshRecord.ClientID != clientID {
		oauthTokenError(w, "invalid_grant", "refresh token does not match this client", http.StatusBadRequest)
		return
	}
	_ = store.RevokeOAuthRefreshToken(r.Context(), refreshToken, cfg.HashSecret)
	writeIssuedTokens(w, r, store, cfg, refreshRecord.UserID, refreshRecord.ClientID, refreshRecord.Resource, refreshRecord.Scope)
}

func writeIssuedTokens(w http.ResponseWriter, r *http.Request, store *svc.Store, cfg svc.ServerEnv, userID string, clientID string, resource string, scope string) {
	accessToken, err := randomToken(svc.OAuthAccessTokenPrefix)
	if err != nil {
		oauthTokenError(w, "server_error", err.Error(), http.StatusInternalServerError)
		return
	}
	refreshToken, err := randomToken(oauthRefreshTokenPrefix)
	if err != nil {
		oauthTokenError(w, "server_error", err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	if err := store.CreateOAuthAccessToken(r.Context(), svc.CreateOAuthTokenInput{
		Token: accessToken, UserID: userID, ClientID: clientID, Resource: resource, Scope: scope,
		ExpiresAt: now.Add(oauthAccessTokenTTL), HashSecret: cfg.HashSecret,
	}); err != nil {
		oauthTokenError(w, "server_error", err.Error(), http.StatusInternalServerError)
		return
	}
	if err := store.CreateOAuthRefreshToken(r.Context(), svc.CreateOAuthTokenInput{
		Token: refreshToken, UserID: userID, ClientID: clientID, Resource: resource, Scope: scope,
		ExpiresAt: now.Add(oauthRefreshTokenTTL), HashSecret: cfg.HashSecret,
	}); err != nil {
		oauthTokenError(w, "server_error", err.Error(), http.StatusInternalServerError)
		return
	}
	svc.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(oauthAccessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"scope":         scope,
	})
}

type authorizeCompleteInput struct {
	ResponseType        string `json:"response_type"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	Resource            string `json:"resource"`
}

type authorizeParams struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
}

func validateAuthorizeQuery(r *http.Request) (authorizeParams, error) {
	query := r.URL.Query()
	params := authorizeParams{
		ResponseType:        strings.TrimSpace(query.Get("response_type")),
		ClientID:            strings.TrimSpace(query.Get("client_id")),
		RedirectURI:         strings.TrimSpace(query.Get("redirect_uri")),
		Scope:               strings.TrimSpace(query.Get("scope")),
		State:               strings.TrimSpace(query.Get("state")),
		CodeChallenge:       strings.TrimSpace(query.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(query.Get("code_challenge_method")),
		Resource:            strings.TrimSpace(query.Get("resource")),
	}
	if params.ResponseType != "code" {
		return params, fmt.Errorf("response_type must be code")
	}
	if params.ClientID == "" || params.RedirectURI == "" || params.CodeChallenge == "" {
		return params, fmt.Errorf("client_id, redirect_uri, and code_challenge are required")
	}
	if params.CodeChallengeMethod != "S256" {
		return params, fmt.Errorf("code_challenge_method must be S256")
	}
	if _, err := url.ParseRequestURI(params.RedirectURI); err != nil {
		return params, fmt.Errorf("redirect_uri is invalid")
	}
	return params, nil
}

func authorizeRequestURL(r *http.Request, input authorizeCompleteInput) (*http.Request, error) {
	values := url.Values{}
	values.Set("response_type", input.ResponseType)
	values.Set("client_id", input.ClientID)
	values.Set("redirect_uri", input.RedirectURI)
	values.Set("code_challenge", input.CodeChallenge)
	values.Set("code_challenge_method", input.CodeChallengeMethod)
	if input.Scope != "" {
		values.Set("scope", input.Scope)
	}
	if input.State != "" {
		values.Set("state", input.State)
	}
	if input.Resource != "" {
		values.Set("resource", input.Resource)
	}
	copyRequest := r.Clone(r.Context())
	copyRequest.URL = &url.URL{Scheme: "https", Host: r.Host, Path: "/oauth/authorize", RawQuery: values.Encode()}
	return copyRequest, nil
}

func redirectOAuthError(w http.ResponseWriter, r *http.Request, code string, description string) {
	redirectURI := strings.TrimSpace(r.URL.Query().Get("redirect_uri"))
	if redirectURI == "" {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": description})
		return
	}
	target, err := url.Parse(redirectURI)
	if err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": description})
		return
	}
	values := target.Query()
	values.Set("error", code)
	values.Set("error_description", description)
	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		values.Set("state", state)
	}
	target.RawQuery = values.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func callbackURL(redirectURI string, code string, state string) (string, error) {
	target, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	values := target.Query()
	values.Set("code", code)
	if state != "" {
		values.Set("state", state)
	}
	target.RawQuery = values.Encode()
	return target.String(), nil
}

func validateRedirectURIs(redirectURIs []string) error {
	for _, redirectURI := range redirectURIs {
		parsed, err := url.ParseRequestURI(strings.TrimSpace(redirectURI))
		if err != nil {
			return fmt.Errorf("redirect_uri is invalid")
		}
		if parsed.Scheme != "https" {
			return fmt.Errorf("redirect_uri must use https")
		}
	}
	return nil
}

func verifyPKCE(verifier string, challenge string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return svc.ConstantTimeEqual(computed, challenge)
}

func oauthTokenError(w http.ResponseWriter, code string, description string, status int) {
	svc.WriteJSON(w, status, map[string]string{"error": code, "error_description": description})
}

func randomToken(prefix string) (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(data), nil
}

func defaultStrings(values []string, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}

func oauthScopes() []string {
	return []string{"moodle:read", "pdf:read", "calendar:read", "offline_access"}
}

func defaultResource(resource string, r *http.Request) string {
	if strings.TrimSpace(resource) != "" {
		return strings.TrimSpace(resource)
	}
	return oauthResource(r)
}

func oauthResource(r *http.Request) string {
	return oauthBaseURL(r) + "/api/mcp"
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
