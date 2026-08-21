package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	publicReleasesEndpoint = "https://api.github.com/repos/KageRyo/netquota/releases"
	windowsInstallerAsset  = "netquota-windows-amd64-setup.exe"
	linuxPackageAsset      = "netquota-linux-amd64.tar.gz"
	checksumAsset          = "SHA256SUMS"
	maxResponseBytes       = 1 << 20
	maxDownloadBytes       = 512 << 20
	downloadTimeout        = 10 * time.Minute
)

var (
	ErrInvalidVersion      = errors.New("invalid semantic version")
	ErrUnverifiedDownload  = errors.New("release does not provide a verifiable checksum")
	ErrUnsupportedPlatform = errors.New("automatic updates are unsupported on this platform")
)

type ProgressFunc func(written, total int64)

type Checker struct {
	Client   *http.Client
	Endpoint string
}

type Release struct {
	TagName     string
	PageURL     string
	AssetName   string
	DownloadURL string
	ChecksumURL string
	AssetSize   int64
}

type apiRelease struct {
	TagName    string     `json:"tag_name"`
	HTMLURL    string     `json:"html_url"`
	Draft      bool       `json:"draft"`
	Prerelease bool       `json:"prerelease"`
	Assets     []apiAsset `json:"assets"`
}

type apiAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

type semanticVersion struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease []string
}

func NewChecker() Checker {
	return Checker{
		Client:   &http.Client{Timeout: 5 * time.Second},
		Endpoint: publicReleasesEndpoint,
	}
}

func (c Checker) Check(ctx context.Context, current string) (Release, bool, error) {
	currentVersion, err := parseVersion(current)
	if err != nil {
		return Release{}, false, fmt.Errorf("current version: %w", err)
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = publicReleasesEndpoint
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, false, fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "NetQuota-Update-Checker")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, false, fmt.Errorf("request releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Release{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, false, fmt.Errorf("releases returned HTTP %d", resp.StatusCode)
	}

	var payload []apiRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&payload); err != nil {
		return Release{}, false, fmt.Errorf("decode releases: %w", err)
	}

	var selected apiRelease
	var selectedVersion semanticVersion
	found := false
	for _, candidate := range payload {
		if candidate.Draft {
			continue
		}
		candidateVersion, err := parseVersion(candidate.TagName)
		if err != nil {
			continue
		}
		candidateIsPrerelease := candidate.Prerelease || len(candidateVersion.PreRelease) > 0
		if len(currentVersion.PreRelease) == 0 && candidateIsPrerelease {
			continue
		}
		if compareVersions(candidateVersion, currentVersion) <= 0 {
			continue
		}
		if !found || compareVersions(candidateVersion, selectedVersion) > 0 {
			selected = candidate
			selectedVersion = candidateVersion
			found = true
		}
	}
	if !found {
		return Release{}, false, nil
	}

	release := Release{TagName: selected.TagName, PageURL: selected.HTMLURL}
	assetName := platformAssetName(runtime.GOOS, runtime.GOARCH)
	if assetName != "" {
		for _, asset := range selected.Assets {
			if asset.Name == assetName {
				release.AssetName = asset.Name
				release.DownloadURL = asset.DownloadURL
				release.AssetSize = asset.Size
			}
			if asset.Name == checksumAsset {
				release.ChecksumURL = asset.DownloadURL
			}
		}
	}
	return release, true, nil
}

func (c Checker) Download(ctx context.Context, release Release, destinationDir string, progress ProgressFunc) (string, error) {
	if release.AssetName == "" || release.DownloadURL == "" {
		return "", errors.New("download update: release has no compatible asset")
	}
	if release.ChecksumURL == "" {
		return "", ErrUnverifiedDownload
	}
	if err := validateReleaseURL(release.DownloadURL); err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	if err := validateReleaseURL(release.ChecksumURL); err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}

	client := c.downloadClient()
	checksum, err := fetchChecksum(ctx, client, release.ChecksumURL, release.AssetName)
	if err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}
	path, err := downloadVerified(ctx, client, release.DownloadURL, release.AssetName, checksum, release.AssetSize, destinationDir, progress)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	return path, nil
}

func (c Checker) downloadClient() *http.Client {
	base := c.Client
	if base == nil {
		base = &http.Client{}
	}
	client := *base
	client.Timeout = downloadTimeout
	redirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateReleaseURL(req.URL.String()); err != nil {
			return err
		}
		if redirect != nil {
			return redirect(req, via)
		}
		return nil
	}
	return &client
}

func validateReleaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "https" || parsed.User != nil {
		return fmt.Errorf("unexpected release URL %q", rawURL)
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host != "github.com" && host != "api.github.com" && !strings.HasSuffix(host, ".githubusercontent.com") {
		return fmt.Errorf("unexpected release host %q", parsed.Hostname())
	}
	return nil
}

func fetchChecksum(ctx context.Context, client *http.Client, checksumURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("create checksum request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", "NetQuota-Update-Checker")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read checksum: %w", err)
	}
	if int64(len(body)) > maxResponseBytes {
		return "", errors.New("checksum is too large")
	}
	return checksumForAsset(string(body), assetName)
}

func checksumForAsset(manifest, assetName string) (string, error) {
	var checksum string
	for _, line := range strings.Split(manifest, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.TrimPrefix(fields[1], "*") != assetName {
			continue
		}
		candidate := strings.ToLower(fields[0])
		if len(candidate) != 64 || !isHex(candidate) {
			return "", fmt.Errorf("invalid checksum for %s", assetName)
		}
		if checksum != "" && checksum != candidate {
			return "", fmt.Errorf("conflicting checksums for %s", assetName)
		}
		checksum = candidate
	}
	if checksum == "" {
		return "", fmt.Errorf("checksum for %s was not found", assetName)
	}
	return checksum, nil
}

func isHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') &&
			(character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

func downloadVerified(ctx context.Context, client *http.Client, downloadURL, assetName, expectedChecksum string, expectedSize int64, destinationDir string, progress ProgressFunc) (path string, err error) {
	if filepath.Base(assetName) != assetName || assetName == "" {
		return "", fmt.Errorf("invalid release asset name %q", assetName)
	}
	if expectedSize < 0 || expectedSize > maxDownloadBytes {
		return "", errors.New("release asset is too large")
	}
	if err := os.MkdirAll(destinationDir, 0o700); err != nil {
		return "", fmt.Errorf("create download directory: %w", err)
	}
	temporary, err := os.CreateTemp(destinationDir, ".netquota-update-*"+filepath.Ext(assetName))
	if err != nil {
		return "", fmt.Errorf("create downloaded file: %w", err)
	}
	path = temporary.Name()
	temporaryPath := path
	keepFile := false
	defer func() {
		if closeErr := temporary.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close downloaded file: %w", closeErr)
		}
		if !keepFile {
			_ = os.Remove(temporaryPath)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "NetQuota-Update-Checker")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("asset returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxDownloadBytes || (expectedSize > 0 && resp.ContentLength >= 0 && resp.ContentLength != expectedSize) {
		return "", errors.New("downloaded asset size does not match the release metadata")
	}

	hasher := sha256.New()
	total := expectedSize
	if total == 0 && resp.ContentLength >= 0 {
		total = resp.ContentLength
	}
	writer := &progressWriter{writer: io.MultiWriter(temporary, hasher), total: total, callback: progress}
	written, err := io.Copy(writer, io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return "", fmt.Errorf("write asset: %w", err)
	}
	if written > maxDownloadBytes || (expectedSize > 0 && written != expectedSize) {
		return "", errors.New("downloaded asset size does not match the release metadata")
	}
	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return "", fmt.Errorf("checksum mismatch: got %s", actualChecksum)
	}
	if err := temporary.Chmod(0o700); err != nil {
		return "", fmt.Errorf("set downloaded file permissions: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("flush downloaded file: %w", err)
	}
	keepFile = true
	return path, nil
}

type progressWriter struct {
	writer   io.Writer
	total    int64
	written  int64
	callback ProgressFunc
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.written += int64(n)
	if n > 0 && w.callback != nil {
		w.callback(w.written, w.total)
	}
	return n, err
}

func platformAssetName(goos, goarch string) string {
	switch {
	case goos == "windows" && goarch == "amd64":
		return windowsInstallerAsset
	case goos == "linux" && goarch == "amd64":
		return linuxPackageAsset
	default:
		return ""
	}
}

func parseVersion(value string) (semanticVersion, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if value == "" {
		return semanticVersion{}, ErrInvalidVersion
	}

	versionAndBuild := strings.Split(value, "+")
	if len(versionAndBuild) > 2 {
		return semanticVersion{}, ErrInvalidVersion
	}
	if len(versionAndBuild) == 2 && !validIdentifiers(versionAndBuild[1], true) {
		return semanticVersion{}, ErrInvalidVersion
	}

	version := versionAndBuild[0]
	preRelease := []string(nil)
	if separator := strings.IndexByte(version, '-'); separator >= 0 {
		preRelease = strings.Split(version[separator+1:], ".")
		if !validIdentifiers(strings.Join(preRelease, "."), false) {
			return semanticVersion{}, ErrInvalidVersion
		}
		version = version[:separator]
	}

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return semanticVersion{}, ErrInvalidVersion
	}
	values := [3]int{}
	for index, part := range parts {
		if !validCoreNumber(part) {
			return semanticVersion{}, ErrInvalidVersion
		}
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return semanticVersion{}, ErrInvalidVersion
		}
		values[index] = parsed
	}
	return semanticVersion{
		Major:      values[0],
		Minor:      values[1],
		Patch:      values[2],
		PreRelease: preRelease,
	}, nil
}

func compareVersions(left, right semanticVersion) int {
	if left.Major != right.Major {
		return compareInts(left.Major, right.Major)
	}
	if left.Minor != right.Minor {
		return compareInts(left.Minor, right.Minor)
	}
	if left.Patch != right.Patch {
		return compareInts(left.Patch, right.Patch)
	}
	return comparePreRelease(left.PreRelease, right.PreRelease)
}

func validCoreNumber(value string) bool {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func validIdentifiers(value string, allowNumericLeadingZero bool) bool {
	if value == "" {
		return false
	}
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		numeric := true
		for index := 0; index < len(identifier); index++ {
			character := identifier[index]
			if character < '0' || character > '9' {
				numeric = false
			}
			if (character < '0' || character > '9') &&
				(character < 'A' || character > 'Z') &&
				(character < 'a' || character > 'z') && character != '-' {
				return false
			}
		}
		if numeric && !allowNumericLeadingZero && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func comparePreRelease(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}

	for index := 0; index < len(left) && index < len(right); index++ {
		leftNumeric := isNumericIdentifier(left[index])
		rightNumeric := isNumericIdentifier(right[index])
		if leftNumeric && rightNumeric {
			if len(left[index]) != len(right[index]) {
				return compareInts(len(left[index]), len(right[index]))
			}
			if left[index] != right[index] {
				if left[index] < right[index] {
					return -1
				}
				return 1
			}
			continue
		}
		if leftNumeric != rightNumeric {
			if leftNumeric {
				return -1
			}
			return 1
		}
		if left[index] != right[index] {
			if left[index] < right[index] {
				return -1
			}
			return 1
		}
	}
	return compareInts(len(left), len(right))
}

func isNumericIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func compareInts(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
