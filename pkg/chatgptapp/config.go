package chatgptapp

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/DotNaos/moodle-services/internal/moodle"
)

const (
	envDatabaseURL       = "DATABASE_URL"
	envAPIKeyHash        = "MCP_API_KEY_HASH"
	envCalendarURL       = "MOODLE_CALENDAR_URL"
	envMobileSessionJSON = "MOODLE_MOBILE_SESSION_JSON"
	envMobileToken       = "MOODLE_MOBILE_TOKEN"
	envMoodleSiteURL     = "MOODLE_SITE_URL"
	envMoodleUserID      = "MOODLE_USER_ID"
	envMoodleSchoolID    = "MOODLE_SCHOOL_ID"
	envSessionJSON       = "MOODLE_SESSION_JSON"
)

type Config struct {
	DatabaseURL string
	APIKeyHash  string
	CalendarURL string
}

func LoadConfigFromEnv() (Config, error) {
	return Config{
		DatabaseURL: strings.TrimSpace(os.Getenv(envDatabaseURL)),
		APIKeyHash:  strings.TrimSpace(os.Getenv(envAPIKeyHash)),
		CalendarURL: strings.TrimSpace(os.Getenv(envCalendarURL)),
	}, nil
}

func ClientFromMobileSessionJSON(raw string) (DataClient, error) {
	var session moodle.MobileSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, fmt.Errorf("decode mobile session: %w", err)
	}
	return moodle.NewMobileClient(session, session.ResolvedSchoolID())
}

func ClientFromEnv() (DataClient, error) {
	if raw := strings.TrimSpace(os.Getenv(envMobileSessionJSON)); raw != "" {
		return ClientFromMobileSessionJSON(raw)
	}

	if token := strings.TrimSpace(os.Getenv(envMobileToken)); token != "" {
		siteURL := strings.TrimSpace(os.Getenv(envMoodleSiteURL))
		userID, err := strconv.Atoi(strings.TrimSpace(os.Getenv(envMoodleUserID)))
		if err != nil || siteURL == "" {
			return nil, fmt.Errorf("%s requires %s and numeric %s", envMobileToken, envMoodleSiteURL, envMoodleUserID)
		}
		session := moodle.MobileSession{
			SchoolID: strings.TrimSpace(os.Getenv(envMoodleSchoolID)),
			SiteURL:  siteURL,
			UserID:   userID,
			Token:    token,
		}
		return moodle.NewMobileClient(session, session.ResolvedSchoolID())
	}

	if raw := strings.TrimSpace(os.Getenv(envSessionJSON)); raw != "" {
		var session moodle.Session
		if err := json.Unmarshal([]byte(raw), &session); err != nil {
			return nil, fmt.Errorf("decode %s: %w", envSessionJSON, err)
		}
		return moodle.NewClient(session)
	}

	return nil, fmt.Errorf("configure %s, or %s plus %s/%s, or %s", envMobileSessionJSON, envMobileToken, envMoodleSiteURL, envMoodleUserID, envSessionJSON)
}
