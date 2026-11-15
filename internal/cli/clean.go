package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/2ykwang/mac-cleanup-go/internal/cleaner"
	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// Skip reasons explain why a requested target was not cleaned.
const (
	skipManual        = "manual"
	skipRiskyNeedItem = "risky_needs_item"
	skipBlocked       = "blocked"
	skipUnavailable   = "unavailable"
	skipScanFailed    = "scan_failed"
	skipNothing       = "nothing_to_clean"
	skipItemLocked    = "item_locked"
	skipUnknownSaved  = "unknown_target"
)

// Per-target clean statuses.
const (
	cleanPlanned = "planned"
	cleanOK      = "ok"
	cleanPartial = "partial"
	cleanFailed  = "failed"
)

type skippedTarget struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
	Path   string `json:"path,omitempty"`
}

type cleanEntry struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Method      types.CleanupMethod `json:"method"`
	Status      string              `json:"status"`
	Items       int                 `json:"items"`
	FailedItems int                 `json:"failed_items,omitempty"`
	SIPSkipped  int                 `json:"sip_skipped_items,omitempty"`
	FreedBytes  int64               `json:"freed_bytes"`
	SideEffect  string              `json:"side_effect,omitempty"`
	Errors      []string            `json:"errors,omitempty"`
	ItemPaths   []string            `json:"item_paths,omitempty"`
}

type cleanOutput struct {
	DryRun       bool            `json:"dry_run"`
	FreedBytes   int64           `json:"freed_bytes"`
	CleanedItems int             `json:"cleaned_items"`
	FailedItems  int             `json:"failed_items"`
	Targets      []cleanEntry    `json:"targets"`
	Skipped      []skippedTarget `json:"skipped"`
}

type cleanOptions struct {
	ids      []string
	items    []string
	selected bool
	dryRun   bool
	yes      bool
	json     bool
}

func (a *App) runClean(args []string) error {
	opts, err := a.parseCleanOptions(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return a.runHelp([]string{"clean"})
		}
		return err
	}
	interactive := isTerminal(a.In) && !opts.json
	// Fail before scanning: an unattended caller should not wait for a scan
	// only to learn it cannot proceed.
	if !opts.dryRun && !opts.yes && !interactive {
		return usageErrorf("confirmation_required",
			"preview with --dry-run, get the user's approval, then re-run with --yes",
			"clean changes files and needs --yes when not run from a terminal")
	}

	plan, svc, err := prepareClean(opts)
	if err != nil {
		return err
	}
	jobs := plan.jobs(svc)
	if opts.dryRun || len(jobs) == 0 {
		return a.writeCleanResult(opts, plan, buildDryRunReport(jobs), opts.dryRun)
	}

	if !opts.yes {
		a.writeCleanText(plan, buildDryRunReport(jobs), true)
		if !a.confirm(plan, jobs) {
			return runtimeErrorf("cancelled", "", "cancelled; nothing was changed")
		}
		// Re-check blocking processes: time passed while the user read the plan.
		jobs = plan.jobs(svc)
	}

	start := time.Now()
	report := svc.Clean(jobs, types.CleanCallbacks{})
	report.Duration = time.Since(start)
	if err := a.writeCleanResult(opts, plan, report, false); err != nil {
		return err
	}
	if report.FailedItems > 0 {
		return &partialError{failed: report.FailedItems}
	}
	return nil
}

// prepareClean scans the requested targets and narrows them to a plan.
func prepareClean(opts cleanOptions) (*cleanPlan, *cleaner.CleanService, error) {
	sess, err := openSession()
	if err != nil {
		return nil, nil, err
	}
	ids, skipped, err := resolveCleanIDs(opts, sess.cfg, sess.userCfg)
	if err != nil {
		return nil, nil, err
	}
	cats, err := (&targetFilter{ids: ids}).apply(sess.cfg)
	if err != nil {
		return nil, nil, err
	}

	plan, err := planClean(scanCategories(sess.registry, cats), sess.userCfg, opts.items)
	if err != nil {
		return nil, nil, err
	}
	plan.skipped = append(skipped, plan.skipped...)
	return plan, cleaner.NewCleanService(sess.registry), nil
}

