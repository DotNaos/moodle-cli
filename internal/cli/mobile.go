package cli

import (
	"fmt"
	"io"

	"github.com/DotNaos/moodle-services/internal/moodle"
	"github.com/spf13/cobra"
)

type mobileQRInspectResult struct {
	Kind                 string `json:"kind" yaml:"kind"`
	SiteURL              string `json:"siteUrl" yaml:"siteUrl"`
	UserID               int    `json:"userId,omitempty" yaml:"userId,omitempty"`
	QRLoginKeyRedacted   string `json:"qrLoginKeyRedacted,omitempty" yaml:"qrLoginKeyRedacted,omitempty"`
	PublicConfigEndpoint string `json:"publicConfigEndpoint" yaml:"publicConfigEndpoint"`
	TokenEndpoint        string `json:"tokenEndpoint,omitempty" yaml:"tokenEndpoint,omitempty"`
	TokenWSFunction      string `json:"tokenWsFunction,omitempty" yaml:"tokenWsFunction,omitempty"`
	SampleTokenRequest   string `json:"sampleTokenRequest,omitempty" yaml:"sampleTokenRequest,omitempty"`
	SafetyNote           string `json:"safetyNote" yaml:"safetyNote"`
}

type mobileQRLoginResult struct {
	Status                string                `json:"status" yaml:"status"`
	SiteURL               string                `json:"siteUrl" yaml:"siteUrl"`
	UserID                int                   `json:"userId" yaml:"userId"`
	MobileSessionPath     string                `json:"mobileSessionPath,omitempty" yaml:"mobileSessionPath,omitempty"`
	QRLoginKeyRedacted    string                `json:"qrLoginKeyRedacted" yaml:"qrLoginKeyRedacted"`
	TokenReceived         bool                  `json:"tokenReceived" yaml:"tokenReceived"`
	PrivateTokenReceived  bool                  `json:"privateTokenReceived" yaml:"privateTokenReceived"`
	TokenRedacted         string                `json:"tokenRedacted,omitempty" yaml:"tokenRedacted,omitempty"`
	PrivateTokenRedacted  string                `json:"privateTokenRedacted,omitempty" yaml:"privateTokenRedacted,omitempty"`
	SiteName              string                `json:"siteName,omitempty" yaml:"siteName,omitempty"`
	Username              string                `json:"username,omitempty" yaml:"username,omitempty"`
	MobileFunctionChecked string                `json:"mobileFunctionChecked,omitempty" yaml:"mobileFunctionChecked,omitempty"`
	CourseCount           int                   `json:"courseCount,omitempty" yaml:"courseCount,omitempty"`
	SampleCourses         []mobileCourseSummary `json:"sampleCourses,omitempty" yaml:"sampleCourses,omitempty"`
	SafetyNote            string                `json:"safetyNote" yaml:"safetyNote"`
}

type mobileCourseSummary struct {
	ID        int    `json:"id" yaml:"id"`
	FullName  string `json:"fullname" yaml:"fullname"`
	ShortName string `json:"shortname" yaml:"shortname"`
}

var mobileQRLoginSkipCheck bool

var mobileCmd = &cobra.Command{
	Use:   "mobile",
	Short: "Inspect Moodle mobile app links",
	Long:  "Inspect Moodle mobile app links and explain which Moodle Mobile web services they use.",
}

var mobileQRCmd = &cobra.Command{
	Use:   "qr",
	Short: "Inspect Moodle mobile QR login links",
}

var mobileQRInspectCmd = &cobra.Command{
	Use:   "inspect <moodlemobile-link>",
	Short: "Explain a Moodle mobile QR login link without redeeming it",
	Long:  "Explain a Moodle mobile QR login link without redeeming it. This does not contact Moodle or consume the one-time QR login key.",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := inspectMobileQRLink(args[0])
		if err != nil {
			return err
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return renderMobileQRInspectText(w, result)
		})
	},
}

var mobileQRLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Create a Moodle mobile token from a freshly scraped profile QR code",
	Long:  "Load your Moodle profile with the saved web session, decode the mobile app QR code, exchange it for a Moodle mobile token, and verify it with read-only mobile API calls.",
	Args:  cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runMobileQRLogin(!mobileQRLoginSkipCheck)
		if err != nil {
			return err
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return renderMobileQRLoginText(w, result)
		})
	},
}

func init() {
	mobileQRLoginCmd.Flags().BoolVar(&mobileQRLoginSkipCheck, "skip-check", false, "Create the mobile token but skip read-only API verification")
	mobileQRCmd.AddCommand(mobileQRInspectCmd)
	mobileQRCmd.AddCommand(mobileQRLoginCmd)
	mobileCmd.AddCommand(mobileQRCmd)
}

func inspectMobileQRLink(raw string) (mobileQRInspectResult, error) {
	link, err := moodle.ParseMobileQRLink(raw)
	if err != nil {
		return mobileQRInspectResult{}, err
	}

	result := mobileQRInspectResult{
		Kind:                 "site-url",
		SiteURL:              link.SiteURL,
		UserID:               link.UserID,
		PublicConfigEndpoint: link.PublicConfigEndpoint(),
		SafetyNote:           "This command only inspects the link. It does not redeem the QR login key or create a Moodle session.",
	}

	if link.IsAutoLogin {
		result.Kind = "qr-auto-login"
		result.QRLoginKeyRedacted = moodle.RedactSecret(link.QRLoginKey)
		result.TokenEndpoint = link.MobileTokenEndpoint()
		result.TokenWSFunction = "tool_mobile_get_tokens_for_qr_login"
		result.SampleTokenRequest = buildQRTokenRequest(link)
	}

	return result, nil
}

