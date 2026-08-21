package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckReportsNewerReleaseAndPlatformDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Accept"), "application/vnd.github+json"; got != want {
			t.Fatalf("Accept header = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
            {
                "tag_name": "v0.2.0",
                "html_url": "https://github.com/KageRyo/netquota/releases/tag/v0.2.0",
                "assets": [
                    {"name": "netquota-windows-amd64-setup.exe", "browser_download_url": "https://example.test/setup.exe"},
                    {"name": "netquota-linux-amd64.tar.gz", "browser_download_url": "https://example.test/netquota.tar.gz"},
                    {"name": "SHA256SUMS", "browser_download_url": "https://example.test/SHA256SUMS"}
                ]
            }
        ]`))
	}))
	defer server.Close()

	release, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !available {
		t.Fatal("expected an available update")
	}
	wantURL := ""
	switch platformAssetName(runtime.GOOS, runtime.GOARCH) {
	case windowsInstallerAsset:
		wantURL = "https://example.test/setup.exe"
	case linuxPackageAsset:
		wantURL = "https://example.test/netquota.tar.gz"
	}
	if release.TagName != "v0.2.0" || release.DownloadURL != wantURL {
		t.Fatalf("release = %+v, want download URL %q", release, wantURL)
	}
	if release.AssetName != platformAssetName(runtime.GOOS, runtime.GOARCH) {
		t.Fatalf("release asset name = %q, want %q", release.AssetName, platformAssetName(runtime.GOOS, runtime.GOARCH))
	}
	if release.ChecksumURL != "https://example.test/SHA256SUMS" {
		t.Fatalf("release checksum URL = %q, want the checksum asset URL", release.ChecksumURL)
	}
}

func TestChecksumForAsset(t *testing.T) {
	want := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, err := checksumForAsset(want+"  "+linuxPackageAsset+"\n", linuxPackageAsset)
	if err != nil {
		t.Fatalf("checksumForAsset: %v", err)
	}
	if got != want {
		t.Fatalf("checksumForAsset = %q, want %q", got, want)
	}
}

func TestCheckerDownloadRequiresChecksumManifest(t *testing.T) {
	_, err := (Checker{}).Download(context.Background(), Release{
		AssetName:   linuxPackageAsset,
		DownloadURL: "https://github.com/KageRyo/netquota/releases/download/v0.1.0/netquota-linux-amd64.tar.gz",
	}, t.TempDir(), nil)
	if err != ErrUnverifiedDownload {
		t.Fatalf("Download error = %v, want %v", err, ErrUnverifiedDownload)
	}
}

func TestDownloadVerifiedWritesAndReportsProgress(t *testing.T) {
	payload := []byte("verified NetQuota update")
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update.tar.gz" {
			t.Fatalf("request path = %q, want /update.tar.gz", r.URL.Path)
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	destination := t.TempDir()
	var lastWritten, lastTotal int64
	path, err := downloadVerified(
		context.Background(),
		server.Client(),
		server.URL+"/update.tar.gz",
		linuxPackageAsset,
		digest,
		int64(len(payload)),
		destination,
		func(written, total int64) {
			lastWritten, lastTotal = written, total
		},
	)
	if err != nil {
		t.Fatalf("downloadVerified: %v", err)
	}
	defer os.Remove(path)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded update: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("downloaded payload = %q, want %q", got, payload)
	}
	if lastWritten != int64(len(payload)) || lastTotal != int64(len(payload)) {
		t.Fatalf("last progress = (%d, %d), want (%d, %d)", lastWritten, lastTotal, len(payload), len(payload))
	}
	if filepath.Dir(path) != destination {
		t.Fatalf("download path = %q, want it below %q", path, destination)
	}
}

func TestDownloadVerifiedRemovesChecksumMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("tampered"))
	}))
	defer server.Close()

	destination := t.TempDir()
	_, err := downloadVerified(
		context.Background(),
		server.Client(),
		server.URL,
		linuxPackageAsset,
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		0,
		destination,
		nil,
	)
	if err == nil {
		t.Fatal("downloadVerified succeeded for a checksum mismatch")
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatalf("read download directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("download directory contains %d files after checksum failure", len(entries))
	}
}

func TestValidateReleaseURL(t *testing.T) {
	for _, url := range []string{
		"https://github.com/KageRyo/netquota/releases/download/v0.1.0/netquota.tar.gz",
		"https://release-assets.githubusercontent.com/example",
	} {
		if err := validateReleaseURL(url); err != nil {
			t.Errorf("validateReleaseURL(%q): %v", url, err)
		}
	}
	for _, url := range []string{
		"http://github.com/KageRyo/netquota/releases/download/v0.1.0/netquota.tar.gz",
		"https://example.com/netquota.tar.gz",
		"https://github.com@example.com/netquota.tar.gz",
	} {
		if err := validateReleaseURL(url); err == nil {
			t.Errorf("validateReleaseURL(%q) succeeded, want an error", url)
		}
	}
}

func TestPlatformAssetName(t *testing.T) {
	tests := []struct {
		name string
		goos string
		arch string
		want string
	}{
		{name: "Windows amd64", goos: "windows", arch: "amd64", want: windowsInstallerAsset},
		{name: "Linux amd64", goos: "linux", arch: "amd64", want: linuxPackageAsset},
		{name: "Linux arm64", goos: "linux", arch: "arm64"},
		{name: "Darwin amd64", goos: "darwin", arch: "amd64"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := platformAssetName(test.goos, test.arch); got != test.want {
				t.Fatalf("platformAssetName(%q, %q) = %q, want %q", test.goos, test.arch, got, test.want)
			}
		})
	}
}

func TestCheckReportsStableReleaseForAlphaBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"v0.1.0","html_url":"https://example.test/release"}]`))
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
		_, _ = w.Write([]byte(`[{"tag_name":"v0.1.0","html_url":"https://example.test/release"}]`))
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

func TestCheckIgnoresMalformedReleaseTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"nightly","html_url":"https://example.test/release"}]`))
	}))
	defer server.Close()

	_, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if available {
		t.Fatal("malformed release tags must not be offered as updates")
	}
}

func TestCheckFindsNewerPrereleaseForAlphaBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
            {"tag_name":"v0.1.0-alpha.2","html_url":"https://example.test/alpha.2","prerelease":true},
            {"tag_name":"v0.1.0-beta.1","html_url":"https://example.test/beta.1","prerelease":true}
        ]`))
	}))
	defer server.Close()

	release, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "v0.1.0-alpha.1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !available || release.TagName != "v0.1.0-beta.1" {
		t.Fatalf("release = %+v, available = %v", release, available)
	}
}

func TestCheckDoesNotOfferPrereleaseToStableBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
            {"tag_name":"v0.2.0-alpha.1","html_url":"https://example.test/alpha.1","prerelease":true}
        ]`))
	}))
	defer server.Close()

	_, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "v0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if available {
		t.Fatal("stable builds must not receive prerelease updates")
	}
}

func TestCheckIgnoresDraftReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
            {"tag_name":"v0.2.0","html_url":"https://example.test/draft","draft":true}
        ]`))
	}))
	defer server.Close()

	_, available, err := (Checker{Client: server.Client(), Endpoint: server.URL}).Check(context.Background(), "v0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if available {
		t.Fatal("draft releases must not be offered as updates")
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
