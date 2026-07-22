package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

const trashBatchSize = 50

var (
	errInvalidTrashPath = errors.New("invalid trash path")
	errPathNotRecycled  = errors.New("path was not reported as recycled")
	recycleCallMu       sync.Mutex
)

type recyclePathsFunc func(paths []string) ([]string, error)

type trashItem struct {
	path string
	size int64
}

func BatchTrash(items []types.CleanableItem, opts types.BatchTrashOptions) *types.CleanResult {
	return batchTrash(items, opts, recyclePathsPlatform)
}

func batchTrash(
	items []types.CleanableItem,
	opts types.BatchTrashOptions,
	recyclePaths recyclePathsFunc,
) *types.CleanResult {
	result := types.NewCleanResult(opts.Category)
	pending := prepareTrashItems(items, opts, result)

	for start := 0; start < len(pending); start += trashBatchSize {
		end := min(start+trashBatchSize, len(pending))
		recycleTrashBatch(pending[start:end], recyclePaths, result)
	}

	logger.Info("trash completed",
		"total", len(items),
		"cleaned", result.CleanedItems,
		"skipped", result.SkippedItems,
		"failed", len(result.Errors))

	return result
}

func prepareTrashItems(
	items []types.CleanableItem,
	opts types.BatchTrashOptions,
	result *types.CleanResult,
) []trashItem {
	pending := make([]trashItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		if _, duplicate := seen[item.Path]; duplicate {
			continue
		}
		seen[item.Path] = struct{}{}

		if opts.Filter != nil && opts.Filter(item) {
			result.SkippedItems++
			continue
		}
		if err := validateTrashPath(item.Path); err != nil {
			addTrashError(result, item.Path, err)
			continue
		}
		if opts.Validate != nil {
			if err := opts.Validate(item); err != nil {
				addTrashError(result, item.Path, err)
				continue
			}
		}

		candidate := trashItem{path: item.Path, size: item.Size}
		if _, err := os.Lstat(item.Path); os.IsNotExist(err) {
			recordTrashSuccess(result, candidate)
			continue
		}
		pending = append(pending, candidate)
	}

	return pending
}

func validateTrashPath(path string) error {
	switch {
	case path == "":
		return fmt.Errorf("%w: path is empty", errInvalidTrashPath)
	case !utf8.ValidString(path):
		return fmt.Errorf("%w: path is not valid UTF-8", errInvalidTrashPath)
	case strings.IndexByte(path, 0) >= 0:
		return fmt.Errorf("%w: path contains a NUL byte", errInvalidTrashPath)
	default:
		return nil
	}
}

func recycleTrashBatch(
	items []trashItem,
	recyclePaths recyclePathsFunc,
	result *types.CleanResult,
) {
	paths := make([]string, len(items))
	for i, item := range items {
		paths[i] = item.path
	}

	movedPaths, recycleErr := callRecyclePaths(paths, recyclePaths)
	moved := make(map[string]struct{}, len(movedPaths))
	for _, path := range movedPaths {
		moved[path] = struct{}{}
	}

	for _, item := range items {
		if _, ok := moved[item.path]; ok {
			recordTrashSuccess(result, item)
			continue
		}
		resolveUnreportedPath(item, recycleErr, result)
	}
}

func callRecyclePaths(paths []string, recyclePaths recyclePathsFunc) ([]string, error) {
	recycleCallMu.Lock()
	defer recycleCallMu.Unlock()

	return recyclePaths(paths)
}

func resolveUnreportedPath(item trashItem, recycleErr error, result *types.CleanResult) {
	_, statErr := os.Lstat(item.path)
	if os.IsNotExist(statErr) {
		recordTrashSuccess(result, item)
		return
	}
	if statErr != nil {
		addTrashError(result, item.path, fmt.Errorf("checking final state: %w", statErr))
		return
	}
	if recycleErr != nil {
		addTrashError(result, item.path, recycleErr)
		return
	}
	addTrashError(result, item.path, errPathNotRecycled)
}

func recordTrashSuccess(result *types.CleanResult, item trashItem) {
	result.CleanedItems++
	result.FreedSpace += item.size
}

func addTrashError(result *types.CleanResult, path string, err error) {
	result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
}