func (a *App) parseCleanOptions(args []string) (cleanOptions, error) {
	fs := newFlagSet("clean")
	var opts cleanOptions
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.yes, "yes", false, "")
	fs.BoolVar(&opts.selected, "selected", false, "")
	fs.BoolVar(&opts.json, "json", false, "")
	fs.Var((*stringList)(&opts.items), "item", "")
	ids, err := parseArgs(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, err
		}
		return opts, a.flagError("clean", err)
	}
	opts.ids = ids

	switch {
	case opts.selected && len(ids) > 0:
		return opts, usageErrorf("invalid_flag", "pass target IDs or --selected", "--selected cannot be combined with target IDs")
	case !opts.selected && len(ids) == 0:
		return opts, usageErrorf("missing_target", "run `mac-cleanup scan` to find target IDs",
			"clean needs at least one target ID")
	}
	return opts, nil
}

// resolveCleanIDs expands --selected. Saved IDs that no longer exist are
// reported as skipped rather than failing the whole run.
func resolveCleanIDs(opts cleanOptions, cfg *types.Config, userCfg *userconfig.UserConfig) ([]string, []skippedTarget, error) {
	if !opts.selected {
		return opts.ids, nil, nil
	}
	saved := userCfg.GetSelectedTargets()
	if len(saved) == 0 {
		return nil, nil, runtimeErrorf("no_selection", "run `mac-cleanup select` in a terminal, or pass target IDs",
			"no targets saved for --selected")
	}
	var ids []string
	var skipped []skippedTarget
	for _, id := range saved {
		if slices.ContainsFunc(cfg.Categories, func(c types.Category) bool { return c.ID == id }) {
			ids = append(ids, id)
			continue
		}
		skipped = append(skipped, skippedTarget{ID: id, Reason: skipUnknownSaved, Detail: "saved target no longer exists"})
	}
	// An empty ID list means "every target" to targetFilter, so stop here.
	if len(ids) == 0 {
		return nil, nil, runtimeErrorf("no_selection", "run `mac-cleanup select` in a terminal, or pass target IDs",
			"none of the saved targets exist anymore")
	}
	return ids, skipped, nil
}

// cleanPlan holds the scan results narrowed to what will be cleaned.
type cleanPlan struct {
	results     map[string]*types.ScanResult
	order       []string
	skipped     []skippedTarget
	sideEffects map[string]string
}

// jobs builds jobs through the cleaner so its last-moment process check
// applies. Targets it drops are reported as blocked.
func (p *cleanPlan) jobs(svc *cleaner.CleanService) []cleaner.CleanJob {
	selected := make(map[string]bool, len(p.order))
	for _, id := range p.order {
		selected[id] = true
	}
	jobs := svc.PrepareJobsWithOrder(p.results, selected, nil, p.order)
	if len(jobs) == len(p.order) {
		return jobs
	}

	kept := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		kept[job.Category.ID] = true
	}
	for _, id := range p.order {
		if !kept[id] {
			p.skipped = append(p.skipped, skippedTarget{ID: id, Reason: skipBlocked, Detail: "a blocking process started"})
		}
	}
	p.order = slices.DeleteFunc(p.order, func(id string) bool { return !kept[id] })
	return jobs
}

