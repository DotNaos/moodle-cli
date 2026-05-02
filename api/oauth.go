package handler

import (
	"encoding/json"
	"errors"
	"net/http"
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
