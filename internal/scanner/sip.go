package scanner

import (
	"path/filepath"
	"strings"
)

// SIP protected path prefixes (cannot be modified even by root)
var sipProtectedPrefixes = []string{
	"/System",
	"/usr",
	"/bin",
	"/sbin",
}

// SIP exception paths (writable even with SIP enabled)
var sipExceptionPrefixes = []string{
	"/usr/local",
}

// IsSIPProtected checks if the given path is protected by macOS SIP.
func IsSIPProtected(path string) bool {
	// Resolve symlinks to get the real path
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we can't resolve, use the original path
		resolved = path
	}

	// Check exception paths first (e.g., /usr/local)
	for _, exception := range sipExceptionPrefixes {
		if strings.HasPrefix(resolved, exception) {
			return false
		}
	}

	for _, protected := range sipProtectedPrefixes {
		if strings.HasPrefix(resolved, protected) {
			return true
		}
	}
	return false
}
