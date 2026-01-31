package version

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	execLookPath     = exec.LookPath
	execCommand      = exec.Command
	httpClient       = &http.Client{Timeout: 2 * time.Second}
	latestReleaseURL = "https://github.com/2ykwang/mac-cleanup-go/releases/latest"
)

const (
	// BrewFormula is the Homebrew formula name for mac-cleanup-go (homebrew-core)
	BrewFormula = "mac-cleanup-go"
)

// CheckResult contains version check results
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Error           error
}

// CheckForUpdate checks if a newer version is available via GitHub Releases.
// Note: GitHub Release may be published 1-3 days before the homebrew-core formula is updated.
func CheckForUpdate(currentVersion string) CheckResult {
	result := CheckResult{CurrentVersion: currentVersion}

	// Skip check for empty version only
	if currentVersion == "" {
		return result
	}

	latestVersion, err := fetchLatestVersion()
	if err != nil {
		result.Error = err
		return result
	}

	result.LatestVersion = latestVersion
	if _, err := execLookPath("brew"); err != nil {
		return result
	}
	if result.LatestVersion != "" {
		result.UpdateAvailable = isNewerVersion(result.LatestVersion, currentVersion)
	}

	return result
}

// RunUpdate executes brew upgrade command
func RunUpdate() error {
	cmd := execCommand("brew", "upgrade", BrewFormula)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fetchLatestVersion() (string, error) {
	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mac-cleanup-go")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return "", errors.New("latest release request failed")
	}

	tag := path.Base(resp.Request.URL.Path)
	if tag == "" || tag == "latest" {
		return "", errors.New("latest tag not resolved")
	}

	return strings.TrimPrefix(tag, "v"), nil
}

// isNewerVersion compares semantic versions
// Returns true if latest > current
func isNewerVersion(latest, current string) bool {
	// Strip 'v' prefix if present
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		l, _ := strconv.Atoi(latestParts[i])
		c, _ := strconv.Atoi(currentParts[i])
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return len(latestParts) > len(currentParts)
}
