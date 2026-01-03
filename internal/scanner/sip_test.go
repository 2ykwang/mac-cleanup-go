package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSIPProtected_ReturnsTrue_ForSystemPaths(t *testing.T) {
	assert.True(t, IsSIPProtected("/System/Library/Caches"))
	assert.True(t, IsSIPProtected("/usr/bin/something"))
	assert.True(t, IsSIPProtected("/bin/bash"))
	assert.True(t, IsSIPProtected("/sbin/mount"))
}

func TestIsSIPProtected_ReturnsFalse_ForUsrLocal(t *testing.T) {
	assert.False(t, IsSIPProtected("/usr/local/Homebrew"))
	assert.False(t, IsSIPProtected("/usr/local/bin/brew"))
	assert.False(t, IsSIPProtected("/usr/local/var/cache"))
}

func TestIsSIPProtected_ReturnsFalse_ForUserPaths(t *testing.T) {
	assert.False(t, IsSIPProtected("/Users/test/Library/Caches"))
	assert.False(t, IsSIPProtected("/Applications/SomeApp.app"))
	assert.False(t, IsSIPProtected("/Library/Caches"))
	assert.False(t, IsSIPProtected("/tmp/test"))
	assert.False(t, IsSIPProtected("/private/tmp/test"))
}

func TestIsSIPProtected_ReturnsFalse_ForVar(t *testing.T) {
	// /var is a symlink to /private/var on macOS, which is writable
	assert.False(t, IsSIPProtected("/var/log/system.log"))
}