func planClean(results []scanned, userCfg *userconfig.UserConfig, itemPaths []string) (*cleanPlan, error) {
	wanted := make(map[string]bool, len(itemPaths))
	for _, p := range itemPaths {
		wanted[cleanItemPath(p)] = true
	}
	found := make(map[string]bool, len(wanted))

	plan := &cleanPlan{results: make(map[string]*types.ScanResult), sideEffects: make(map[string]string)}
	var unscanned []string
	for _, s := range results {
		if skip, ok := skipBeforeItems(s, len(wanted) > 0); ok {
			plan.skipped = append(plan.skipped, skip)
			unscanned = append(unscanned, skip.ID+" ("+skip.Reason+")")
			continue
		}

		if len(wanted) > 0 && s.sideEffect != "" && hasWantedItem(s.result.Items, wanted) {
			return nil, usageErrorf("item_not_supported",
				"clean the whole target after the user approves it: `mac-cleanup clean "+s.cat.ID+" --dry-run`",
				"%s cannot be cleaned item by item: it %s", s.cat.ID, s.sideEffect)
		}

		picked, skipped := pickItems(s, userCfg, wanted, found)
		plan.skipped = append(plan.skipped, skipped...)
		if len(picked) == 0 {
			plan.skipped = append(plan.skipped, skippedTarget{ID: s.cat.ID, Reason: skipNothing})
			continue
		}
		narrowed := *s.result
		narrowed.Items = picked
		plan.results[s.cat.ID] = &narrowed
		plan.order = append(plan.order, s.cat.ID)
		if s.sideEffect != "" {
			plan.sideEffects[s.cat.ID] = s.sideEffect
		}
	}

	for _, p := range itemPaths {
		if found[cleanItemPath(p)] {
			continue
		}
		hint := "use exact paths from `mac-cleanup scan <id> --items --limit 0 --json` and include that target ID"
		if len(unscanned) > 0 {
			hint += "; not scanned: " + strings.Join(unscanned, ", ")
		}
		return nil, usageErrorf("unknown_item", hint, "%q is not a scanned item of the given targets", p)
	}
	return plan, nil
}

func hasWantedItem(items []types.CleanableItem, wanted map[string]bool) bool {
	return slices.ContainsFunc(items, func(item types.CleanableItem) bool { return wanted[item.Path] })
}

func cleanItemPath(p string) string {
	return filepath.Clean(utils.ExpandPath(p))
}

// pickItems returns the items to clean. With --item paths only those are
// picked, overriding saved exclusions; locked items are never picked.
func pickItems(s scanned, userCfg *userconfig.UserConfig, wanted, found map[string]bool) ([]types.CleanableItem, []skippedTarget) {
	var picked []types.CleanableItem
	var skipped []skippedTarget
	for _, item := range s.result.Items {
		if len(wanted) > 0 && !wanted[item.Path] {
			continue
		}
		found[item.Path] = true
		if item.Status == types.ItemStatusProcessLocked {
			if len(wanted) > 0 {
				skipped = append(skipped, skippedTarget{
					ID: s.cat.ID, Reason: skipItemLocked, Path: item.Path, Detail: "in use by a running process",
				})
			}
			continue
		}
		if len(wanted) == 0 && userCfg.IsExcluded(s.cat.ID, item.Path) {
			continue
		}
		picked = append(picked, item)
	}
	return picked, skipped
}

// skipBeforeItems reports targets that are skipped regardless of their items.
func skipBeforeItems(s scanned, hasItemFilter bool) (skippedTarget, bool) {
	skip := skippedTarget{ID: s.cat.ID}
	switch {
	case s.cat.Method == types.MethodManual:
		skip.Reason, skip.Detail = skipManual, "clean by hand; see `mac-cleanup targets "+s.cat.ID+"`"
	case s.status == statusBlocked:
		skip.Reason, skip.Detail = skipBlocked, "quit "+s.blockedBy+" first"
	case s.status == statusUnavailable:
		skip.Reason, skip.Detail = skipUnavailable, "not installed or nothing to scan"
	case s.status == statusError:
		skip.Reason, skip.Detail = skipScanFailed, s.err.Error()
	case s.cat.Safety == types.SafetyLevelRisky && !hasItemFilter:
		skip.Reason = skipRiskyNeedItem
		skip.Detail = "may hold user data; pass --item <path> for each approved item (see `mac-cleanup scan " + s.cat.ID + " --items`)"
	default:
		return skippedTarget{}, false
	}
	return skip, true
}

