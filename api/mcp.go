package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/DotNaos/moodle-services/pkg/chatgptapp"
	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	cfg, err := chatgptapp.LoadConfigFromEnv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	service, apiKey, status, err := serviceFromRequest(r, cfg)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	chatgptapp.Handler{Service: service, APIKey: apiKey}.ServeHTTP(w, r)
}

func serviceFromRequest(r *http.Request, cfg chatgptapp.Config) (chatgptapp.Service, string, int, error) {
	apiKey := chatgptapp.APIKeyFromRequest(r)
	if cfg.DatabaseURL == "" {
		if cfg.APIKeyHash != "" {
			if apiKey == "" || !chatgptapp.ConstantTimeEqual(chatgptapp.HashAPIKey(apiKey), cfg.APIKeyHash) {
				return chatgptapp.Service{}, "", http.StatusUnauthorized, chatgptapp.ErrUnauthorized
			}
		}
		client, err := chatgptapp.ClientFromEnv()
		if err != nil {
			return chatgptapp.Service{}, "", http.StatusInternalServerError, err
		}
		return chatgptapp.Service{Client: client, CalendarURL: cfg.CalendarURL}, apiKey, http.StatusOK, nil
	}

	if apiKey == "" {
		return chatgptapp.Service{}, "", http.StatusUnauthorized, chatgptapp.ErrUnauthorized
	}

	if strings.TrimSpace(os.Getenv("APP_ENCRYPTION_KEY")) != "" {
		service, status, err := serviceFromNewStore(r, cfg.DatabaseURL, apiKey)
		if err == nil || status != http.StatusInternalServerError {
			return service, apiKey, status, err
		}
	}

	store, err := chatgptapp.OpenStore(cfg.DatabaseURL)
	if err != nil {
		return chatgptapp.Service{}, "", http.StatusInternalServerError, err
	}
	defer store.Close()

	credentials, err := store.CredentialsForAPIKey(r.Context(), apiKey)
	if errors.Is(err, chatgptapp.ErrUnauthorized) {
		return chatgptapp.Service{}, "", http.StatusUnauthorized, err
	}
	if err != nil {
		return chatgptapp.Service{}, "", http.StatusInternalServerError, err
	}

	client, err := chatgptapp.ClientFromMobileSessionJSON(credentials.MobileSessionJSON)
	if err != nil {
		return chatgptapp.Service{}, "", http.StatusInternalServerError, err
	}
	return chatgptapp.Service{
		Client:      client,
		CalendarURL: credentials.CalendarURL,
	}, apiKey, http.StatusOK, nil
}

func serviceFromNewStore(r *http.Request, databaseURL string, apiKey string) (chatgptapp.Service, int, error) {
	store, err := svc.OpenStore(databaseURL)
	if err != nil {
		return chatgptapp.Service{}, http.StatusInternalServerError, err
	}
	defer store.Close()
	hashSecret := strings.TrimSpace(os.Getenv("API_KEY_HASH_SECRET"))
	if hashSecret == "" {
		hashSecret = strings.TrimSpace(os.Getenv("APP_ENCRYPTION_KEY"))
	}
	credentials, err := store.MoodleCredentialsForAPIKey(r.Context(), apiKey, []byte(hashSecret))
	if errors.Is(err, svc.ErrUnauthorized) {
		return chatgptapp.Service{}, http.StatusUnauthorized, chatgptapp.ErrUnauthorized
	}
	if err != nil {
		return chatgptapp.Service{}, http.StatusInternalServerError, err
	}
	box, err := svc.NewBox(os.Getenv("APP_ENCRYPTION_KEY"))
	if err != nil {
		return chatgptapp.Service{}, http.StatusInternalServerError, err
	}
	sessionJSON, err := box.DecryptString(credentials.EncryptedMobileSessionJSON)
	if err != nil {
		return chatgptapp.Service{}, http.StatusInternalServerError, err
	}
	var session svc.MobileSession
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		return chatgptapp.Service{}, http.StatusInternalServerError, err
	}
	client, err := svc.NewMobileClient(session, session.ResolvedSchoolID())
	if err != nil {
		return chatgptapp.Service{}, http.StatusInternalServerError, err
	}
	calendarURL := strings.TrimSpace(os.Getenv("MOODLE_CALENDAR_URL"))
	if credentials.EncryptedCalendarURL != "" {
		if decrypted, err := box.DecryptString(credentials.EncryptedCalendarURL); err == nil {
			calendarURL = decrypted
		}
	}
	return chatgptapp.Service{Client: client, CalendarURL: calendarURL}, http.StatusOK, nil
}
