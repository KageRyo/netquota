package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckReportsNewerReleaseAndInstaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Accept"), "application/vnd.github+json"; got != want {
			t.Fatalf("Accept header = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "tag_name": "v0.2.0",
            "html_url": "https://github.com/KageRyo/netquota/releases/tag/v0.2.0",
            "assets": [{"name": "netquota-windows-amd64-setup.exe", "browser_download_url": "https://example.test/setup.exe"}]
        }`))
	}))
	defer server.Close()

	release, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !available {
		t.Fatal("expected an available update")
	}
	if release.TagName != "v0.2.0" || release.InstallerURL != "https://example.test/setup.exe" {
		t.Fatalf("release = %+v", release)
	}
}

func TestCheckIgnoresOlderRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.0","html_url":"https://example.test/release"}`))
	}))
	defer server.Close()

	_, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "v0.1.1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if available {
		t.Fatal("did not expect an available update")
	}
}

func TestCheckTreatsNoPublishedReleaseAsUpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if available {
		t.Fatal("did not expect an available update")
	}
}

func TestCheckRejectsMalformedReleaseTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"nightly","html_url":"https://example.test/release"}`))
	}))
	defer server.Close()

	_, _, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "0.1.0")
	if err == nil {
		t.Fatal("expected malformed release tag error")
	}
}
