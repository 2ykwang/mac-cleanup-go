package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/config"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newTestDockerVolumeModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cfg, err := config.LoadEmbedded()
	require.NoError(t, err)
	m := NewModel(cfg, "")
	m.width, m.height = 110, 32
	m.scanning = false
	volumes, ok := m.registry.Get("docker-volumes")
	require.True(t, ok)
	r := types.NewScanResult(volumes.Category())
	r.Items = []types.CleanableItem{
		{Path: "docker:volume:postgres-data", Name: "Volume: postgres-data", Size: 4096, FileCount: 1},
		{Path: "docker:volume:other-data", Name: "Volume: other-data", Size: 2048, FileCount: 1},
		{Path: "docker:volume:locked-data", Name: "Volume: locked-data", Size: 1024, FileCount: 1, Status: types.ItemStatusProcessLocked},
	}
	r.TotalSize, r.TotalFileCount = 7168, 3
	m.results = []*types.ScanResult{r}
	m.resultMap[r.Category.ID] = r
	return m
}

func TestDockerVolumes_RequiresPerItemApprovalInTUI(t *testing.T) {
	m := newTestDockerVolumeModel(t)
	assert.False(t, m.selected["docker-volumes"])

	m.handleListKey(tea.KeyPressMsg{Code: tea.KeySpace})
	jobsBeforeApproval := m.cleanService.PrepareJobs(m.resultMap, m.selected, m.excluded)
	m.view, m.previewCatID, m.previewItemIndex = ViewPreview, "docker-volumes", 0
	m.handlePreviewKey(tea.KeyPressMsg{Code: tea.KeySpace})
	jobsAfterApproval := m.cleanService.PrepareJobs(m.resultMap, m.selected, m.excluded)

	assert.Empty(t, jobsBeforeApproval)
	require.Len(t, jobsAfterApproval, 1)
	require.Len(t, jobsAfterApproval[0].Items, 1)
	assert.Equal(t, "docker:volume:postgres-data", jobsAfterApproval[0].Items[0].Path)
	assert.True(t, m.isExcluded("docker-volumes", "docker:volume:other-data"))
}
