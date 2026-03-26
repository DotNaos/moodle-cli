package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestReleaseReturnsNoStableReleaseFor404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &Client{
		Owner:      "DotNaos",
		Repo:       "moodle-cli",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	_, err := client.LatestRelease(context.Background())
	if err != ErrNoStableRelease {
		t.Fatalf("expected ErrNoStableRelease, got %v", err)
	}
}
