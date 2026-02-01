package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	execLookPath  = exec.LookPath
	execCommand   = exec.Command
	httpClient    = &http.Client{Timeout: 2 * time.Second}
	formulaAPIURL = "https://formulae.brew.sh/api/formula/" + BrewFormula + ".json"
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

// brewFormulaResponse is a minimal representation of the Homebrew Formulae API response.
type brewFormulaResponse struct {
	Versions struct {
		Stable string `json:"stable"`
	} `json:"versions"`
}

// CheckForUpdate checks if a newer version is available via Homebrew Formulae API.
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
	req, err := http.NewRequest(http.MethodGet, formulaAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mac-cleanup-go")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("homebrew formula API returned status %d", resp.StatusCode)
	}

	var formula brewFormulaResponse
	if err := json.NewDecoder(resp.Body).Decode(&formula); err != nil {
		return "", fmt.Errorf("failed to parse formula response: %w", err)
	}

	if formula.Versions.Stable == "" {
		return "", errors.New("stable version not found in formula response")
	}

	return formula.Versions.Stable, nil
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
