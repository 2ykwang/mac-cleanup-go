package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

func TestExtractBrewItemName_CellarPath(t *testing.T) {
	path := "/usr/local/Cellar/node/18.0.0"

	result := extractBrewItemName(path)

	assert.Equal(t, "node@18.0.0", result)
}

func TestExtractBrewItemName_CellarPathAppleSilicon(t *testing.T) {
	path := "/opt/homebrew/Cellar/bat/0.26.1"

	result := extractBrewItemName(path)

	assert.Equal(t, "bat@0.26.1", result)
}

func TestExtractBrewItemName_CaskroomPath(t *testing.T) {
	path := "/opt/homebrew/Caskroom/visual-studio-code/1.85.0"

	result := extractBrewItemName(path)

	assert.Equal(t, "visual-studio-code@1.85.0", result)
}

func TestExtractBrewItemName_VersionedPackage(t *testing.T) {
	path := "/opt/homebrew/Cellar/python@3.11/3.11.0"

	result := extractBrewItemName(path)

	assert.Equal(t, "python@3.11@3.11.0", result)
}

func TestExtractBrewItemName_SingleElement(t *testing.T) {
	path := "filename"

	result := extractBrewItemName(path)

	assert.Equal(t, "filename", result)
}

func TestExtractBrewItemName_TwoElements(t *testing.T) {
	path := "parent/child"

	result := extractBrewItemName(path)

	assert.Equal(t, "parent@child", result)
}

func TestExtractBrewItemName_EmptyPath(t *testing.T) {
	path := ""

	result := extractBrewItemName(path)

	assert.Equal(t, "", result)
}

func TestNewBrewScanner_ReturnsNonNil(t *testing.T) {
	cat := types.Category{ID: "homebrew", Name: "Homebrew"}

	s := NewBrewScanner(cat)

	assert.NotNil(t, s)
}

func TestNewBrewScanner_StoresCategory(t *testing.T) {
	cat := types.Category{
		ID:     "homebrew",
		Name:   "Homebrew Cache",
		Safety: types.SafetyLevelSafe,
	}

	s := NewBrewScanner(cat)

	assert.Equal(t, "homebrew", s.category.ID)
	assert.Equal(t, "Homebrew Cache", s.category.Name)
}

func TestBrewScanner_Category_ReturnsConfiguredCategory(t *testing.T) {
	cat := types.Category{
		ID:     "homebrew",
		Name:   "Homebrew",
		Safety: types.SafetyLevelModerate,
	}
	s := NewBrewScanner(cat)

	result := s.Category()

	assert.Equal(t, "homebrew", result.ID)
	assert.Equal(t, "Homebrew", result.Name)
	assert.Equal(t, types.SafetyLevelModerate, result.Safety)
}

func TestBrewScanner_IsAvailable_ReturnsBool(t *testing.T) {
	cat := types.Category{ID: "homebrew", CheckCmd: "brew"}
	s := NewBrewScanner(cat)

	available := s.IsAvailable()

	t.Logf("Homebrew available: %v", available)
}
