package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/DotNaos/moodle-cli/internal/moodle"
)

func ensureAuthenticatedClient() (*moodle.Client, error) {
	session, client, err := loadSessionClient()
	if err != nil {
		return nil, err
	}

	if err := client.ValidateSession(); err != nil {
		if !errors.Is(err, moodle.ErrSessionExpired) {
			return nil, err
		}
		if err := autoRelogin(session.SchoolID); err != nil {
			return nil, err
		}
		_, client, err = loadSessionClient()
		if err != nil {
			return nil, err
		}
		if err := client.ValidateSession(); err != nil {
			if errors.Is(err, moodle.ErrSessionExpired) {
				return nil, fmt.Errorf("session expired, please run 'moodle login' again")
			}
			return nil, err
		}
	}

	return client, nil
}

func loadSessionClient() (moodle.Session, *moodle.Client, error) {
	session, err := moodle.LoadSession(opts.SessionPath)
	if err != nil {
		return moodle.Session{}, nil, fmt.Errorf("load session: %w", err)
	}
	client, err := moodle.NewClient(session)
	if err != nil {
		return moodle.Session{}, nil, err
	}
	return session, client, nil
}

func autoRelogin(schoolID string) error {
	resolvedSchool, username, password, err := resolveLoginInputs(schoolID, "", "")
	if err != nil {
		return err
	}
	if username == "" || password == "" {
		return fmt.Errorf("session expired and auto-login requires stored credentials; run 'moodle config set --username <email> --password <password>' or 'moodle login --show-browser'")
	}

	result, err := moodle.LoginWithPlaywright(moodle.LoginOptions{
		SchoolID: resolvedSchool,
		Username: username,
		Password: password,
		Headless: true,
		Timeout:  loginTimeout,
	})
	if err != nil {
		return err
	}

	payload := moodle.Session{SchoolID: result.SchoolID, Cookies: result.Cookies, CreatedAt: time.Now()}
	if err := moodle.SaveSession(opts.SessionPath, payload); err != nil {
		return err
	}
	return nil
}

func resolveLoginInputs(explicitSchool string, explicitUsername string, explicitPassword string) (string, string, string, error) {
	school := explicitSchool
	username := explicitUsername
	password := explicitPassword

	if username == "" {
		username = os.Getenv("MOODLE_USERNAME")
		if username == "" {
			username = os.Getenv("OS_STUDY_USERNAME")
		}
	}
	if password == "" {
		password = os.Getenv("MOODLE_PASSWORD")
		if password == "" {
			password = os.Getenv("OS_STUDY_PASSWORD")
		}
	}

	if username == "" || password == "" || school == "" {
		cfg, err := config.LoadConfig(opts.ConfigPath)
		if err != nil {
			return "", "", "", err
		}
		if school == "" && cfg.SchoolID != "" {
			school = cfg.SchoolID
		}
		if username == "" && cfg.Username != "" {
			username = cfg.Username
		}
		if password == "" && cfg.Password != "" {
			password = cfg.Password
		}
	}

	return school, username, password, nil
}
