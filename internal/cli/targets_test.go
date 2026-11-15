package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestRunTargets_FiltersByGroupAndSafety(t *testing.T) {
	app := testCategory("slack", types.SafetyLevelRisky, types.MethodTrash, "/x")
	app.Group = "app"
	defer stubSession(testConfig(
		testCategory("npm", types.SafetyLevelSafe, types.MethodTrash, "/x"),
		testCategory("go", types.SafetyLevelModerate, types.MethodTrash, "/x"),
		app,
	), emptyUserConfig())()
	ta := newTestApp("")

	code := ta.Run([]string{"targets", "--group", "dev", "--safety", "safe,risky", "--json"})

	require.Equal(t, ExitOK, code, ta.err.String())
	var out targetsOutput
	require.NoError(t, json.Unmarshal(ta.out.Bytes(), &out))
	require.Len(t, out.Targets, 1)
	assert.Equal(t, "npm", out.Targets[0].ID)
	assert.Empty(t, out.Targets[0].Paths, "paths only appear when IDs are given")
}

func TestRunTargets_UnknownGroup_ListsValidGroups(t *testing.T) {
	defer stubSession(testConfig(), emptyUserConfig())()
	ta := newTestApp("")

	code := ta.Run([]string{"targets", "--group", "nope", "--json"})

	assert.Equal(t, ExitUsage, code)
	e := decodeError(t, ta.err.Bytes())
	assert.Equal(t, "unknown_group", e.Code)
	assert.Contains(t, e.Hint, "dev")
}
