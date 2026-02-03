package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/cleaner"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

var (
	ErrNoSelection         = errors.New("no targets selected")
	ErrNoEligibleTargets   = errors.New("no eligible targets selected")
	ErrNilRunnerConfig     = errors.New("runner config is nil")
	ErrNilRunnerRegistry   = errors.New("runner registry is nil")
	ErrNilRunnerUserConfig = errors.New("runner user config is nil")
)

// Runner executes CLI clean or dry-run using stored user config.
type Runner struct {
	cfg      *types.Config
	registry *target.Registry
	userCfg  *userconfig.UserConfig
}

// NewRunner creates a Runner with a default registry.
func NewRunner(cfg *types.Config, userCfg *userconfig.UserConfig) (*Runner, error) {
	if cfg == nil {
		return nil, ErrNilRunnerConfig
	}
	registry, err := target.DefaultRegistry(cfg)
	if err != nil {
		return nil, err
	}
	return &Runner{cfg: cfg, registry: registry, userCfg: userCfg}, nil
}

// NewRunnerWithRegistry creates a Runner with a custom registry (for tests).
func NewRunnerWithRegistry(cfg *types.Config, registry *target.Registry, userCfg *userconfig.UserConfig) *Runner {
	return &Runner{cfg: cfg, registry: registry, userCfg: userCfg}
}

// Run executes a dry run or actual clean and returns a report and warnings.
func (r *Runner) Run(dryRun bool) (*types.Report, []string, error) {
	if r.cfg == nil {
		return nil, nil, ErrNilRunnerConfig
	}
	if r.registry == nil {
		return nil, nil, ErrNilRunnerRegistry
	}
	if r.userCfg == nil {
		return nil, nil, ErrNilRunnerUserConfig
	}

	selectedIDs := r.userCfg.GetSelectedTargets()
	if len(selectedIDs) == 0 {
		return nil, nil, ErrNoSelection
	}

	selectedSet := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selectedSet[id] = true
	}

	resultMap := make(map[string]*types.ScanResult)
	selected := make(map[string]bool)
	var selectedOrder []string
	var warnings []string

	for _, cat := range r.cfg.Categories {
		if !selectedSet[cat.ID] {
			continue
		}
		if cat.Safety == types.SafetyLevelRisky {
			warnings = append(warnings, fmt.Sprintf("skipped risky target: %s", cat.Name))
			continue
		}
		if cat.Method == types.MethodManual {
			warnings = append(warnings, fmt.Sprintf("skipped manual target: %s", cat.Name))
			continue
		}

		tgt, ok := r.registry.Get(cat.ID)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("missing target: %s", cat.Name))
			continue
		}

		result, err := tgt.Scan()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("scan failed: %s (%v)", cat.Name, err))
		}
		if result == nil {
			continue
		}
		resultMap[cat.ID] = result
		selected[cat.ID] = true
		selectedOrder = append(selectedOrder, cat.ID)
	}

	if len(selected) == 0 {
		return nil, warnings, ErrNoEligibleTargets
	}

	cleanService := cleaner.NewCleanService(r.registry)
	jobs := cleanService.PrepareJobsWithOrder(resultMap, selected, r.userCfg.ExcludedPathsMap(), selectedOrder)

	start := time.Now()
	var report *types.Report
	if dryRun {
		report = buildDryRunReport(jobs)
	} else {
		report = cleanService.Clean(jobs, types.CleanCallbacks{})
	}
	report.Duration = time.Since(start)

	return report, warnings, nil
}

func buildDryRunReport(jobs []cleaner.CleanJob) *types.Report {
	report := &types.Report{Results: make([]types.CleanResult, 0, len(jobs))}

	for _, job := range jobs {
		result := types.NewCleanResult(job.Category)
		for _, item := range job.Items {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
		report.FreedSpace += result.FreedSpace
		report.CleanedItems += result.CleanedItems
		report.Results = append(report.Results, *result)
	}

	return report
}
