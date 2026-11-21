package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

const defaultItemLimit = 20

// Scan statuses reported per target.
const (
	statusOK          = "ok"
	statusUnavailable = "unavailable"
	statusBlocked     = "blocked"
	statusError       = "error"
)

// scanned is one target's scan outcome. result is nil unless status is ok.
type scanned struct {
	cat       types.Category
	result    *types.ScanResult
	status    string
	blockedBy string
	err       error
}

// scanCategories scans in parallel and returns results in input order.
func scanCategories(registry *target.Registry, cats []types.Category, cutoff time.Time) []scanned {
	out := make([]scanned, len(cats))
	sem := make(chan struct{}, utils.DefaultWorkers())
	var wg sync.WaitGroup
	for i, cat := range cats {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out[i] = scanOne(registry, cat, cutoff)
		}()
	}
	wg.Wait()
	return out
}

func scanOne(registry *target.Registry, cat types.Category, cutoff time.Time) scanned {
	s := scanned{cat: cat, status: statusOK}
	for _, proc := range cat.BlockedByProcesses {
		if utils.IsProcessRunning(proc) {
			s.status, s.blockedBy = statusBlocked, proc
			return s
		}
	}
	tgt, ok := registry.Get(cat.ID)
	if !ok {
		s.status, s.err = statusError, fmt.Errorf("no scanner registered")
		return s
	}
	if !tgt.IsAvailable() {
		s.status = statusUnavailable
		return s
	}
	result, err := target.ScanBefore(tgt, cutoff)
	if err != nil || result == nil {
		s.status, s.err = statusError, err
		if s.err == nil {
			s.err = fmt.Errorf("scan returned no result")
		}
		return s
	}
	if result.Error != nil {
		s.status, s.err = statusError, result.Error
		return s
	}
	s.result = result
	return s
}

type itemInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	FileCount   int64     `json:"file_count,omitempty"`
	ModifiedAt  time.Time `json:"modified_at,omitzero"`
	Locked      bool      `json:"locked,omitempty"`
	Excluded    bool      `json:"excluded,omitempty"`
}

type scanEntry struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Safety         types.SafetyLevel   `json:"safety"`
	Method         types.CleanupMethod `json:"method"`
	Status         string              `json:"status"`
	SizeBytes      int64               `json:"size_bytes"`
	CleanableBytes int64               `json:"cleanable_bytes"`
	ItemCount      int                 `json:"item_count"`
	BlockedBy      string              `json:"blocked_by,omitempty"`
	Error          string              `json:"error,omitempty"`
	Guide          string              `json:"guide,omitempty"`
	Items          []itemInfo          `json:"items,omitempty"`
	MoreItems      int                 `json:"more_items,omitempty"`
}

type scanOutput struct {
	Targets        []scanEntry `json:"targets"`
	SizeBytes      int64       `json:"size_bytes"`
	CleanableBytes int64       `json:"cleanable_bytes"`
}

func (a *App) runScan(args []string) error {
	fs := newFlagSet("scan")
	var filter targetFilter
	filter.register(fs)
	age := registerOlderThan(fs)
	withItems := fs.Bool("items", false, "")
	limit := fs.Int("limit", defaultItemLimit, "")
	jsonOutput := fs.Bool("json", false, "")
	ids, err := parseArgs(fs, args)
	if err != nil {
		return a.flagError("scan", err)
	}
	if *limit < 0 {
		return usageErrorf("invalid_flag", "use 0 for no limit", "--limit must not be negative")
	}
	filter.ids = ids

	sess, err := openSession()
	if err != nil {
		return err
	}
	cats, err := filter.apply(sess.cfg)
	if err != nil {
		return err
	}

	out := buildScanOutput(scanCategories(sess.registry, cats, ageCutoff(*age)), sess.userCfg, *withItems, *limit)
	if *jsonOutput {
		return writeJSON(a.Out, out)
	}
	a.printScan(out, *withItems, *age)
	return nil
}

func buildScanOutput(results []scanned, userCfg *userconfig.UserConfig, withItems bool, limit int) scanOutput {
	out := scanOutput{Targets: make([]scanEntry, 0, len(results))}
	for _, s := range results {
		entry := newScanEntry(s, userCfg, withItems, limit)
		out.SizeBytes += entry.SizeBytes
		out.CleanableBytes += entry.CleanableBytes
		out.Targets = append(out.Targets, entry)
	}
	// Stable sort keeps catalog order among equal sizes.
	slices.SortStableFunc(out.Targets, func(x, y scanEntry) int {
		return cmp.Compare(y.SizeBytes, x.SizeBytes)
	})
	return out
}

