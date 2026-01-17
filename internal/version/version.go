package version

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	// BrewFormula is the Homebrew formula name for mac-cleanup-go
	BrewFormula = "2ykwang/2ykwang/mac-cleanup-go"
)

// CheckResult contains version check results
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Error           error
}

// CheckForUpdate checks if a newer version is available via Homebrew
func CheckForUpdate(currentVersion string) CheckResult {
	result := CheckResult{CurrentVersion: currentVersion}

	// Skip check for empty version only
	if currentVersion == "" {
		return result
	}

	// Check if brew is available
	if _, err := exec.LookPath("brew"); err != nil {
		return result
	}

	// Run: brew info 2ykwang/2ykwang/mac-cleanup-go
	cmd := exec.Command("brew", "info", BrewFormula)
	output, err := cmd.Output()
	if err != nil {
		result.Error = err
		return result
	}

	// Parse version from output
	result.LatestVersion = parseBrewVersion(string(output))
	if result.LatestVersion != "" {
		result.UpdateAvailable = isNewerVersion(result.LatestVersion, currentVersion)
	}

	return result
}

// RunUpdate executes brew upgrade command
func RunUpdate() error {
	cmd := exec.Command("brew", "upgrade", BrewFormula)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// parseBrewVersion extracts version from brew info output
// Example: "==> 2ykwang/2ykwang/mac-cleanup-go: stable 1.3.1"
func parseBrewVersion(output string) string {
	re := regexp.MustCompile(`mac-cleanup-go:\s*stable\s+(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
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
