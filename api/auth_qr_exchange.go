package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

func AuthQrExchange(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodPost) {
		return
	}
	var input struct {
		QR   string `json:"qr"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		svc.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
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
