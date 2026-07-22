//go:build !darwin || !cgo

package utils

import "errors"

func recyclePathsPlatform(_ []string) ([]string, error) {
	return nil, errors.New("NSWorkspace.recycleURLs requires macOS with cgo enabled")
}
