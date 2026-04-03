package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// githubRepo is the owner/repo for the GitHub Releases API.
	githubRepo = "rundops/dops"

	// cacheTTL controls how often we re-check for updates.
	cacheTTL = 1 * time.Hour

	// httpTimeout prevents blocking the TUI on slow networks.
	httpTimeout = 3 * time.Second

	// cacheFile is the name of the cached update-check result.
	cacheFile = "update-check.json"
)

// CheckResult holds the outcome of an update check.
type CheckResult struct {
	Available bool   // true if a newer version exists
	Latest    string // the latest version string (e.g. "0.2.0")
}

// cache is the on-disk format for the cached check result.
type cache struct {
	CheckedAt string `json:"checked_at"`
	Latest    string `json:"latest"`
}

// githubRelease is a minimal representation of the GitHub API response.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// Check queries GitHub for the latest release and compares it to current.
// It caches results under dopsDir to avoid redundant API calls.
// Returns a zero CheckResult (Available=false) on any error — never surfaces errors to the caller.
func Check(current, dopsDir string) CheckResult {
	if shouldSkip(current) {
		return CheckResult{}
	}

	cachePath := filepath.Join(dopsDir, cacheFile)

	// Try the cache first.
	if r, ok := readCache(cachePath, current); ok {
		return r
	}

	latest, err := fetchLatest()
	if err != nil {
		return CheckResult{}
	}

	writeCache(cachePath, latest)

	if isNewer(latest, current) {
		return CheckResult{Available: true, Latest: latest}
	}
	return CheckResult{}
}

// shouldSkip returns true for dev builds or empty versions.
func shouldSkip(v string) bool {
	return v == "" || v == "dev" || strings.Contains(v, "-dev")
}

// readCache reads the cached result if it's fresh enough.
// Returns the CheckResult and true if the cache is valid.
func readCache(path, current string) (CheckResult, bool) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is internal cache file (~/.dops/cache.json)
	if err != nil {
		return CheckResult{}, false
	}

	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		return CheckResult{}, false
	}

	checkedAt, err := time.Parse(time.RFC3339, c.CheckedAt)
	if err != nil {
		return CheckResult{}, false
	}

	if time.Since(checkedAt) > cacheTTL {
		return CheckResult{}, false
	}

	if isNewer(c.Latest, current) {
		return CheckResult{Available: true, Latest: c.Latest}, true
	}
	return CheckResult{}, true
}

// writeCache persists the latest version to disk. Errors are silently ignored.
func writeCache(path, latest string) {
	c := cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    latest,
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o750)
	_ = os.WriteFile(path, data, 0o600)
}

// apiBaseURL is the GitHub API base URL. Override in tests to use httptest.
var apiBaseURL = "https://api.github.com"

// fetchLatest hits the GitHub Releases API and returns the latest version tag.
func fetchLatest() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBaseURL, githubRepo)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(url) // nolint:noctx -- fire-and-forget update check
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api: %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release response: %w", err)
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// isNewer returns true if latest is a higher semver than current.
// Both should be plain semver strings like "0.2.0" (no "v" prefix).
func isNewer(latest, current string) bool {
	l := parseSemver(latest)
	c := parseSemver(current)
	if l == nil || c == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

// parseSemver splits "1.2.3" into [1, 2, 3]. Returns nil on failure.
func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		// Strip pre-release suffix (e.g. "3-rc1" → "3")
		if idx := strings.IndexByte(p, '-'); idx >= 0 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}
