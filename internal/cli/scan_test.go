package cli

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/mocks"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func testConfig(cats ...types.Category) *types.Config {
	return &types.Config{
		Groups:     []types.Group{{ID: "dev", Name: "Dev"}, {ID: "app", Name: "Apps"}},
		Categories: cats,
	}
}

func testCategory(id string, safety types.SafetyLevel, method types.CleanupMethod, paths ...string) types.Category {
	return types.Category{ID: id, Name: id, Group: "dev", Safety: safety, Method: method, Paths: paths}
}

func runScanJSON(t *testing.T, args ...string) scanOutput {
	t.Helper()
	app := newTestApp("")
	code := app.Run(append([]string{"scan", "--json"}, args...))
	require.Equal(t, ExitOK, code, app.err.String())

	var out scanOutput
	require.NoError(t, json.Unmarshal(app.out.Bytes(), &out))
	return out
}

func TestRunScan_CleanableBytesSkipsExcludedItems(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"a.log": 100, "b.log": 30})
	userCfg := emptyUserConfig()
	userCfg.ExcludedPaths["logs"] = []string{files["b.log"]}
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodTrash, dir+"/*.log")), userCfg)()

	out := runScanJSON(t, "--items")

	require.Len(t, out.Targets, 1)
	logs := out.Targets[0]
	assert.Equal(t, statusOK, logs.Status)
	assert.Equal(t, int64(130), logs.SizeBytes)
	assert.Equal(t, int64(100), logs.CleanableBytes)
	require.Len(t, logs.Items, 2)
	assert.Equal(t, files["a.log"], logs.Items[0].Path, "items are largest first")
	assert.True(t, logs.Items[1].Excluded)
}

func TestScanEntry_PreservesItemDisplayName(t *testing.T) {
	cat := testCategory("docker", types.SafetyLevelModerate, types.MethodBuiltin)
	result := types.NewScanResult(cat)
	result.Items = []types.CleanableItem{{
		Path: "docker:image:abc", Name: "Image: example", DisplayName: "Image: example (used by app)",
	}}

	entry := newScanEntry(scanned{cat: cat, result: result, status: statusOK}, emptyUserConfig(), true, 20)

	require.Len(t, entry.Items, 1)
	assert.Equal(t, "Image: example", entry.Items[0].Name)
	assert.Equal(t, "Image: example (used by app)", entry.Items[0].DisplayName)
}

func TestRunScan_RiskyTargetHasNoCleanableBytes(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"chat.db": 50})
	defer stubSession(testConfig(testCategory("chat", types.SafetyLevelRisky, types.MethodTrash, dir+"/*")), emptyUserConfig())()

	out := runScanJSON(t)

	require.Len(t, out.Targets, 1)
	assert.Equal(t, int64(50), out.Targets[0].SizeBytes)
	assert.Zero(t, out.Targets[0].CleanableBytes)
	assert.Zero(t, out.CleanableBytes)
}

func TestRunScan_BlockedTargetReportsProcess(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a.log": 10})
	cat := testCategory("app-cache", types.SafetyLevelSafe, types.MethodTrash, dir+"/*")
	cat.BlockedByProcesses = []string{"SomeApp"}
	defer stubSession(testConfig(cat), emptyUserConfig())()
	origRunning := utils.IsProcessRunning
	utils.IsProcessRunning = func(name string) bool { return name == "SomeApp" }
	defer func() { utils.IsProcessRunning = origRunning }()

	out := runScanJSON(t)

	require.Len(t, out.Targets, 1)
	assert.Equal(t, statusBlocked, out.Targets[0].Status)
	assert.Equal(t, "SomeApp", out.Targets[0].BlockedBy)
	assert.Zero(t, out.Targets[0].SizeBytes)
}

func TestRunScan_MissingPathIsUnavailable(t *testing.T) {
	defer stubSession(testConfig(testCategory("gone", types.SafetyLevelSafe, types.MethodTrash, t.TempDir()+"/missing/*")), emptyUserConfig())()

	out := runScanJSON(t)

	require.Len(t, out.Targets, 1)
	assert.Equal(t, statusUnavailable, out.Targets[0].Status)
}

func TestRunCommands_ReportErrorStoredInScanResult(t *testing.T) {
	cat := testCategory("docker", types.SafetyLevelModerate, types.MethodBuiltin)
	result := types.NewScanResult(cat)
	result.Error = errors.New("docker disk usage failed")
	scanner := &mocks.MockTarget{}
	scanner.On("Category").Return(cat)
	scanner.On("IsAvailable").Return(true)
	scanner.On("Scan").Return(result, nil)
	registry := target.NewRegistry()
	registry.Register(scanner)

	original := newRegistry
	newRegistry = func(*types.Config) (*target.Registry, error) { return registry, nil }
	defer func() { newRegistry = original }()
	defer stubSession(testConfig(cat), emptyUserConfig())()

	scan := runScanJSON(t, "docker")
	require.Len(t, scan.Targets, 1)
	assert.Equal(t, statusError, scan.Targets[0].Status)
	assert.Equal(t, result.Error.Error(), scan.Targets[0].Error)

	clean := runCleanJSON(t, ExitOK, "docker", "--dry-run")
	assert.Empty(t, clean.Targets)
	require.Len(t, clean.Skipped, 1)
	assert.Equal(t, skipScanFailed, clean.Skipped[0].Reason)
	assert.Equal(t, result.Error.Error(), clean.Skipped[0].Detail)
	scanner.AssertExpectations(t)
}

func TestRunScan_LimitReportsMoreItems(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a": 3, "b": 2, "c": 1})
	defer stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodTrash, dir+"/*")), emptyUserConfig())()

	out := runScanJSON(t, "--items", "--limit", "1")

	require.Len(t, out.Targets[0].Items, 1)
	assert.Equal(t, 2, out.Targets[0].MoreItems)
}

func TestRunScan_UnknownTarget_ReturnsUsageError(t *testing.T) {
	defer stubSession(testConfig(), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"scan", "nope", "--json"})

	assert.Equal(t, ExitUsage, code)
	assert.Equal(t, "unknown_target", decodeError(t, app.err.Bytes()).Code)
}

func TestRunScan_TextOutputHidesEmptyTargets(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]int{"a.log": 10})
	defer stubSession(testConfig(
		testCategory("logs", types.SafetyLevelSafe, types.MethodTrash, dir+"/*.log"),
		testCategory("gone", types.SafetyLevelSafe, types.MethodTrash, dir+"/missing/*"),
	), emptyUserConfig())()
	app := newTestApp("")

	code := app.Run([]string{"scan"})

	assert.Equal(t, ExitOK, code)
	assert.Contains(t, app.out.String(), "logs")
	assert.NotContains(t, app.out.String(), "gone ")
	assert.Contains(t, app.out.String(), "1 not installed")
}
