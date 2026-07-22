//go:build darwin && cgo

package utils

/*
#cgo CFLAGS: -fblocks -mmacosx-version-min=12.0
#cgo LDFLAGS: -framework AppKit -framework Foundation -mmacosx-version-min=12.0
#include <stdint.h>
#include <stdlib.h>

char *mac_cleanup_recycle_paths(const char *paths_json, int64_t timeout_nanoseconds);
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unsafe"
)

const defaultTrashTimeout = 10 * time.Second

var trashTimeout = defaultTrashTimeout

type nativeRecycleResponse struct {
	MovedPaths []string `json:"moved_paths"`
	Error      string   `json:"error"`
}

func recyclePathsPlatform(paths []string) ([]string, error) {
	payload, err := json.Marshal(paths)
	if err != nil {
		return nil, fmt.Errorf("encoding recycle paths: %w", err)
	}

	input := C.CString(string(payload))
	defer C.free(unsafe.Pointer(input))

	output := C.mac_cleanup_recycle_paths(input, C.int64_t(trashTimeout.Nanoseconds()))
	if output == nil {
		return nil, errors.New("NSWorkspace.recycleURLs returned no response")
	}
	defer C.free(unsafe.Pointer(output))

	var response nativeRecycleResponse
	if err := json.Unmarshal([]byte(C.GoString(output)), &response); err != nil {
		return nil, fmt.Errorf("decoding recycle response: %w", err)
	}
	if response.Error != "" {
		return response.MovedPaths, errors.New(response.Error)
	}
	return response.MovedPaths, nil
}