func (a *App) confirm(plan *cleanPlan, jobs []cleaner.CleanJob) bool {
	var permanent []string
	for _, job := range jobs {
		if job.Category.Method == types.MethodPermanent {
			permanent = append(permanent, job.Category.ID)
		}
	}
	fmt.Fprintln(a.Err)
	if len(permanent) > 0 {
		fmt.Fprintf(a.Err, "Permanently deletes (not recoverable): %s\n", strings.Join(permanent, ", "))
	}
	for _, job := range jobs {
		if effect := plan.sideEffects[job.Category.ID]; effect != "" {
			fmt.Fprintf(a.Err, "Also: %s %s\n", job.Category.ID, effect)
		}
	}
	fmt.Fprint(a.Err, "Proceed? [y/N] ")
	line, _ := bufio.NewReader(a.In).ReadString('\n') // EOF counts as "no"
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func (a *App) writeCleanResult(opts cleanOptions, plan *cleanPlan, report *types.Report, dryRun bool) error {
	if opts.json {
		return writeJSON(a.Out, newCleanOutput(report, plan, dryRun))
	}
	a.writeCleanText(plan, report, dryRun)
	return nil
}

func (a *App) writeCleanText(plan *cleanPlan, report *types.Report, dryRun bool) {
	color := colorEnabled(a.Out)
	theme := styles.New(true)
	if color {
		// CLI path has no event loop, so detect background once up front.
		theme = styles.New(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))
	}
	fmt.Fprint(a.Out, FormatReport(report, dryRun, theme, color))
	for _, r := range report.Results {
		if effect := plan.sideEffects[r.Category.ID]; effect != "" {
			fmt.Fprintf(a.Out, "\nNote: %s %s\n", r.Category.ID, effect)
		}
	}
	if len(plan.skipped) > 0 {
		fmt.Fprintln(a.Out, "\nSkipped")
		for _, s := range plan.skipped {
			line := fmt.Sprintf("  %s: %s", s.ID, s.Reason)
			if s.Path != "" {
				line += " " + s.Path
			}
			if s.Detail != "" {
				line += " (" + s.Detail + ")"
			}
			fmt.Fprintln(a.Out, line)
		}
	}
	if dryRun && report.CleanedItems > 0 {
		fmt.Fprintln(a.Out, "\nNothing was changed. Re-run with --yes instead of --dry-run to clean.")
	}
}

func newCleanOutput(report *types.Report, plan *cleanPlan, dryRun bool) cleanOutput {
	out := cleanOutput{
		DryRun:       dryRun,
		FreedBytes:   report.FreedSpace,
		CleanedItems: report.CleanedItems,
		FailedItems:  report.FailedItems,
		Targets:      make([]cleanEntry, 0, len(report.Results)),
		Skipped:      plan.skipped,
	}
	if out.Skipped == nil {
		out.Skipped = []skippedTarget{}
	}
	for _, r := range report.Results {
		entry := cleanEntry{
			ID:          r.Category.ID,
			Name:        r.Category.Name,
			Method:      r.Category.Method,
			Status:      cleanStatus(r, dryRun),
			Items:       r.CleanedItems,
			FailedItems: len(r.Errors),
			SIPSkipped:  r.SkippedItems,
			FreedBytes:  r.FreedSpace,
			SideEffect:  plan.sideEffects[r.Category.ID],
			Errors:      r.Errors,
		}
		if dryRun {
			for _, item := range plan.results[r.Category.ID].Items {
				entry.ItemPaths = append(entry.ItemPaths, item.Path)
			}
		}
		out.Targets = append(out.Targets, entry)
	}
	return out
}

func cleanStatus(r types.CleanResult, dryRun bool) string {
	switch {
	case dryRun:
		return cleanPlanned
	case len(r.Errors) == 0:
		return cleanOK
	case r.CleanedItems > 0:
		return cleanPartial
	default:
		return cleanFailed
	}
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
