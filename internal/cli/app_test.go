package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

type testApp struct {
	*App
	out *bytes.Buffer
	err *bytes.Buffer
}

func newTestApp(input string) testApp {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	return testApp{
		App: &App{Version: "1.2.3", Skill: "skill body\n", In: strings.NewReader(input), Out: out, Err: errOut},
		out: out,
		err: errOut,
	}
}

// stubSession swaps config loading for the given config. Call the returned
// function with defer to restore the originals.
func stubSession(cfg *types.Config, userCfg *userconfig.UserConfig) func() {
	origInit, origLoad, origUser := initLogger, loadConfig, loadUserConfig
	initLogger = func(bool) error { return nil }
	loadConfig = func() (*types.Config, error) { return cfg, nil }
	loadUserConfig = func() (*userconfig.UserConfig, error) { return userCfg, nil }
	return func() {
		initLogger, loadConfig, loadUserConfig = origInit, origLoad, origUser
	}
}

func emptyUserConfig() *userconfig.UserConfig {
	return &userconfig.UserConfig{ExcludedPaths: map[string][]string{}}
}

// writeFiles creates files of the given sizes under dir and returns their paths.
func writeFiles(t *testing.T, dir string, sizes map[string]int) map[string]string {
	t.Helper()
	paths := make(map[string]string, len(sizes))
	for name, size := range sizes {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, bytes.Repeat([]byte("x"), size), 0o644))
		paths[name] = p
	}
	return paths
}

func decodeError(t *testing.T, data []byte) cliError {
	t.Helper()
	var payload struct {
		Error cliError `json:"error"`
	}
	require.NoError(t, json.Unmarshal(data, &payload))
	return payload.Error
}

func TestRun_UnknownCommand_ReturnsUsageExit(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"bogus"})

	assert.Equal(t, ExitUsage, code)
	assert.Contains(t, app.err.String(), `unknown command "bogus"`)
	assert.Contains(t, app.err.String(), "hint: run `mac-cleanup help`")
	assert.Empty(t, app.out.String())
}

func TestRun_UpdateWithUnexpectedArgument_DoesNotUpdate(t *testing.T) {
	defer stubSession(testConfig(), emptyUserConfig())()
	original := runUpdate
	called := false
	runUpdate = func() error { called = true; return nil }
	defer func() { runUpdate = original }()
	app := newTestApp("")

	code := app.Run([]string{"update", "unexpected"})

	assert.Equal(t, ExitUsage, code)
	assert.False(t, called)
	assert.Contains(t, app.err.String(), "unexpected argument")
}

func TestRun_JSONFlag_WritesErrorAsJSON(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"clean", "--json"})

	assert.Equal(t, ExitUsage, code)
	e := decodeError(t, app.err.Bytes())
	assert.Equal(t, "missing_target", e.Code)
	assert.NotEmpty(t, e.Hint)
}

func TestRun_LegacyFlag_PointsToSubcommand(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"--clean", "--dry-run"})

	assert.Equal(t, ExitUsage, code)
	assert.Contains(t, app.err.String(), "mac-cleanup clean --selected")
}

func TestRun_NoArgsWithoutTerminal_DoesNotOpenTUI(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")
	opened := false
	app.RunTUI = func(*types.Config) error {
		opened = true
		return nil
	}

	code := app.Run(nil)

	assert.Equal(t, ExitUsage, code)
	assert.False(t, opened)
	assert.Contains(t, app.err.String(), "needs a terminal")
}

func TestRun_NoArgsWithTerminal_OpensTUI(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	origTerminal := isTerminal
	isTerminal = func(any) bool { return true }
	defer func() { isTerminal = origTerminal }()
	app := newTestApp("")
	opened := false
	app.RunTUI = func(*types.Config) error {
		opened = true
		return nil
	}

	code := app.Run(nil)

	assert.Equal(t, ExitOK, code)
	assert.True(t, opened)
}

func TestRun_HelpFlagOnCommand_PrintsCommandUsage(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"scan", "--help"})

	assert.Equal(t, ExitOK, code)
	assert.Contains(t, app.out.String(), "Usage: mac-cleanup scan")
}

func TestRun_Help_ListsEveryCommand(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"help"})

	assert.Equal(t, ExitOK, code)
	for _, cmd := range commandList() {
		assert.Contains(t, app.out.String(), "  "+cmd.name+" ")
	}
}

func TestRun_Skill_PrintsEmbeddedSkill(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"skill"})

	assert.Equal(t, ExitOK, code)
	assert.Equal(t, "skill body\n", app.out.String())
}

func TestRun_DebugFlag_IsAcceptedAnywhere(t *testing.T) {
	defer stubSession(&types.Config{}, emptyUserConfig())()
	var debug bool
	initLogger = func(d bool) error {
		debug = d
		return nil
	}
	app := newTestApp("")

	code := app.Run([]string{"version", "--debug"})

	assert.Equal(t, ExitOK, code)
	assert.True(t, debug)
	assert.Equal(t, "mac-cleanup 1.2.3\n", app.out.String())
}
