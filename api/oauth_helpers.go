package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

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
