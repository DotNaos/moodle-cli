package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/DotNaos/moodle-services/internal/moodle"
	"github.com/spf13/cobra"
)

var loginSchool string
var loginUsername string
var loginPassword string
var loginHeadless bool = true
var loginShowBrowser bool
var loginTimeout time.Duration

type loginCommandResult struct {
	Status      string `json:"status" yaml:"status"`
	SchoolID    string `json:"schoolId" yaml:"schoolId"`
	SessionPath string `json:"sessionPath" yaml:"sessionPath"`
	CreatedAt   string `json:"createdAt" yaml:"createdAt"`
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Login via browser and store a session",
	Long:    "Open a browser to log in with your Moodle username and password.\nThe session cookie is saved and reused for future commands.\n\nCredentials can be provided via flags, config, or environment variables:\n  MOODLE_USERNAME / MOODLE_PASSWORD\n  OS_STUDY_USERNAME / OS_STUDY_PASSWORD",
	Example: "  moodle login --username you@example.com --password \"secret\"\n  moodle login --show-browser",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginShowBrowser {
			loginHeadless = false
		}

		school, username, password, err := resolveLoginInputs(loginSchool, loginUsername, loginPassword)
		if err != nil {
			return err
		}

		result, err := moodle.LoginWithPlaywright(moodle.LoginOptions{
			SchoolID: school,
			Username: username,
			Password: password,
			Headless: loginHeadless,
			Timeout:  loginTimeout,
		})
		if err != nil {
			return err
		}

		payload := moodle.Session{SchoolID: result.SchoolID, Cookies: result.Cookies, CreatedAt: time.Now()}
		if err := moodle.SaveSession(opts.SessionPath, payload); err != nil {
			return err
		}

		output := loginCommandResult{
			Status:      "saved",
			SchoolID:    payload.SchoolID,
			SessionPath: opts.SessionPath,
			CreatedAt:   payload.CreatedAt.Format(time.RFC3339),
		}
		return writeCommandOutput(cmd, output, func(w io.Writer) error {
			if _, err := fmt.Fprintf(w, "session saved to %s\n", opts.SessionPath); err != nil {
				return err
			}
			if isDockerContainer() && (opts.SessionPath == "/data/session.json" || strings.HasPrefix(opts.SessionPath, "/data/")) {
				if _, err := fmt.Fprintln(w, "Mount /data to a host folder or named volume if you want separate 'docker run' commands to reuse this session."); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginSchool, "school", "", "School id override. Only fhgr is currently active; multi-school support is not active")
	loginCmd.Flags().StringVar(&loginUsername, "username", "", "Username/email for login")
	loginCmd.Flags().StringVar(&loginPassword, "password", "", "Password for login")
	loginCmd.Flags().BoolVar(&loginShowBrowser, "show-browser", false, "Show browser window (non-headless)")
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", 120*time.Second, "Login timeout")

	loginCmd.RegisterFlagCompletionFunc("school", completeSchoolIDs)
}
