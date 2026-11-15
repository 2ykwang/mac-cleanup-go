package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs_FlagsAfterPositionals(t *testing.T) {
	fs := newFlagSet("test")
	dryRun := fs.Bool("dry-run", false, "")

	args, err := parseArgs(fs, []string{"npm", "--dry-run", "yarn"})

	require.NoError(t, err)
	assert.Equal(t, []string{"npm", "yarn"}, args)
	assert.True(t, *dryRun)
}

func TestParseArgs_DoubleDashEndsFlags(t *testing.T) {
	fs := newFlagSet("test")
	dryRun := fs.Bool("dry-run", false, "")

	args, err := parseArgs(fs, []string{"npm", "--", "--dry-run"})

	require.NoError(t, err)
	assert.Equal(t, []string{"npm", "--dry-run"}, args)
	assert.False(t, *dryRun)
}

func TestSplitList_AcceptsCommasAndRepeats(t *testing.T) {
	assert.Equal(t, []string{"safe", "moderate", "risky"}, splitList([]string{"safe, moderate", "risky", ""}))
}
