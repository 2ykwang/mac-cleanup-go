package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestSafetyHintStyle_Safe(t *testing.T) {
	style := safetyHintStyle(types.SafetyHintSafe)

	assert.Equal(t, MutedStyle, style)
}

func TestSafetyHintStyle_Warning(t *testing.T) {
	style := safetyHintStyle(types.SafetyHintWarning)

	assert.Equal(t, WarningStyle, style)
}

func TestSafetyHintStyle_Danger(t *testing.T) {
	style := safetyHintStyle(types.SafetyHintDanger)

	assert.Equal(t, DangerStyle, style)
}

func TestSafetyHintDot_ReturnsStyledDot(t *testing.T) {
	dotSafe := safetyHintDot(types.SafetyHintSafe)
	dotWarning := safetyHintDot(types.SafetyHintWarning)
	dotDanger := safetyHintDot(types.SafetyHintDanger)

	assert.Contains(t, dotSafe, "●")
	assert.Contains(t, dotWarning, "●")
	assert.Contains(t, dotDanger, "●")
}
