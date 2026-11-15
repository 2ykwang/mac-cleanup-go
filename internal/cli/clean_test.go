package cli

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// Tests use MethodPermanent so --yes never touches the real Trash.

func runCleanJSON(t *testing.T, wantCode int, args ...string) cleanOutput {
	t.Helper()
	app := newTestApp("")
	code := app.Run(append([]string{"clean", "--json"}, args...))
	require.Equal(t, wantCode, code, app.err.String())

	var out cleanOutput
	require.NoError(t, json.Unmarshal(app.out.Bytes(), &out))
	return out
}

func skipReasons(out cleanOutput) map[string]string {
	reasons := make(map[string]string, len(out.Skipped))
	for _, s := range out.Skipped {
		reasons[s.ID] = s.Reason
	}
	return reasons
}

func TestRunClean_WithoutYes_RequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"clean", "logs", "--json"})

	assert.Equal(t, ExitUsage, code)
	assert.Equal(t, "confirmation_required", decodeError(t, app.err.Bytes()).Code)
	assert.FileExists(t, files["a.log"])
}

func TestRunClean_HelpAfterYes_DoesNotScanOrClean(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"clean", "logs", "--yes", "--help"})

	assert.Equal(t, ExitOK, code)
	assert.Contains(t, app.out.String(), "Usage: mac-cleanup clean")
	assert.FileExists(t, files["a.log"])
}

func TestRunClean_DryRun_ChangesNothing(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10, "b.log": 5})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()

	out := runCleanJSON(t, ExitOK, "logs", "--dry-run")

	assert.True(t, out.DryRun)
	assert.Equal(t, int64(15), out.FreedBytes)
	require.Len(t, out.Targets, 1)
	assert.Equal(t, cleanPlanned, out.Targets[0].Status)
	assert.ElementsMatch(t, []string{files["a.log"], files["b.log"]}, out.Targets[0].ItemPaths)
	assert.FileExists(t, files["a.log"])
	assert.NotNil(t, out.Skipped, "skipped is always an array")
}

func TestRunClean_Yes_HonorsExcludedPaths(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"keep.log": 10, "drop.log": 5})
	userCfg := emptyUserConfig()
	userCfg.ExcludedPaths["logs"] = []string{files["keep.log"]}
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), userCfg)()

	out := runCleanJSON(t, ExitOK, "logs", "--yes")

	assert.False(t, out.DryRun)
	assert.Equal(t, 1, out.CleanedItems)
	assert.Equal(t, cleanOK, out.Targets[0].Status)
	assert.FileExists(t, files["keep.log"])
	assert.NoFileExists(t, files["drop.log"])
}

func TestRunClean_Item_CleansOnlyThatPath(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10, "b.log": 5})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()

	out := runCleanJSON(t, ExitOK, "logs", "--item", files["b.log"], "--yes")

	assert.Equal(t, 1, out.CleanedItems)
	assert.FileExists(t, files["a.log"])
	assert.NoFileExists(t, files["b.log"])
}

func TestRunClean_ItemDryRun_ReportsOnlySelectedPath(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10, "b.log": 5})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()

	out := runCleanJSON(t, ExitOK, "logs", "--item", files["b.log"], "--dry-run")

	require.Len(t, out.Targets, 1)
	assert.Equal(t, []string{files["b.log"]}, out.Targets[0].ItemPaths)
	assert.FileExists(t, files["a.log"])
	assert.FileExists(t, files["b.log"])
}

func TestRunClean_UnknownItem_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"clean", "logs", "--item", dir + "/other.log", "--dry-run", "--json"})

	assert.Equal(t, ExitUsage, code)
	assert.Equal(t, "unknown_item", decodeError(t, app.err.Bytes()).Code)
}

type sideEffectTarget struct {
	target.Target
}

func (sideEffectTarget) CleanSideEffect() string { return "also prunes elsewhere" }

// stubSideEffect wraps the target id so it reports a cleanup side effect.
func stubSideEffect(id string) func() {
	orig := newRegistry
	newRegistry = func(cfg *types.Config) (*target.Registry, error) {
		registry, err := orig(cfg)
		if err != nil {
			return nil, err
		}
		tgt, _ := registry.Get(id)
		registry.Register(sideEffectTarget{tgt})
		return registry, nil
	}
	return func() { newRegistry = orig }
}

func TestRunClean_ItemOnSideEffectTarget_IsRejected(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	defer stubSideEffect("logs")()
	app := newTestApp("")

	code := app.Run([]string{"clean", "logs", "--item", files["a.log"], "--yes", "--json"})

	assert.Equal(t, ExitUsage, code)
	assert.Equal(t, "item_not_supported", decodeError(t, app.err.Bytes()).Code)
	assert.FileExists(t, files["a.log"])
}

