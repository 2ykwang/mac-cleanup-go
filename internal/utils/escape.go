package utils

import (
	"errors"
	"strings"
)

// ErrInvalidPath is returned when a path contains invalid characters for AppleScript.
var ErrInvalidPath = errors.New("path contains invalid characters")

// EscapeForAppleScript escapes a string for safe use in AppleScript.
func EscapeForAppleScript(s string) (string, error) {
	if strings.ContainsAny(s, "\n\r") {
		return "", ErrInvalidPath
	}
	// Escape backslash first (order matters!)
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s, nil
}
