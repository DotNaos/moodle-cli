package cli

import (
	"fmt"
	"time"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var loginSchool string
var loginUsername string
var loginPassword string
var loginHeadless bool = true
var loginShowBrowser bool
var loginTimeout time.Duration

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login via browser (SSO friendly)",
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

		fmt.Printf("session saved to %s\n", opts.SessionPath)
		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginSchool, "school", "", "School id (e.g. fhgr, phgr)")
	loginCmd.Flags().StringVar(&loginUsername, "username", "", "Username/email for login")
	loginCmd.Flags().StringVar(&loginPassword, "password", "", "Password for login")
	loginCmd.Flags().BoolVar(&loginShowBrowser, "show-browser", false, "Show browser window (non-headless)")
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", 120*time.Second, "Login timeout")
}