func newScanEntry(s scanned, userCfg *userconfig.UserConfig, withItems bool, limit int) scanEntry {
	entry := scanEntry{
		ID: s.cat.ID, Name: s.cat.Name, Safety: s.cat.Safety, Method: s.cat.Method,
		Status: s.status, BlockedBy: s.blockedBy,
	}
	if s.err != nil {
		entry.Error = s.err.Error()
	}
	if s.cat.Method == types.MethodManual {
		entry.Guide = strings.TrimSpace(s.cat.Guide)
	}
	if s.result == nil {
		return entry
	}

	entry.SizeBytes = s.result.TotalSize
	entry.ItemCount = len(s.result.Items)
	autoClean := s.cat.Method != types.MethodManual && s.cat.Safety != types.SafetyLevelRisky

	items := slices.Clone(s.result.Items)
	slices.SortStableFunc(items, func(x, y types.CleanableItem) int { return cmp.Compare(y.Size, x.Size) })
	for _, item := range items {
		locked := item.Status == types.ItemStatusProcessLocked
		excluded := userCfg.IsExcluded(s.cat.ID, item.Path)
		if autoClean && !locked && !excluded {
			entry.CleanableBytes += item.Size
		}
		if !withItems {
			continue
		}
		if limit > 0 && len(entry.Items) >= limit {
			entry.MoreItems++
			continue
		}
		entry.Items = append(entry.Items, itemInfo{
			Path: item.Path, SizeBytes: item.Size, FileCount: item.FileCount,
			Name: item.Name, DisplayName: item.DisplayName, ModifiedAt: item.ModifiedAt, Locked: locked, Excluded: excluded,
		})
	}
	return entry
}

func (a *App) printScan(out scanOutput, withItems bool, age time.Duration) {
	tw := tabwriter.NewWriter(a.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSAFETY\tMETHOD\tITEMS\tSIZE\tNAME")
	var empty, unavailable int
	var notes []string
	for _, t := range out.Targets {
		switch {
		case t.Status == statusBlocked:
			notes = append(notes, fmt.Sprintf("%s: blocked while %s is running", t.ID, t.BlockedBy))
			continue
		case t.Status == statusError:
			notes = append(notes, fmt.Sprintf("%s: scan failed: %s", t.ID, t.Error))
			continue
		case t.Status == statusUnavailable:
			unavailable++
			continue
		case t.ItemCount == 0:
			empty++
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
			t.ID, t.Safety, t.Method, t.ItemCount, utils.FormatSize(t.SizeBytes), t.Name)
		if withItems {
			a.printItems(tw, t)
		}
	}
	tw.Flush()

	fmt.Fprintf(a.Out, "\nCleanable: %s of %s found\n", utils.FormatSize(out.CleanableBytes), utils.FormatSize(out.SizeBytes))
	if review := reviewBytes(out.Targets); review > 0 {
		fmt.Fprintf(a.Out, "Review first: %s in risky or manual targets (clean needs --item or manual steps)\n", utils.FormatSize(review))
	}
	if empty+unavailable > 0 {
		fmt.Fprintf(a.Out, "Hidden: %d empty, %d not installed\n", empty, unavailable)
	}
	for _, n := range notes {
		fmt.Fprintln(a.Out, "Note: "+n)
	}
	next := "Next: mac-cleanup clean <id>... --dry-run"
	if age >= 0 {
		next += fmt.Sprintf(" --older-than %dd", age/day)
	}
	fmt.Fprintln(a.Out, next)
}

func reviewBytes(targets []scanEntry) int64 {
	var total int64
	for _, t := range targets {
		if t.Safety == types.SafetyLevelRisky || t.Method == types.MethodManual {
			total += t.SizeBytes
		}
	}
	return total
}

func (a *App) printItems(tw *tabwriter.Writer, t scanEntry) {
	for _, item := range t.Items {
		var tags []string
		if item.Locked {
			tags = append(tags, "locked")
		}
		if item.Excluded {
			tags = append(tags, "excluded")
		}
		tag := ""
		if len(tags) > 0 {
			tag = " [" + strings.Join(tags, ", ") + "]"
		}
		fmt.Fprintf(tw, "\t\t\t\t%s\t  %4s  %s%s\n",
			utils.FormatSize(item.SizeBytes), utils.FormatAge(item.ModifiedAt), item.Path, tag)
	}
	if t.MoreItems > 0 {
		fmt.Fprintf(tw, "\t\t\t\t\t  ... %d more (--limit 0 shows all)\n", t.MoreItems)
	}
}
