package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	latestReleaseEndpoint = "https://api.github.com/repos/KageRyo/netquota/releases/latest"
	installerAssetName    = "netquota-windows-amd64-setup.exe"
	maxResponseBytes      = 1 << 20
)

var ErrInvalidVersion = errors.New("invalid semantic version")

type Checker struct {
	Client   *http.Client
	Endpoint string
}

type Release struct {
	TagName      string
	PageURL      string
	InstallerURL string
}

type apiRelease struct {
	TagName string     `json:"tag_name"`
	HTMLURL string     `json:"html_url"`
	Assets  []apiAsset `json:"assets"`
}

type apiAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
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
		Endpoint: latestReleaseEndpoint,
	}
}

func (c Checker) Check(ctx context.Context, current string) (Release, bool, error) {
	currentVersion, err := parseVersion(current)
	if err != nil {
		return Release{}, false, fmt.Errorf("current version: %w", err)
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = latestReleaseEndpoint
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
		return Release{}, false, fmt.Errorf("request latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Release{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, false, fmt.Errorf("latest release returned HTTP %d", resp.StatusCode)
	}

	var payload apiRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&payload); err != nil {
		return Release{}, false, fmt.Errorf("decode latest release: %w", err)
	}
	latestVersion, err := parseVersion(payload.TagName)
	if err != nil {
		return Release{}, false, fmt.Errorf("latest release tag: %w", err)
	}
	if compareVersions(latestVersion, currentVersion) <= 0 {
		return Release{}, false, nil
	}

	release := Release{TagName: payload.TagName, PageURL: payload.HTMLURL}
	for _, asset := range payload.Assets {
		if asset.Name == installerAssetName {
			release.InstallerURL = asset.DownloadURL
			break
		}
	}
	return release, true, nil
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
