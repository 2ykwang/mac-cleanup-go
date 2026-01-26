package target

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type DockerTarget struct {
	category types.Category
}

const (
	dockerShortIDLength = 12
	dockerNameLimit     = 50
)

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

type dockerDfVerbose struct {
	Images     []dockerDfImage     `json:"Images"`
	Containers []dockerDfContainer `json:"Containers"`
	Volumes    []dockerDfVolume    `json:"Volumes"`
}

type dockerDfImage struct {
	ID         string `json:"ID"`
	Repository string `json:"Repository"`
	Tag        string `json:"Tag"`
	Size       string `json:"Size"`
	UniqueSize string `json:"UniqueSize"`
}

type dockerDfContainer struct {
	Image  string `json:"Image"`
	Names  string `json:"Names"`
	Mounts string `json:"Mounts"`
}

type dockerDfVolume struct {
	Name string `json:"Name"`
	Size string `json:"Size"`
}

func (s *DockerTarget) Scan() (*types.ScanResult, error) {
	result := types.NewScanResult(s.category)

	if !s.IsAvailable() {
		return result, nil
	}

	verbose, err := s.fetchVerboseDf()
	if err != nil {
		result.Error = err
		return result, nil
	}

	buildCacheSize := s.fetchBuildCacheSize()

	volumeSet := make(map[string]struct{}, len(verbose.Volumes))
	for _, v := range verbose.Volumes {
		if v.Name != "" {
			volumeSet[v.Name] = struct{}{}
		}
	}

	imageUsedBy := make(map[string]map[string]struct{})
	volumeUsedBy := make(map[string]map[string]struct{})

	for _, c := range verbose.Containers {
		name := strings.TrimPrefix(strings.TrimSpace(c.Names), "/")
		if name == "" {
			continue
		}
		imageRef := strings.TrimSpace(c.Image)
		if imageRef != "" {
			addUsedBy(imageUsedBy, imageRef, name)
			if !dockerHasTag(imageRef) {
				addUsedBy(imageUsedBy, imageRef+":latest", name)
			}
		}
		if c.Mounts != "" {
			for _, mount := range strings.Split(c.Mounts, ",") {
				mount = strings.TrimSpace(mount)
				if mount == "" {
					continue
				}
				if _, ok := volumeSet[mount]; ok {
					addUsedBy(volumeUsedBy, mount, name)
				}
			}
		}
	}

	type dockerImageAggregate struct {
		ID   string
		Size int64
		Tags map[string]struct{}
	}

	imageAggregates := make(map[string]*dockerImageAggregate, len(verbose.Images))
	for _, img := range verbose.Images {
		if img.ID == "" {
			continue
		}

		var size int64
		if img.UniqueSize != "" {
			size = parseDockerSize(img.UniqueSize)
		} else {
			size = parseDockerSize(img.Size)
		}

		if size == 0 {
			continue
		}

		agg := imageAggregates[img.ID]
		if agg == nil {
			agg = &dockerImageAggregate{
				ID:   img.ID,
				Size: size,
				Tags: make(map[string]struct{}),
			}
			imageAggregates[img.ID] = agg
		} else if size > agg.Size {
			agg.Size = size
		}

		repo := strings.TrimSpace(img.Repository)
		tag := strings.TrimSpace(img.Tag)
		if repo != "" && repo != "<none>" && tag != "" && tag != "<none>" {
			agg.Tags[repo+":"+tag] = struct{}{}
		}
	}

	imageIDs := make([]string, 0, len(imageAggregates))
	for id := range imageAggregates {
		imageIDs = append(imageIDs, id)
	}
	sort.Strings(imageIDs)

	for _, imageID := range imageIDs {
		agg := imageAggregates[imageID]
		tags := make([]string, 0, len(agg.Tags))
		for tag := range agg.Tags {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		var label string
		if len(tags) > 0 {
			label = tags[0]
		} else {
			shortID := strings.TrimPrefix(imageID, "sha256:")
			if shortID == "" {
				shortID = "unknown"
			}
			if len(shortID) > dockerShortIDLength {
				shortID = shortID[:dockerShortIDLength]
			}
			label = "untagged@" + shortID
		}

		baseName := "Image: " + truncateName(label, dockerNameLimit)
		if len(tags) > 1 {
			baseName = fmt.Sprintf("%s (+%d tags)", baseName, len(tags)-1)
		}

		combined := make(map[string]struct{})
		for _, key := range []string{imageID, strings.TrimPrefix(imageID, "sha256:")} {
			for name := range imageUsedBy[key] {
				combined[name] = struct{}{}
			}
		}
		for _, tag := range tags {
			for name := range imageUsedBy[tag] {
				combined[name] = struct{}{}
			}
		}
		usedBy := usedByList(combined)
		displayName := appendUsedBy(baseName, usedBy)

		item := types.CleanableItem{
			Path:        "docker:image:" + imageID,
			Size:        agg.Size,
			FileCount:   1,
			Name:        baseName,
			DisplayName: displayName,
			IsDirectory: false,
		}
		if len(usedBy) > 0 {
			item.Status = types.ItemStatusProcessLocked
		}
		result.Items = append(result.Items, item)
		result.TotalSize += agg.Size
		result.TotalFileCount++
	}

	for _, v := range verbose.Volumes {
		if v.Name == "" {
			continue
		}
		size := parseDockerSize(v.Size)
		if size == 0 {
			continue
		}
		baseName := "Volume: " + truncateName(v.Name, dockerNameLimit)
		usedBy := usedByList(volumeUsedBy[v.Name])
		displayName := appendUsedBy(baseName, usedBy)

		item := types.CleanableItem{
			Path:        "docker:volume:" + v.Name,
			Size:        size,
			FileCount:   1,
			Name:        baseName,
			DisplayName: displayName,
			IsDirectory: false,
		}
		if len(usedBy) > 0 {
			item.Status = types.ItemStatusProcessLocked
		}
		result.Items = append(result.Items, item)
		result.TotalSize += size
		result.TotalFileCount++
	}

	if buildCacheSize > 0 {
		name := "Docker Build Cache"
		item := types.CleanableItem{
			Path:        "docker:build-cache",
			Size:        buildCacheSize,
			FileCount:   1,
			Name:        name,
			DisplayName: name,
			IsDirectory: false,
		}
		result.Items = append(result.Items, item)
		result.TotalSize += buildCacheSize
		result.TotalFileCount++
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
		switch {
		case strings.HasPrefix(item.Path, "docker:image:"):
			imageID := strings.TrimPrefix(item.Path, "docker:image:")
			if imageID != "" {
				cmd = execCommand("docker", "image", "rm", imageID)
			} else {
				logger.Debug("docker prune skipped empty image id", "path", item.Path)
			}
		case strings.HasPrefix(item.Path, "docker:volume:"):
			volumeName := strings.TrimPrefix(item.Path, "docker:volume:")
			if volumeName != "" {
				cmd = execCommand("docker", "volume", "rm", volumeName)
			} else {
				logger.Debug("docker prune skipped empty volume name", "path", item.Path)
			}
		case item.Path == "docker:build-cache":
			cmd = execCommand("docker", "builder", "prune", "-af")
		default:
			logger.Debug("docker prune skipped unknown path", "path", item.Path)
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

func (s *DockerTarget) fetchVerboseDf() (*dockerDfVerbose, error) {
	cmd := execCommand("docker", "system", "df", "-v", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("docker system df -v failed", "error", err)
		return nil, err
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return &dockerDfVerbose{}, nil
	}

	var df dockerDfVerbose
	if err := json.Unmarshal([]byte(line), &df); err != nil {
		logger.Warn("docker df -v json parse failed", "error", err)
		return nil, err
	}
	return &df, nil
}

func (s *DockerTarget) fetchBuildCacheSize() int64 {
	cmd := execCommand("docker", "system", "df", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("docker system df failed", "error", err)
		return 0
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
		if strings.EqualFold(df.Type, "Build Cache") {
			return parseDockerSize(df.Reclaimable)
		}
	}
	return 0
}

func addUsedBy(store map[string]map[string]struct{}, key, name string) {
	if key == "" || name == "" {
		return
	}
	if _, ok := store[key]; !ok {
		store[key] = make(map[string]struct{})
	}
	store[key][name] = struct{}{}
}

func usedByList(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func appendUsedBy(name string, usedBy []string) string {
	if len(usedBy) == 0 {
		return name
	}
	first := usedBy[0]
	extra := len(usedBy) - 1
	if extra > 0 {
		return fmt.Sprintf("%s [Used By: %s (+%d)]", name, first, extra)
	}
	return fmt.Sprintf("%s [Used By: %s]", name, first)
}

func dockerHasTag(ref string) bool {
	if ref == "" {
		return false
	}
	if strings.Contains(ref, "@") {
		return true
	}
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	return lastColon > lastSlash
}

func truncateName(name string, limit int) string {
	if limit <= 0 {
		return name
	}
	runes := []rune(name)
	if len(runes) <= limit {
		return name
	}
	return string(runes[:limit])
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

	value, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(value * float64(multiplier))
}
