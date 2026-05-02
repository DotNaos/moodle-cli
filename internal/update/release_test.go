package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArchiveAssetName(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{goos: "darwin", goarch: "arm64", want: "moodle_darwin_arm64.tar.gz"},
		{goos: "linux", goarch: "amd64", want: "moodle_linux_amd64.tar.gz"},
		{goos: "windows", goarch: "amd64", want: "moodle_windows_amd64.zip"},
	}

	for _, tt := range tests {
		got, err := ArchiveAssetName(tt.goos, tt.goarch)
		if err != nil {
			t.Fatalf("unexpected error for %s/%s: %v", tt.goos, tt.goarch, err)
		}
		if got != tt.want {
			t.Fatalf("expected %q, got %q", tt.want, got)
		}
	}
}

func TestArchiveAssetNameRejectsUnsupportedOS(t *testing.T) {
	if _, err := ArchiveAssetName("plan9", "amd64"); err == nil {
		t.Fatal("expected unsupported OS error")
	}
}

func TestFindAsset(t *testing.T) {
	release := Release{
		Assets: []ReleaseAsset{{Name: "checksums.txt"}, {Name: "moodle_linux_amd64.tar.gz"}},
	}

	asset, err := FindAsset(release, "moodle_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset.Name != "moodle_linux_amd64.tar.gz" {
		t.Fatalf("unexpected asset: %+v", asset)
	}
}

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "dev build", current: "dev", latest: "v1.2.3", want: true},
		{name: "older stable", current: "v1.2.2", latest: "v1.2.3", want: true},
		{name: "same stable", current: "v1.2.3", latest: "v1.2.3", want: false},
		{name: "newer current", current: "v1.3.0", latest: "v1.2.3", want: false},
	}

	for _, tt := range tests {
		if got := needsUpdate(tt.current, tt.latest); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/DotNaos/moodle-services/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v1.2.3","assets":[{"name":"checksums.txt","browser_download_url":"https://example.com/checksums.txt"}]}`))
	}))
	defer server.Close()

	client := &Client{
		Owner:      "DotNaos",
		Repo:       "moodle-services",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	release, err := client.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("expected tag v1.2.3, got %q", release.TagName)
	}
}
