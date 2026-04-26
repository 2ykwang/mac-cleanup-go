package styles

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DarkUsesLightForeground(t *testing.T) {
	s := New(true)

	r, g, b, _ := s.Text.RGBA()

	assert.True(t, isLight(r, g, b), "Text color should be light on dark background")
}

func TestNew_LightUsesDarkForeground(t *testing.T) {
	s := New(false)

	r, g, b, _ := s.Text.RGBA()

	assert.False(t, isLight(r, g, b), "Text color should be dark on light background")
}

func TestNew_AdaptiveColorsDifferBetweenModes(t *testing.T) {
	dark := New(true)
	light := New(false)

	assert.NotEqual(t, snapshot(dark.Border), snapshot(light.Border), "Border must adapt")
	assert.NotEqual(t, snapshot(dark.Muted), snapshot(light.Muted), "Muted must adapt")
	assert.NotEqual(t, snapshot(dark.Text), snapshot(light.Text), "Text must adapt")
	assert.NotEqual(t, snapshot(dark.Secondary), snapshot(light.Secondary), "Secondary must adapt")
	assert.NotEqual(t, snapshot(dark.Success), snapshot(light.Success), "Success must adapt")
	assert.NotEqual(t, snapshot(dark.Warning), snapshot(light.Warning), "Warning must adapt")
}

func TestNew_StylesPickUpThemeColors(t *testing.T) {
	dark := New(true)
	light := New(false)

	darkMuted := dark.MutedStyle.GetForeground()
	lightMuted := light.MutedStyle.GetForeground()

	require.NotNil(t, darkMuted)
	require.NotNil(t, lightMuted)

	assert.NotEqual(t, snapshot(darkMuted), snapshot(lightMuted),
		"MutedStyle foreground should differ between dark and light themes")
}

func TestNew_StaticColorsAreBackgroundIndependent(t *testing.T) {
	dark := New(true)
	light := New(false)

	assert.Equal(t, snapshot(dark.Primary), snapshot(light.Primary), "Primary must not change")
	assert.Equal(t, snapshot(dark.Danger), snapshot(light.Danger), "Danger must not change")
}

func snapshot(c color.Color) [4]uint32 {
	r, g, b, a := c.RGBA()
	return [4]uint32{r, g, b, a}
}

func isLight(r, g, b uint32) bool {
	return luma(r, g, b) > 0x7FFF
}

func luma(r, g, b uint32) uint32 {
	return (2126*r + 7152*g + 722*b) / 10000
}
