//go:build perf

package benchfixtures

import (
	"os"
	"path/filepath"
)

type BenchDirSpec struct {
	Name  string
	Depth int
}

type BenchDir struct {
	Name  string
	Dir   string
	Depth int
}

func PrepareBenchDirs(envVar, tempPrefix string, specs []BenchDirSpec, filesPerDir, fanout int) ([]BenchDir, func(), error) {
	root := os.Getenv(envVar)
	shouldCleanup := root == ""
	if shouldCleanup {
		var err error
		root, err = os.MkdirTemp("", tempPrefix)
		if err != nil {
			return nil, func() {}, err
		}
	}

	cleanup := func() {
		if shouldCleanup {
			_ = os.RemoveAll(root)
		}
	}

	dirs := make([]BenchDir, len(specs))
	for i, spec := range specs {
		dir := filepath.Join(root, spec.Name)
		dirs[i] = BenchDir{Name: spec.Name, Dir: dir, Depth: spec.Depth}
		if _, err := os.Stat(dir); err != nil {
			if !os.IsNotExist(err) {
				cleanup()
				return nil, func() {}, err
			}
			if err := createTree(dir, spec.Depth, filesPerDir, fanout); err != nil {
				cleanup()
				return nil, func() {}, err
			}
		}
	}

	return dirs, cleanup, nil
}

func createTree(root string, depth, filesPerDir, fanout int) error {
	data := make([]byte, 1024)
	var create func(path string, d int) error
	create = func(path string, d int) error {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
		for i := 0; i < filesPerDir; i++ {
			filename := filepath.Join(path, string(rune('a'+i))+".txt")
			if err := os.WriteFile(filename, data, 0o644); err != nil {
				return err
			}
		}
		if d < depth {
			for i := 0; i < fanout; i++ {
				if err := create(filepath.Join(path, "d"+string(rune('a'+i))), d+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return create(root, 1)
}
