package scanner

import (
	"encoding/json"
	"os/exec"
	"strings"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

type DockerScanner struct {
	category types.Category
}

func NewDockerScanner(cat types.Category) *DockerScanner {
	return &DockerScanner{category: cat}
}

func (s *DockerScanner) Category() types.Category {
	return s.category
}

func (s *DockerScanner) IsAvailable() bool {
	if !utils.CommandExists("docker") {
		return false
	}
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

type dockerDfOutput struct {
	Type        string `json:"Type"`
	TotalCount  string `json:"TotalCount"`
	Active      string `json:"Active"`
	Size        string `json:"Size"`
	Reclaimable string `json:"Reclaimable"`
}

func (s *DockerScanner) Scan() (*types.ScanResult, error) {
	result := &types.ScanResult{
		Category: s.category,
		Items:    make([]types.CleanableItem, 0),
	}

	if !s.IsAvailable() {
		return result, nil
	}

	cmd := exec.Command("docker", "system", "df", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
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
			continue
		}

		size := parseDockerSize(df.Reclaimable)
		if size == 0 {
			continue
		}

		item := types.CleanableItem{
			Path:        "docker:" + strings.ToLower(df.Type),
			Size:        size,
			Name:        dockerTypeName(df.Type),
			IsDirectory: false,
		}
		result.Items = append(result.Items, item)
		result.TotalSize += size
	}

	return result, nil
}

func (s *DockerScanner) Clean(items []types.CleanableItem, dryRun bool) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	if dryRun {
		for _, item := range items {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
		return result, nil
	}

	for _, item := range items {
		var cmd *exec.Cmd
		switch item.Path {
		case "docker:images":
			cmd = exec.Command("docker", "image", "prune", "-af")
		case "docker:containers":
			cmd = exec.Command("docker", "container", "prune", "-f")
		case "docker:local volumes":
			cmd = exec.Command("docker", "volume", "prune", "-af")
		case "docker:build cache":
			cmd = exec.Command("docker", "builder", "prune", "-af")
		}

		if cmd != nil {
			if err := cmd.Run(); err != nil {
				result.Errors = append(result.Errors, err.Error())
			} else {
				result.FreedSpace += item.Size
				result.CleanedItems++
			}
		}
	}

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
