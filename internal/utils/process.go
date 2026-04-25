package utils

import "strings"

// IsProcessRunning reports whether a process with the given name is currently running.
// Uses `pgrep -x` for an exact name match. Returns false if pgrep is missing or errors.
var IsProcessRunning = func(name string) bool {
	if name == "" {
		return false
	}
	out, err := execCommand("pgrep", "-x", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