func TestRunClean_SideEffectTarget_IsDisclosedInDryRun(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	defer stubSideEffect("logs")()

	out := runCleanJSON(t, ExitOK, "logs", "--dry-run")

	require.Len(t, out.Targets, 1)
	assert.Equal(t, "also prunes elsewhere", out.Targets[0].SideEffect)
}

func TestRunClean_RiskyAndManual_AreSkipped(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"chat.db": 10})
	defer stubSession(testConfig(
		testCategory("chat", types.SafetyLevelRisky, types.MethodPermanent, dir+"/*"),
		testCategory("guide", types.SafetyLevelSafe, types.MethodManual, dir+"/*"),
	), emptyUserConfig())()

	out := runCleanJSON(t, ExitOK, "chat", "guide", "--yes")

	assert.Empty(t, out.Targets)
	assert.Equal(t, map[string]string{"chat": skipRiskyNeedItem, "guide": skipManual}, skipReasons(out))
	assert.FileExists(t, files["chat.db"])
}

func TestRunClean_RiskyWithItem_IsCleaned(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"cache.bin": 10, "chat.db": 10})
	defer stubSession(testConfig(testCategory("chat", types.SafetyLevelRisky, types.MethodPermanent, dir+"/*")), emptyUserConfig())()

	out := runCleanJSON(t, ExitOK, "chat", "--item", files["cache.bin"], "--yes")

	assert.Equal(t, 1, out.CleanedItems)
	assert.NoFileExists(t, files["cache.bin"])
	assert.FileExists(t, files["chat.db"])
}

func TestRunClean_FailedItems_ExitPartial(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a.log": 10})
	require.NoError(t, os.Chmod(dir, 0o555))
	defer func() { _ = os.Chmod(dir, 0o755) }()
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()

	out := runCleanJSON(t, ExitPartial, "logs", "--yes")

	assert.Equal(t, 1, out.FailedItems)
	require.Len(t, out.Targets, 1)
	assert.Equal(t, cleanFailed, out.Targets[0].Status)
	assert.NotEmpty(t, out.Targets[0].Errors)
}

func TestRunClean_Selected_UsesSavedTargets(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a.log": 10})
	userCfg := emptyUserConfig()
	userCfg.SelectedTargets = []string{"logs", "removed-target"}
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), userCfg)()

	out := runCleanJSON(t, ExitOK, "--selected", "--dry-run")

	require.Len(t, out.Targets, 1)
	assert.Equal(t, "logs", out.Targets[0].ID)
	assert.Equal(t, map[string]string{"removed-target": skipUnknownSaved}, skipReasons(out))
}

func TestRunClean_SelectedWithoutSaved_ReturnsNoSelection(t *testing.T) {
	defer stubSession(testConfig(), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"clean", "--selected", "--dry-run", "--json"})

	assert.Equal(t, ExitError, code)
	assert.Equal(t, "no_selection", decodeError(t, app.err.Bytes()).Code)
}

func TestRunClean_SelectedAllRemoved_DoesNotCleanEverything(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10})
	userCfg := emptyUserConfig()
	userCfg.SelectedTargets = []string{"removed-target"}
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), userCfg)()
	app := newTestApp("")

	code := app.Run([]string{"clean", "--selected", "--yes", "--json"})

	assert.Equal(t, ExitError, code)
	assert.Equal(t, "no_selection", decodeError(t, app.err.Bytes()).Code)
	assert.FileExists(t, files["a.log"])
}

func TestRunClean_SelectedWithIDs_IsUsageError(t *testing.T) {
	defer stubSession(testConfig(), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"clean", "logs", "--selected", "--dry-run"})

	assert.Equal(t, ExitUsage, code)
	assert.Contains(t, app.err.String(), "--selected cannot be combined")
}

func TestRunClean_TerminalDeclined_ChangesNothing(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	t.Setenv("NO_COLOR", "1") // skip the terminal background query
	origTerminal := isTerminal
	isTerminal = func(any) bool { return true }
	defer func() { isTerminal = origTerminal }()
	app := newTestApp("n\n")

	code := app.Run([]string{"clean", "logs"})

	assert.Equal(t, ExitError, code)
	assert.Contains(t, app.err.String(), "Permanently deletes")
	assert.Contains(t, app.err.String(), "cancelled")
	assert.FileExists(t, files["a.log"])
}

func TestRunClean_TerminalAccepted_Cleans(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), emptyUserConfig())()
	t.Setenv("NO_COLOR", "1") // skip the terminal background query
	origTerminal := isTerminal
	isTerminal = func(any) bool { return true }
	defer func() { isTerminal = origTerminal }()
	app := newTestApp("y\n")

	code := app.Run([]string{"clean", "logs"})

	assert.Equal(t, ExitOK, code, app.err.String())
	assert.NoFileExists(t, files["a.log"])
}
