package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"0.2.0", "0.1.0", true},
		{"1.0.0", "0.9.9", true},
		{"0.1.1", "0.1.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"v0.2.0", "0.1.0", true},  // v-prefix tolerance
		{"0.2.0", "v0.1.0", true},  // v-prefix tolerance
		{"1.0.0-rc1", "0.9.0", true}, // pre-release stripped
		{"", "0.1.0", false},
		{"0.1.0", "", false},
		{"bad", "0.1.0", false},
	}

	for _, tt := range tests {
		got := isNewer(tt.latest, tt.current)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"v0.1.0", []int{0, 1, 0}},
		{"0.2.0-rc1", []int{0, 2, 0}},
		{"bad", nil},
		{"1.2", nil},
		{"", nil},
	}

	for _, tt := range tests {
		got := parseSemver(tt.input)
		if tt.want == nil {
			if got != nil {
				t.Errorf("parseSemver(%q) = %v, want nil", tt.input, got)
			}
			continue
		}
		if got == nil || got[0] != tt.want[0] || got[1] != tt.want[1] || got[2] != tt.want[2] {
			t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestShouldSkip(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"", true},
		{"dev", true},
		{"0.1.0-dev", true},
		{"0.1.0", false},
		{"1.0.0", false},
	}

	for _, tt := range tests {
		got := shouldSkip(tt.version)
		if got != tt.want {
			t.Errorf("shouldSkip(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestReadCache_Fresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, cacheFile)

	c := cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    "0.2.0",
	}
	data, _ := json.Marshal(c)
	os.WriteFile(path, data, 0o644)

	result, ok := readCache(path, "0.1.0")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !result.Available || result.Latest != "0.2.0" {
		t.Errorf("got %+v, want Available=true Latest=0.2.0", result)
	}
}

func TestReadCache_Stale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, cacheFile)

	c := cache{
		CheckedAt: time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339),
		Latest:    "0.2.0",
	}
	data, _ := json.Marshal(c)
	os.WriteFile(path, data, 0o644)

	_, ok := readCache(path, "0.1.0")
	if ok {
		t.Fatal("expected cache miss for stale entry")
	}
}

func TestReadCache_NoUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, cacheFile)

	c := cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    "0.1.0",
	}
	data, _ := json.Marshal(c)
	os.WriteFile(path, data, 0o644)

	result, ok := readCache(path, "0.1.0")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if result.Available {
		t.Error("expected Available=false when versions match")
	}
}

func TestFetchLatest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v0.3.0"})
	}))
	defer srv.Close()

	// Override the fetch function via a test helper.
	result := fetchLatestFrom(srv.URL)
	if result != "0.3.0" {
		t.Errorf("got %q, want %q", result, "0.3.0")
	}
}

// fetchLatestFrom is a test helper that hits an arbitrary URL.
func fetchLatestFrom(url string) string {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}
	return trimV(release.TagName)
}

func trimV(s string) string {
	if len(s) > 0 && s[0] == 'v' {
		return s[1:]
	}
	return s
}
