package version

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setupLatestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	originalURL := latestReleaseURL
	originalClient := httpClient

	latestReleaseURL = server.URL + "/releases/latest"
	httpClient = server.Client()

	t.Cleanup(func() {
		latestReleaseURL = originalURL
		httpClient = originalClient
		server.Close()
	})
}

func stubBrewAvailable(t *testing.T) {
	t.Helper()

	original := execLookPath
	execLookPath = func(_ string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	t.Cleanup(func() {
		execLookPath = original
	})
}

func stubBrewMissing(t *testing.T) {
	t.Helper()

	original := execLookPath
	execLookPath = func(_ string) (string, error) {
		return "", errors.New("brew not found")
	}
	t.Cleanup(func() {
		execLookPath = original
	})
}

func TestFetchLatestVersion_ResolvesTag(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v2.0.0", http.StatusFound)
		case "/releases/tag/v2.0.0":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	got, err := fetchLatestVersion()

	assert.NoError(t, err)
	assert.Equal(t, "2.0.0", got)
}

func TestFetchLatestVersion_UnresolvedTag(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/latest" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := fetchLatestVersion()

	assert.Error(t, err)
}

func TestFetchLatestVersion_InvalidStatus(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := fetchLatestVersion()

	assert.Error(t, err)
}

func TestIsNewerVersion_NewerMajor(t *testing.T) {
	assert.True(t, isNewerVersion("2.0.0", "1.9.9"))
}

func TestIsNewerVersion_NewerMinor(t *testing.T) {
	assert.True(t, isNewerVersion("1.2.0", "1.1.9"))
}

func TestIsNewerVersion_NewerPatch(t *testing.T) {
	assert.True(t, isNewerVersion("1.1.2", "1.1.1"))
}

func TestIsNewerVersion_SameVersion(t *testing.T) {
	assert.False(t, isNewerVersion("1.0.0", "1.0.0"))
}

func TestIsNewerVersion_OlderVersion(t *testing.T) {
	assert.False(t, isNewerVersion("1.0.0", "1.1.0"))
}

func TestIsNewerVersion_WithVPrefix(t *testing.T) {
	assert.True(t, isNewerVersion("v1.2.0", "v1.1.0"))
	assert.True(t, isNewerVersion("1.2.0", "v1.1.0"))
	assert.True(t, isNewerVersion("v1.2.0", "1.1.0"))
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v2.0.0", http.StatusFound)
		case "/releases/tag/v2.0.0":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	stubBrewAvailable(t)

	result := CheckForUpdate("dev")

	assert.Equal(t, "dev", result.CurrentVersion)
	assert.Equal(t, "2.0.0", result.LatestVersion)
	assert.True(t, result.UpdateAvailable)
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	result := CheckForUpdate("")

	assert.False(t, result.UpdateAvailable)
	assert.Empty(t, result.LatestVersion)
	assert.NoError(t, result.Error)
}

func TestCheckForUpdate_UpdateAvailable(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v2.0.0", http.StatusFound)
		case "/releases/tag/v2.0.0":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	stubBrewAvailable(t)

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "2.0.0", result.LatestVersion)
	assert.True(t, result.UpdateAvailable)
	assert.NoError(t, result.Error)
}

func TestCheckForUpdate_NoUpdateAvailable(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v1.0.0", http.StatusFound)
		case "/releases/tag/v1.0.0":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	stubBrewAvailable(t)

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "1.0.0", result.LatestVersion)
	assert.False(t, result.UpdateAvailable)
	assert.NoError(t, result.Error)
}

func TestCheckForUpdate_RequestError(t *testing.T) {
	originalURL := latestReleaseURL
	originalClient := httpClient
	latestReleaseURL = "https://example.invalid/releases/latest"
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("request failed")
		}),
	}
	t.Cleanup(func() {
		latestReleaseURL = originalURL
		httpClient = originalClient
	})

	result := CheckForUpdate("1.0.0")

	assert.Error(t, result.Error)
	assert.False(t, result.UpdateAvailable)
}

func TestCheckForUpdate_BrewMissing(t *testing.T) {
	setupLatestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v2.0.0", http.StatusFound)
		case "/releases/tag/v2.0.0":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	stubBrewMissing(t)

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "2.0.0", result.LatestVersion)
	assert.False(t, result.UpdateAvailable)
	assert.NoError(t, result.Error)
}

func TestRunUpdate_Success(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := RunUpdate()

	assert.NoError(t, err)
}

func TestRunUpdate_Error(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := RunUpdate()

	assert.Error(t, err)
}
