package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/DotNaos/moodle-services/internal/config"
	"github.com/DotNaos/moodle-services/internal/moodle"
)

type sessionValidatingClient interface {
	ValidateSession() error
}

type courseDataClient interface {
	ValidateSession() error
	FetchCourses() ([]moodle.Course, error)
	FetchCourseResources(courseID string) ([]moodle.Resource, string, error)
}

var loginWithPlaywright = moodle.LoginWithPlaywright
var runtimeLoginOverrides loginInputOverrides

type loginInputOverrides struct {
	School   string
	Username string
	Password string
}

type savedSessionResult struct {
	Session moodle.Session
	Path    string
}

func ensureAuthenticatedClient() (*moodle.Client, error) {
	_, client, err := ensureValidatedSession(
		func() (moodle.Session, sessionValidatingClient, error) {
			session, client, err := loadSessionClient()
			if err != nil {
				return moodle.Session{}, nil, err
			}
			return session, client, nil
		},
		bootstrapSession,
		autoRelogin,
	)
	if err != nil {
		return nil, err
	}

	moodleClient, ok := client.(*moodle.Client)
	if !ok {
		return nil, fmt.Errorf("internal error: unexpected client type %T", client)
	}

	return moodleClient, nil
}

func ensureCourseDataClient() (courseDataClient, error) {
	client, err := ensureAuthenticatedClient()
	if err == nil {
		return client, nil
	}

	session, loadErr := moodle.LoadMobileSession(opts.MobileSessionPath)
	if loadErr != nil {
		return nil, err
	}
	mobileClient, mobileErr := moodle.NewMobileClient(session, session.ResolvedSchoolID())
	if mobileErr != nil {
		return nil, mobileErr
	}
	if validateErr := mobileClient.ValidateSession(); validateErr != nil {
		return nil, validateErr
	}
	return mobileClient, nil
}

func ensureAPIClient() (courseDataClient, error) {
	return ensureCourseDataClient()
}

func ensureServeSession() error {
	if runtimeLoginOverrides.any() {
		school, username, password, err := resolveLoginInputs("", "", "")
		if err != nil {
			return err
		}
		if username == "" || password == "" {
			return fmt.Errorf("serve login requires username and password. Provide them via flags, environment variables, or saved config")
		}
		_, err = loginAndSaveSession(school, username, password)
		return err
	}

	_, err := ensureCourseDataClient()
	return err
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

func ensureValidatedSession(load func() (moodle.Session, sessionValidatingClient, error), bootstrap func() error, relogin func(string) error) (moodle.Session, sessionValidatingClient, error) {
	session, client, err := load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return moodle.Session{}, nil, err
		}
		if err := bootstrap(); err != nil {
			return moodle.Session{}, nil, err
		}
		return load()
	}

	if err := client.ValidateSession(); err != nil {
		if !errors.Is(err, moodle.ErrSessionExpired) {
			return moodle.Session{}, nil, err
		}
		if err := relogin(session.SchoolID); err != nil {
			return moodle.Session{}, nil, err
		}
		session, client, err = load()
		if err != nil {
			return moodle.Session{}, nil, err
		}
		if err := client.ValidateSession(); err != nil {
			if errors.Is(err, moodle.ErrSessionExpired) {
				return moodle.Session{}, nil, fmt.Errorf("session expired, please run 'moodle login' again")
			}
			return moodle.Session{}, nil, err
		}
	}

	return session, client, nil
}

func bootstrapSession() error {
	school, username, password, err := resolveLoginInputs("", "", "")
	if err != nil {
		return err
	}
	if username == "" || password == "" {
		return missingSessionError()
	}
	_, err = loginAndSaveSession(school, username, password)
	return err
}

func autoRelogin(schoolID string) error {
	resolvedSchool, username, password, err := resolveLoginInputs(schoolID, "", "")
	if err != nil {
		return err
	}
	if username == "" || password == "" {
		return fmt.Errorf("session expired and auto-login requires stored credentials; run 'moodle config set --username <email> --password <password>' or 'moodle login --show-browser'")
	}

	_, err = loginAndSaveSession(resolvedSchool, username, password)
	return err
}

func loginAndSaveSession(school string, username string, password string) (savedSessionResult, error) {
	result, err := loginWithPlaywright(moodle.LoginOptions{
		SchoolID: school,
		Username: username,
		Password: password,
		Headless: true,
		Timeout:  loginTimeout,
	})
	if err != nil {
		return savedSessionResult{}, err
	}

	payload := moodle.Session{SchoolID: result.SchoolID, Cookies: result.Cookies, CreatedAt: time.Now()}
	if err := moodle.SaveSession(opts.SessionPath, payload); err != nil {
		return savedSessionResult{}, err
	}
	return savedSessionResult{
		Session: payload,
		Path:    opts.SessionPath,
	}, nil
}

func missingSessionError() error {
	msg := fmt.Sprintf("no saved Moodle session found at %s. Run 'moodle login' first or configure username and password so the CLI can sign in automatically", opts.SessionPath)
	if isDockerContainer() {
		msg += ". When using Docker, mount /data to a host folder or named volume if you want separate 'docker run' commands to reuse the same session"
	}
	return errors.New(msg)
}

func isDockerContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func (o loginInputOverrides) any() bool {
	return o.School != "" || o.Username != "" || o.Password != ""
}

func resolveLoginInputs(explicitSchool string, explicitUsername string, explicitPassword string) (string, string, string, error) {
	school := explicitSchool
	username := explicitUsername
	password := explicitPassword

	if school == "" && runtimeLoginOverrides.School != "" {
		school = runtimeLoginOverrides.School
	}
	if username == "" && runtimeLoginOverrides.Username != "" {
		username = runtimeLoginOverrides.Username
	}
	if password == "" && runtimeLoginOverrides.Password != "" {
		password = runtimeLoginOverrides.Password
	}

	if school == "" {
		school = os.Getenv("MOODLE_SCHOOL")
		if school == "" {
			school = os.Getenv("OS_STUDY_SCHOOL")
		}
	}
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

	if school == "" {
		school = moodle.GetDefaultSchool().ID
	}

	return school, username, password, nil
}
