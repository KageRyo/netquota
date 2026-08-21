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

func TestCheckReportsStableReleaseForAlphaBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.0","html_url":"https://example.test/release"}`))
	}))
	defer server.Close()

	_, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "v0.1.0-alpha.1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !available {
		t.Fatal("expected the stable release to be newer than alpha")
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

func TestParseVersionFollowsSemanticVersioningPrecedence(t *testing.T) {
	versions := []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
	}
	for index := 0; index < len(versions)-1; index++ {
		left, err := parseVersion(versions[index])
		if err != nil {
			t.Fatalf("parse %q: %v", versions[index], err)
		}
		right, err := parseVersion(versions[index+1])
		if err != nil {
			t.Fatalf("parse %q: %v", versions[index+1], err)
		}
		if got := compareVersions(left, right); got >= 0 {
			t.Fatalf("compare %q and %q = %d, want a negative value", versions[index], versions[index+1], got)
		}
	}
}

func TestParseVersionRejectsInvalidSemanticVersions(t *testing.T) {
	for _, value := range []string{
		"1.02.0",
		"1.0.0-alpha.01",
		"1.0.0-",
		"1.0",
		"1.0.0+",
		"1.0.0-alpha..1",
	} {
		if _, err := parseVersion(value); err == nil {
			t.Fatalf("parse %q succeeded, want an error", value)
		}
	}
}

func TestParseVersionIgnoresBuildMetadataForPrecedence(t *testing.T) {
	left, err := parseVersion("1.0.0-alpha.1+build.1")
	if err != nil {
		t.Fatalf("parse left: %v", err)
	}
	right, err := parseVersion("1.0.0-alpha.1+build.2")
	if err != nil {
		t.Fatalf("parse right: %v", err)
	}
	if got := compareVersions(left, right); got != 0 {
		t.Fatalf("compare versions with different build metadata = %d, want 0", got)
	}
}