func runMobileQRLogin(checkAPI bool) (mobileQRLoginResult, error) {
	client, err := ensureAuthenticatedClient()
	if err != nil {
		return mobileQRLoginResult{}, err
	}
	link, err := client.FetchMobileQRLink()
	if err != nil {
		return mobileQRLoginResult{}, err
	}
	token, err := client.ExchangeMobileQRToken(link)
	if err != nil {
		return mobileQRLoginResult{}, err
	}
	session := moodle.MobileSessionFromToken(token)
	session.SchoolID = client.School.ID
	if err := moodle.SaveMobileSession(opts.MobileSessionPath, session); err != nil {
		return mobileQRLoginResult{}, err
	}

	result := mobileQRLoginResult{
		Status:               "created",
		SiteURL:              token.SiteURL,
		UserID:               token.UserID,
		MobileSessionPath:    opts.MobileSessionPath,
		QRLoginKeyRedacted:   moodle.RedactSecret(token.QRLoginKey),
		TokenReceived:        token.Token != "",
		PrivateTokenReceived: token.PrivateToken != "",
		TokenRedacted:        moodle.RedactSecret(token.Token),
		PrivateTokenRedacted: moodle.RedactSecret(token.PrivateToken),
		SafetyNote:           "The real mobile token was saved locally and is not printed in full. Treat the mobile session file like a password.",
	}

	if !checkAPI {
		return result, nil
	}

	info, err := client.FetchMobileSiteInfo(token)
	if err != nil {
		return mobileQRLoginResult{}, err
	}
	result.SiteName = info.SiteName
	result.Username = info.UserName
	result.MobileFunctionChecked = "core_webservice_get_site_info"

	courses, err := client.FetchMobileUserCourses(token)
	if err != nil {
		return mobileQRLoginResult{}, err
	}
	result.CourseCount = len(courses)
	for i, course := range courses {
		if i >= 5 {
			break
		}
		result.SampleCourses = append(result.SampleCourses, mobileCourseSummary{
			ID:        course.ID,
			FullName:  course.FullName,
			ShortName: course.ShortName,
		})
	}

	return result, nil
}

func buildQRTokenRequest(link moodle.MobileQRLink) string {
	return fmt.Sprintf(
		"POST %s\n[{\"index\":0,\"methodname\":\"tool_mobile_get_tokens_for_qr_login\",\"args\":{\"qrloginkey\":\"%s\",\"userid\":\"%d\"}}]",
		link.MobileTokenEndpoint(),
		moodle.RedactSecret(link.QRLoginKey),
		link.UserID,
	)
}

func renderMobileQRInspectText(w io.Writer, result mobileQRInspectResult) error {
	if _, err := fmt.Fprintf(w, "site: %s\n", result.SiteURL); err != nil {
		return err
	}
	if result.UserID != 0 {
		if _, err := fmt.Fprintf(w, "user id: %d\n", result.UserID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "type: %s\n", result.Kind); err != nil {
		return err
	}
	if result.QRLoginKeyRedacted != "" {
		if _, err := fmt.Fprintf(w, "qr login key: %s\n", result.QRLoginKeyRedacted); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "public config check: %s with methodname=tool_mobile_get_public_config\n", result.PublicConfigEndpoint); err != nil {
		return err
	}
	if result.TokenEndpoint != "" {
		if _, err := fmt.Fprintf(w, "token exchange: %s with methodname=%s\n", result.TokenEndpoint, result.TokenWSFunction); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "note: %s\n", result.SafetyNote)
	return err
}

func renderMobileQRLoginText(w io.Writer, result mobileQRLoginResult) error {
	if _, err := fmt.Fprintf(w, "status: %s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "site: %s\n", result.SiteURL); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "user id: %d\n", result.UserID); err != nil {
		return err
	}
	if result.MobileSessionPath != "" {
		if _, err := fmt.Fprintf(w, "mobile session: %s\n", result.MobileSessionPath); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "qr login key: %s\n", result.QRLoginKeyRedacted); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "mobile token received: %t\n", result.TokenReceived); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "private token received: %t\n", result.PrivateTokenReceived); err != nil {
		return err
	}
	if result.SiteName != "" {
		if _, err := fmt.Fprintf(w, "site name: %s\n", result.SiteName); err != nil {
			return err
		}
	}
	if result.Username != "" {
		if _, err := fmt.Fprintf(w, "username: %s\n", result.Username); err != nil {
			return err
		}
	}
	if result.CourseCount > 0 {
		if _, err := fmt.Fprintf(w, "courses visible through mobile API: %d\n", result.CourseCount); err != nil {
			return err
		}
	}
	for _, course := range result.SampleCourses {
		if _, err := fmt.Fprintf(w, "- %s (%s, %d)\n", course.FullName, course.ShortName, course.ID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "note: %s\n", result.SafetyNote)
	return err
}
