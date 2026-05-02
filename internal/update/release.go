package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	ver "github.com/DotNaos/moodle-services/internal/version"
)

const (
	DefaultOwner         = "DotNaos"
	DefaultRepo          = "moodle-services"
	DefaultCheckInterval = 24 * time.Hour
)

var ErrNoStableRelease = errors.New("no stable release published yet")

type Client struct {
	Owner      string
	Repo       string
	BaseURL    string
	HTTPClient *http.Client
}

type Release struct {
	TagName string         `json:"tag_name"`
	Draft   bool           `json:"draft"`
	Assets  []ReleaseAsset `json:"assets"`
}

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Availability struct {
	CurrentVersion string
	LatestTag      string
	NeedsUpdate    bool
}

func NewClient() *Client {
	return &Client{
		Owner:   DefaultOwner,
		Repo:    DefaultRepo,
		BaseURL: "https://api.github.com",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) LatestRelease(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.latestReleaseURL(), nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "moodle-services update-check")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return Release{}, ErrNoStableRelease
		}
		return Release{}, fmt.Errorf("latest release check failed: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, err
	}
	if release.Draft || release.TagName == "" {
		return Release{}, fmt.Errorf("latest release response missing stable tag")
	}
	return release, nil
}

func (c *Client) Check(ctx context.Context, currentVersion string) (Availability, Release, error) {
	release, err := c.LatestRelease(ctx)
	if err != nil {
		return Availability{}, Release{}, err
	}

	availability := Availability{
		CurrentVersion: currentVersion,
		LatestTag:      release.TagName,
		NeedsUpdate:    needsUpdate(currentVersion, release.TagName),
	}
	return availability, release, nil
}

func needsUpdate(currentVersion string, latestTag string) bool {
	current := strings.TrimSpace(currentVersion)
	latest := strings.TrimSpace(latestTag)
	if latest == "" {
		return false
	}
	if strings.EqualFold(current, ver.DefaultVersion) || current == "" {
		return true
	}
	cmp, err := ver.Compare(current, latest)
	if err != nil {
		return !strings.EqualFold(normalizeTag(current), normalizeTag(latest))
	}
	return cmp < 0
}

func normalizeTag(tag string) string {
	trimmed := strings.TrimSpace(tag)
	if !strings.HasPrefix(trimmed, "v") {
		return "v" + trimmed
	}
	return trimmed
}

func ArchiveAssetName(goos string, goarch string) (string, error) {
	switch goos {
	case "darwin", "linux":
		return fmt.Sprintf("moodle_%s_%s.tar.gz", goos, goarch), nil
	case "windows":
		return fmt.Sprintf("moodle_%s_%s.zip", goos, goarch), nil
	default:
		return "", fmt.Errorf("unsupported OS for updates: %s", goos)
	}
}

func CurrentArchiveAssetName() (string, error) {
	return ArchiveAssetName(runtime.GOOS, runtime.GOARCH)
}

func FindAsset(release Release, name string) (ReleaseAsset, error) {
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return ReleaseAsset{}, fmt.Errorf("release asset not found: %s", name)
}

func ChecksumAsset(release Release) (ReleaseAsset, error) {
	return FindAsset(release, "checksums.txt")
}

func (c *Client) latestReleaseURL() string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	return fmt.Sprintf("%s/repos/%s/%s/releases/latest", base, c.Owner, c.Repo)
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}
