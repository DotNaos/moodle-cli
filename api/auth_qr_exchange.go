package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

const internalWebSecretEnv = "MOODLE_WEB_INTERNAL_SECRET"

type qrExchangeInput struct {
	QR   string `json:"qr"`
	Name string `json:"name"`
}

func AuthQrExchange(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodPost) {
		return
	}
	if r.URL.Query().Get("clerk") == "1" {
		authClerkQRExchange(w, r)
		return
	}
	var input qrExchangeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	exchangeAndPersistQR(w, r, input, "")
}

func authClerkQRExchange(w http.ResponseWriter, r *http.Request) {
	expectedSecret := strings.TrimSpace(os.Getenv(internalWebSecretEnv))
	if expectedSecret == "" {
		svc.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": internalWebSecretEnv + " is not configured"})
		return
	}
	providedSecret := strings.TrimSpace(r.Header.Get("X-Moodle-Internal-Secret"))
	if !svc.ConstantTimeEqual(providedSecret, expectedSecret) {
		svc.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	clerkUserID := strings.TrimSpace(r.Header.Get("X-Clerk-User-Id"))
	if clerkUserID == "" {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing Clerk user id"})
		return
	}
	var input qrExchangeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	exchangeAndPersistQR(w, r, input, clerkUserID)
}

func exchangeAndPersistQR(w http.ResponseWriter, r *http.Request, input qrExchangeInput, clerkUserID string) {
	link, err := svc.ParseMobileQRLink(input.QR)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	token, err := svc.ExchangeMobileQRToken(link)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	session := svc.MobileSessionFromToken(token)
	session.SchoolID = svc.ActiveSchoolID
	client, err := svc.NewMobileClient(session, session.ResolvedSchoolID())
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	siteInfo, err := client.FetchMobileSiteInfo()
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	sessionData, err := json.Marshal(session)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	cfg := svc.LoadServerEnv()
	box, err := svc.EncryptionBox(cfg)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	encryptedSession, err := box.EncryptString(string(sessionData))
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	st, err := svc.OpenStoreFromEnv(cfg)
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	defer st.Close()
	displayName := strings.TrimSpace(siteInfo.UserName)
	if displayName == "" {
		displayName = strings.TrimSpace(input.Name)
	}
	user, err := st.UpsertMoodleAccount(r.Context(), svc.UpsertMoodleAccountInput{
		SiteURL:                    session.SiteURL,
		MoodleUserID:               session.UserID,
		DisplayName:                displayName,
		ClerkUserID:                clerkUserID,
		SchoolID:                   session.SchoolID,
		EncryptedMobileSessionJSON: encryptedSession,
	})
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	apiKey, err := svc.GenerateAPIKey()
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	record, err := st.CreateAPIKey(r.Context(), user.ID, "Initial API key", apiKey, cfg.HashSecret, []string{"moodle:read", "pdf:read", "calendar:read"})
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	svc.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "apiKey": apiKey, "apiKeyRecord": record})
}
