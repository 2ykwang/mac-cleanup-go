package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscapeForAppleScript_Normal(t *testing.T) {
	result, err := EscapeForAppleScript("/Users/test/Library/Caches")
	require.NoError(t, err)
	assert.Equal(t, "/Users/test/Library/Caches", result)
}

func TestEscapeForAppleScript_WithQuotes(t *testing.T) {
	result, err := EscapeForAppleScript(`/Users/test/My "Documents"`)
	require.NoError(t, err)
	assert.Equal(t, `/Users/test/My \"Documents\"`, result)
}

func TestEscapeForAppleScript_WithBackslash(t *testing.T) {
	result, err := EscapeForAppleScript(`/Users/test/path\with\backslash`)
	require.NoError(t, err)
	assert.Equal(t, `/Users/test/path\\with\\backslash`, result)
}

func TestEscapeForAppleScript_WithQuotesAndBackslash(t *testing.T) {
	// Tests that backslash is escaped before quotes
	result, err := EscapeForAppleScript(`/Users/test\"path`)
	require.NoError(t, err)
	// Input: /Users/test\"path
	// After backslash escape: /Users/test\\"path
	// After quote escape: /Users/test\\\"path
	assert.Equal(t, `/Users/test\\\"path`, result)
}

func TestEscapeForAppleScript_WithNewline_ReturnsError(t *testing.T) {
	_, err := EscapeForAppleScript("/Users/test/path\nwith\nnewlines")
	assert.ErrorIs(t, err, ErrInvalidPath)
}

func TestEscapeForAppleScript_WithCarriageReturn_ReturnsError(t *testing.T) {
	_, err := EscapeForAppleScript("/Users/test/path\rwith\rreturns")
	assert.ErrorIs(t, err, ErrInvalidPath)
}

func TestEscapeForAppleScript_EmptyString(t *testing.T) {
	result, err := EscapeForAppleScript("")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestEscapeForAppleScript_UnicodeCharacters(t *testing.T) {
	result, err := EscapeForAppleScript("/Users/test/한글경로/日本語")
	require.NoError(t, err)
	assert.Equal(t, "/Users/test/한글경로/日本語", result)
}
