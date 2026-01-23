package target

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type DockerTarget struct {
	category types.Category
}

func init() {
	RegisterBuiltin("docker", func(cat types.Category, _ []types.Category) Target {
		return NewDockerTarget(cat)
	})
}

func NewDockerTarget(cat types.Category) *DockerTarget {
	return &DockerTarget{category: cat}
}

func (s *DockerTarget) Category() types.Category {
	return s.category
}

func (s *DockerTarget) IsAvailable() bool {
	if !utils.CommandExists("docker") {
		logger.Debug("docker command not found")
		return false
	}
	cmd := execCommand("docker", "version")
	if err := cmd.Run(); err != nil {
		logger.Warn("docker daemon not running", "error", err)
		return false
	}
	return true
}

type dockerDfOutput struct {
	Type        string `json:"Type"`
	TotalCount  string `json:"TotalCount"`
	Active      string `json:"Active"`
	Size        string `json:"Size"`
	Reclaimable string `json:"Reclaimable"`
}

func (s *DockerTarget) Scan() (*types.ScanResult, error) {
	result := types.NewScanResult(s.category)

	if !s.IsAvailable() {
		return result, nil
	}

	cmd := execCommand("docker", "system", "df", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("docker system df failed", "error", err)
		result.Error = err
		return result, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var df dockerDfOutput
		if err := json.Unmarshal([]byte(line), &df); err != nil {
			logger.Debug("docker df json parse failed", "line", line, "error", err)
			continue
		}

		size := parseDockerSize(df.Reclaimable)
		if size == 0 {
			continue
		}

		fileCount, _ := strconv.ParseInt(df.TotalCount, 10, 64)

		item := types.CleanableItem{
			Path:        "docker:" + strings.ToLower(df.Type),
			Size:        size,
			FileCount:   fileCount,
			Name:        dockerTypeName(df.Type),
			IsDirectory: false,
		}
		result.Items = append(result.Items, item)
		result.TotalSize += size
		result.TotalFileCount += fileCount
	}

	logger.Info("docker scan completed",
		"resourceTypes", len(result.Items),
		"totalSize", result.TotalSize)

	return result, nil
}

func (s *DockerTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := types.NewCleanResult(s.category)

	for _, item := range items {
		var cmd *exec.Cmd
		switch item.Path {
		case "docker:images":
			cmd = execCommand("docker", "image", "prune", "-af")
		case "docker:containers":
			cmd = execCommand("docker", "container", "prune", "-f")
		case "docker:local volumes":
			cmd = execCommand("docker", "volume", "prune", "-af")
		case "docker:build cache":
			cmd = execCommand("docker", "builder", "prune", "-af")
		}

		if cmd != nil {
			if err := cmd.Run(); err != nil {
				logger.Warn("docker prune failed", "resourceType", item.Path, "error", err)
				result.Errors = append(result.Errors, err.Error())
			} else {
				logger.Debug("docker prune succeeded", "resourceType", item.Path, "size", item.Size)
				result.FreedSpace += item.Size
				result.CleanedItems++
			}
		}
	}

	logger.Info("docker clean completed",
		"cleanedItems", result.CleanedItems,
		"freedSpace", result.FreedSpace,
		"errors", len(result.Errors))

	return result, nil
}

func parseDockerSize(s string) int64 {
	if idx := strings.Index(s, "("); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}

	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "0B" || s == "" {
		return 0
	}

	var multiplier int64 = 1
	switch {
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "TB"):
		multiplier = 1024 * 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "TB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}

	var value float64
	_ = json.Unmarshal([]byte(s), &value) // parse failure returns 0
	return int64(value * float64(multiplier))
}

func dockerTypeName(t string) string {
	switch strings.ToLower(t) {
	case "images":
		return "Docker Images"
	case "containers":
		return "Docker Containers"
	case "local volumes":
		return "Docker Volumes [!DB DATA RISK]"
	case "build cache":
		return "Docker Build Cache"
	default:
		return "Docker " + t
	}
}
