package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type hunkRange struct {
	start int
	count int
}

func main() {
	base := flag.String("base", "origin/main", "Base ref for git diff")
	profile := flag.String("profile", "coverage.out", "Go coverprofile path")
	worktree := flag.Bool("worktree", false, "Include unstaged changes when diffing")
	flag.Parse()

	changed, err := diffChangedLines(*base, *worktree)
	if err != nil {
		fmt.Fprintf(os.Stderr, "patch-cover: %v\n", err)
		os.Exit(1)
	}

	if len(changed) == 0 {
		fmt.Println("Patch coverage: no changed lines")
		return
	}

	modulePath, _ := readModulePath()
	covered, measured, err := loadCoverageLines(*profile, modulePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "patch-cover: %v\n", err)
		os.Exit(1)
	}

	var total, hit int
	var uncovered []string
	for file, lines := range changed {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		coveredLines := covered[file]
		measuredLines := measured[file]
		for line := range lines {
			if !measuredLines[line] {
				continue
			}
			total++
			if coveredLines[line] {
				hit++
			} else {
				uncovered = append(uncovered, fmt.Sprintf("%s:%d", file, line))
			}
		}
	}

	percent := 0.0
	if total > 0 {
		percent = float64(hit) / float64(total) * 100
	}

	if total == 0 {
		fmt.Println("Patch coverage: no measurable lines")
		return
	}

	fmt.Printf("Patch coverage: %.1f%% (%d/%d)\n", percent, hit, total)
	if len(uncovered) > 0 {
		fmt.Println("Uncovered lines:")
		for _, entry := range uncovered {
			fmt.Printf("  %s\n", entry)
		}
	}
}

func diffChangedLines(base string, worktree bool) (map[string]map[int]bool, error) {
	args := []string{"diff", "--unified=0"}
	if worktree {
		args = append(args, base)
	} else {
		args = append(args, base+"...HEAD")
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	changed := make(map[string]map[int]bool)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var currentFile string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if path == "/dev/null" {
				currentFile = ""
				continue
			}
			currentFile = normalizePath(path)
			continue
		}
		if !strings.HasPrefix(line, "@@ ") || currentFile == "" {
			continue
		}
		hunk, err := parseHunk(line)
		if err != nil {
			continue
		}
		if hunk.count == 0 {
			continue
		}
		if changed[currentFile] == nil {
			changed[currentFile] = make(map[int]bool)
		}
		for i := 0; i < hunk.count; i++ {
			changed[currentFile][hunk.start+i] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan diff: %w", err)
	}
	return changed, nil
}

func parseHunk(line string) (hunkRange, error) {
	// Example: @@ -1,0 +2,4 @@
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return hunkRange{}, errors.New("invalid hunk")
	}
	add := parts[2]
	if !strings.HasPrefix(add, "+") {
		return hunkRange{}, errors.New("invalid hunk add range")
	}
	add = strings.TrimPrefix(add, "+")
	r := strings.Split(add, ",")
	start, err := strconv.Atoi(r[0])
	if err != nil {
		return hunkRange{}, err
	}
	count := 1
	if len(r) > 1 {
		count, err = strconv.Atoi(r[1])
		if err != nil {
			return hunkRange{}, err
		}
	}
	return hunkRange{start: start, count: count}, nil
}

func normalizePath(path string) string {
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	path = strings.TrimPrefix(path, "./")
	return filepath.Clean(path)
}

func loadCoverageLines(profile string, modulePath string) (map[string]map[int]bool, map[string]map[int]bool, error) {
	data, err := os.ReadFile(profile)
	if err != nil {
		return nil, nil, fmt.Errorf("read profile: %w", err)
	}

	covered := make(map[string]map[int]bool)
	measured := make(map[string]map[int]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		fileRange := fields[0]
		countStr := fields[2]

		file, startLine, endLine, err := parseCoverRange(fileRange, modulePath)
		if err != nil {
			continue
		}

		if measured[file] == nil {
			measured[file] = make(map[int]bool)
		}
		for lineNo := startLine; lineNo <= endLine; lineNo++ {
			measured[file][lineNo] = true
		}

		count, err := strconv.Atoi(countStr)
		if err != nil || count == 0 {
			continue
		}
		if covered[file] == nil {
			covered[file] = make(map[int]bool)
		}
		for lineNo := startLine; lineNo <= endLine; lineNo++ {
			covered[file][lineNo] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan profile: %w", err)
	}
	return covered, measured, nil
}

func parseCoverRange(entry string, modulePath string) (string, int, int, error) {
	// Example: internal/config/config.go:10.12,12.2
	parts := strings.Split(entry, ":")
	if len(parts) != 2 {
		return "", 0, 0, errors.New("invalid cover range")
	}
	path := normalizeCoveragePath(parts[0], modulePath)
	rng := parts[1]
	points := strings.Split(rng, ",")
	if len(points) != 2 {
		return "", 0, 0, errors.New("invalid cover range points")
	}

	startLine, err := parseLine(points[0])
	if err != nil {
		return "", 0, 0, err
	}
	endLine, err := parseLine(points[1])
	if err != nil {
		return "", 0, 0, err
	}
	return path, startLine, endLine, nil
}

func parseLine(part string) (int, error) {
	fields := strings.Split(part, ".")
	if len(fields) < 1 {
		return 0, errors.New("invalid line")
	}
	return strconv.Atoi(fields[0])
}

func readModulePath() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", errors.New("module path not found")
}

func normalizeCoveragePath(path string, modulePath string) string {
	path = strings.TrimPrefix(path, "./")
	if modulePath != "" {
		prefix := modulePath + "/"
		path = strings.TrimPrefix(path, prefix)
	}
	return filepath.Clean(path)
}
