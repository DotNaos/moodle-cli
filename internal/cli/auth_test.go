package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DotNaos/moodle-services/internal/moodle"
)

type fakeValidatingClient struct {
	validateErr error
}

func (f fakeValidatingClient) ValidateSession() error {
	return f.validateErr
}

func TestEnsureValidatedSessionBootstrapsWhenSessionIsMissing(t *testing.T) {
	loadCalls := 0
	bootstrapCalls := 0
	reloginCalls := 0

	session, client, err := ensureValidatedSession(
		func() (moodle.Session, sessionValidatingClient, error) {
			loadCalls++
			if loadCalls == 1 {
				return moodle.Session{}, nil, os.ErrNotExist
			}
			return moodle.Session{SchoolID: "fhgr"}, fakeValidatingClient{}, nil
		},
		func() error {
			bootstrapCalls++
			return nil
		},
		func(string) error {
			reloginCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected bootstrap to recover missing session, got %v", err)
	}
	if session.SchoolID != "fhgr" {
		t.Fatalf("unexpected session %+v", session)
	}
	if _, ok := client.(fakeValidatingClient); !ok {
		t.Fatalf("unexpected client type %T", client)
	}
	if loadCalls != 2 {
		t.Fatalf("expected two load attempts, got %d", loadCalls)
	}
	if bootstrapCalls != 1 {
		t.Fatalf("expected one bootstrap attempt, got %d", bootstrapCalls)
	}
	if reloginCalls != 0 {
		t.Fatalf("expected no relogin attempt, got %d", reloginCalls)
	}
}

func TestEnsureValidatedSessionReloginsWhenSessionExpired(t *testing.T) {
	loadCalls := 0
	bootstrapCalls := 0
	reloginCalls := 0
	reloginSchool := ""

	session, client, err := ensureValidatedSession(
		func() (moodle.Session, sessionValidatingClient, error) {
			loadCalls++
			if loadCalls == 1 {
				return moodle.Session{SchoolID: "fhgr"}, fakeValidatingClient{validateErr: moodle.ErrSessionExpired}, nil
			}
			return moodle.Session{SchoolID: "fhgr"}, fakeValidatingClient{}, nil
		},
		func() error {
			bootstrapCalls++
			return nil
		},
		func(schoolID string) error {
			reloginCalls++
			reloginSchool = schoolID
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected relogin to recover expired session, got %v", err)
	}
	if session.SchoolID != "fhgr" {
		t.Fatalf("unexpected session %+v", session)
	}
	if _, ok := client.(fakeValidatingClient); !ok {
		t.Fatalf("unexpected client type %T", client)
	}
	if loadCalls != 2 {
		t.Fatalf("expected two load attempts, got %d", loadCalls)
	}
	if bootstrapCalls != 0 {
		t.Fatalf("expected no bootstrap attempt, got %d", bootstrapCalls)
	}
	if reloginCalls != 1 {
		t.Fatalf("expected one relogin attempt, got %d", reloginCalls)
	}
	if reloginSchool != "fhgr" {
		t.Fatalf("expected relogin for fhgr, got %q", reloginSchool)
	}
}

func TestResolveLoginInputsUsesRuntimeOverrides(t *testing.T) {
	originalOverrides := runtimeLoginOverrides
	runtimeLoginOverrides = loginInputOverrides{
		School:   "fhgr",
		Username: "runtime-user",
		Password: "runtime-pass",
	}
	t.Cleanup(func() {
		runtimeLoginOverrides = originalOverrides
	})

	school, username, password, err := resolveLoginInputs("", "", "")
	if err != nil {
		t.Fatalf("expected runtime overrides to resolve, got %v", err)
	}
	if school != "fhgr" || username != "runtime-user" || password != "runtime-pass" {
		t.Fatalf("unexpected resolved inputs: %q %q %q", school, username, password)
	}
}

func TestResolveLoginInputsUsesSchoolEnvironment(t *testing.T) {
	originalOverrides := runtimeLoginOverrides
	runtimeLoginOverrides = loginInputOverrides{}
	t.Cleanup(func() {
		runtimeLoginOverrides = originalOverrides
	})

	t.Setenv("MOODLE_SCHOOL", "fhgr")
	t.Setenv("MOODLE_USERNAME", "env-user")
	t.Setenv("MOODLE_PASSWORD", "env-pass")

	school, username, password, err := resolveLoginInputs("", "", "")
	if err != nil {
		t.Fatalf("expected environment inputs to resolve, got %v", err)
	}
	if school != "fhgr" || username != "env-user" || password != "env-pass" {
		t.Fatalf("unexpected resolved inputs: %q %q %q", school, username, password)
	}
}

func TestResolveLoginInputsDefaultsToActiveSchool(t *testing.T) {
	originalOverrides := runtimeLoginOverrides
	originalConfigPath := opts.ConfigPath
	runtimeLoginOverrides = loginInputOverrides{}
	tempDir := t.TempDir()
	opts.ConfigPath = filepath.Join(tempDir, "config.json")
	t.Cleanup(func() {
		runtimeLoginOverrides = originalOverrides
		opts.ConfigPath = originalConfigPath
	})

	t.Setenv("MOODLE_SCHOOL", "")
	t.Setenv("OS_STUDY_SCHOOL", "")
	t.Setenv("MOODLE_USERNAME", "env-user")
	t.Setenv("MOODLE_PASSWORD", "env-pass")

	school, username, password, err := resolveLoginInputs("", "", "")
	if err != nil {
		t.Fatalf("expected default school to resolve, got %v", err)
	}
	if school != moodle.ActiveSchoolID || username != "env-user" || password != "env-pass" {
		t.Fatalf("unexpected resolved inputs: %q %q %q", school, username, password)
	}
}

func TestEnsureServeSessionPerformsFreshLoginWithRuntimeOverrides(t *testing.T) {
	originalSessionPath := opts.SessionPath
	originalConfigPath := opts.ConfigPath
	originalLoginTimeout := loginTimeout
	originalLoginWithPlaywright := loginWithPlaywright
	originalOverrides := runtimeLoginOverrides

	tempDir := t.TempDir()
	opts.SessionPath = filepath.Join(tempDir, "session.json")
	opts.ConfigPath = filepath.Join(tempDir, "config.json")
	loginTimeout = 3 * time.Second
	runtimeLoginOverrides = loginInputOverrides{
		School:   "fhgr",
		Username: "fresh-user",
		Password: "fresh-pass",
	}
	loginWithPlaywright = func(options moodle.LoginOptions) (moodle.LoginResult, error) {
		if options.SchoolID != "fhgr" || options.Username != "fresh-user" || options.Password != "fresh-pass" {
			t.Fatalf("unexpected login options: %+v", options)
		}
		return moodle.LoginResult{SchoolID: options.SchoolID, Cookies: "cookie=value"}, nil
	}

	t.Cleanup(func() {
		opts.SessionPath = originalSessionPath
		opts.ConfigPath = originalConfigPath
		loginTimeout = originalLoginTimeout
		loginWithPlaywright = originalLoginWithPlaywright
		runtimeLoginOverrides = originalOverrides
	})

	if err := ensureServeSession(); err != nil {
		t.Fatalf("expected serve session bootstrap to succeed, got %v", err)
	}

	session, err := moodle.LoadSession(opts.SessionPath)
	if err != nil {
		t.Fatalf("expected session to be saved, got %v", err)
	}
	if session.SchoolID != "fhgr" || session.Cookies != "cookie=value" {
		t.Fatalf("unexpected saved session: %+v", session)
	}
}
