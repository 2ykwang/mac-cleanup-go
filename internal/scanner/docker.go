package scanner

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
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

// dockerImageInfo represents a single image from `docker image ls --format json`.
type dockerImageInfo struct {
	ID           string `json:"ID"`           // Short image ID (e.g., "3960ed74dfe3")
	Repository   string `json:"Repository"`   // Image repository (e.g., "nginx", "<none>")
	Tag          string `json:"Tag"`          // Image tag (e.g., "latest", "<none>")
	Size         string `json:"Size"`         // Human-readable size (e.g., "192MB")
	CreatedSince string `json:"CreatedSince"` // Relative time (e.g., "2 days ago")
	Containers   string `json:"Containers"`   // Number of containers using this image
}

// IsDangling returns true if the image has no repository and tag.
func (d *dockerImageInfo) IsDangling() bool {
	return d.Repository == "<none>" && d.Tag == "<none>"
}

// IsInUse returns true if any container is using this image.
func (d *dockerImageInfo) IsInUse() bool {
	count, _ := strconv.Atoi(d.Containers)
	return count > 0
}

func (s *DockerScanner) Scan() (*types.ScanResult, error) {
	result := &types.ScanResult{
		Category: s.category,
		Items:    make([]types.CleanableItem, 0),
	}

	if !s.IsAvailable() {
		return result, nil
	}

	cmd := exec.Command("docker", "image", "ls", "--format", "json")
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

		var img dockerImageInfo
		if err := json.Unmarshal([]byte(line), &img); err != nil {
			continue
		}

		size := parseDockerSize(img.Size)

		item := s.buildImageItem(img, size)
		result.Items = append(result.Items, item)
		result.TotalSize += size
		result.TotalFileCount++
	}

	return result, nil
}

// buildImageItem creates a CleanableItem from dockerImageInfo with SafetyHint and Columns.
func (s *DockerScanner) buildImageItem(img dockerImageInfo, size int64) types.CleanableItem {
	// Determine display name
	name := img.Repository
	if img.Tag != "<none>" {
		name += ":" + img.Tag
	}
	if img.IsDangling() {
		name = "<none>:<none>"
	}

	// Determine status and safety hint
	status := "unused"
	safetyHint := types.SafetyHintSafe
	selected := true

	if img.IsDangling() {
		status = "dangling"
	} else if img.IsInUse() {
		status = "in-use"
		safetyHint = types.SafetyHintWarning
		selected = false
	}

	return types.CleanableItem{
		Path:        img.ID, // Image ID for deletion
		Size:        size,
		FileCount:   1,
		Name:        name,
		IsDirectory: false,
		Columns: []types.Column{
			{Header: "Repository", Value: img.Repository},
			{Header: "Tag", Value: img.Tag},
			{Header: "Status", Value: status},
			{Header: "Created", Value: img.CreatedSince},
		},
		SafetyHint: safetyHint,
		Selected:   selected,
	}
}

func (s *DockerScanner) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	for _, item := range items {
		// item.Path contains the image ID (e.g., "abc123")
		cmd := exec.Command("docker", "rmi", item.Path)
		if err := cmd.Run(); err != nil {
			result.Errors = append(result.Errors, item.Name+": "+err.Error())
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
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
