package cli

import (
	"flag"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

type targetInfo struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Group     string              `json:"group"`
	Safety    types.SafetyLevel   `json:"safety"`
	Method    types.CleanupMethod `json:"method"`
	Note      string              `json:"note,omitempty"`
	Paths     []string            `json:"paths,omitempty"`
	BlockedBy []string            `json:"blocked_by_processes,omitempty"`
	Guide     string              `json:"guide,omitempty"`
	Disposal  types.DisposalMode  `json:"disposal"`
}

type targetsOutput struct {
	Targets []targetInfo `json:"targets"`
}

// targetFilter narrows the catalog by explicit IDs, group, and safety.
type targetFilter struct {
	ids    []string
	group  string
	safety []string
}

func (f *targetFilter) register(fs *flag.FlagSet) {
	fs.StringVar(&f.group, "group", "", "")
	fs.Var((*stringList)(&f.safety), "safety", "")
}

// apply returns matching categories in catalog order. Unknown IDs, groups, and
// safety levels are usage errors so a typo never silently selects nothing.
func (f *targetFilter) apply(cfg *types.Config) ([]types.Category, error) {
	byID := make(map[string]types.Category, len(cfg.Categories))
	for _, cat := range cfg.Categories {
		byID[cat.ID] = cat
	}
	for _, id := range f.ids {
		if _, ok := byID[id]; !ok {
			return nil, usageErrorf("unknown_target", "run `mac-cleanup targets` to list IDs", "unknown target %q", id)
		}
	}
	if f.group != "" && !slices.ContainsFunc(cfg.Groups, func(g types.Group) bool { return g.ID == f.group }) {
		return nil, usageErrorf("unknown_group", "groups: "+groupIDs(cfg), "unknown group %q", f.group)
	}
	safety := splitList(f.safety)
	for _, s := range safety {
		switch types.SafetyLevel(s) {
		case types.SafetyLevelSafe, types.SafetyLevelModerate, types.SafetyLevelRisky:
		default:
			return nil, usageErrorf("unknown_safety", "use safe, moderate, or risky", "unknown safety level %q", s)
		}
	}

	var out []types.Category
	for _, cat := range cfg.Categories {
		if len(f.ids) > 0 && !slices.Contains(f.ids, cat.ID) {
			continue
		}
		if f.group != "" && cat.Group != f.group {
			continue
		}
		if len(safety) > 0 && !slices.Contains(safety, string(cat.Safety)) {
			continue
		}
		out = append(out, cat)
	}
	return out, nil
}

func groupIDs(cfg *types.Config) string {
	ids := make([]string, 0, len(cfg.Groups))
	for _, g := range cfg.Groups {
		ids = append(ids, g.ID)
	}
	return strings.Join(ids, ", ")
}

func (a *App) runTargets(args []string) error {
	fs := newFlagSet("targets")
	var filter targetFilter
	filter.register(fs)
	jsonOutput := fs.Bool("json", false, "")
	ids, err := parseArgs(fs, args)
	if err != nil {
		return a.flagError("targets", err)
	}
	filter.ids = ids

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading targets: %w", err)
	}
	cats, err := filter.apply(cfg)
	if err != nil {
		return err
	}

	detailed := len(ids) > 0
	out := targetsOutput{Targets: make([]targetInfo, 0, len(cats))}
	for _, cat := range cats {
		info := targetInfo{
			ID: cat.ID, Name: cat.Name, Group: cat.Group,
			Safety: cat.Safety, Method: cat.Method, Note: cat.Note,
			Disposal: cat.DisposalMode(),
		}
		if detailed {
			info.Paths = cat.Paths
			info.BlockedBy = cat.BlockedByProcesses
			info.Guide = cat.Guide
		}
		out.Targets = append(out.Targets, info)
	}

	if *jsonOutput {
		return writeJSON(a.Out, out)
	}
	if detailed {
		a.printTargetDetails(out.Targets)
		return nil
	}
	a.printTargetTable(out.Targets)
	return nil
}

func (a *App) printTargetTable(targets []targetInfo) {
	tw := tabwriter.NewWriter(a.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tGROUP\tSAFETY\tMETHOD\tACTION\tNAME")
	for _, t := range targets {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", t.ID, t.Group, t.Safety, t.Method, t.Disposal.ShortLabel(), t.Name)
	}
	tw.Flush()
	fmt.Fprintf(a.Out, "\n%d targets. Next: mac-cleanup scan [<id>...]\n", len(targets))
}

func (a *App) printTargetDetails(targets []targetInfo) {
	for i, t := range targets {
		if i > 0 {
			fmt.Fprintln(a.Out)
		}
		fmt.Fprintf(a.Out, "%s (%s)\n", t.ID, t.Name)
		fmt.Fprintf(a.Out, "  group: %s  safety: %s  method: %s\n", t.Group, t.Safety, t.Method)
		fmt.Fprintf(a.Out, "  action: %s\n", t.Disposal.Label())
		if t.Note != "" {
			fmt.Fprintf(a.Out, "  note: %s\n", t.Note)
		}
		if len(t.BlockedBy) > 0 {
			fmt.Fprintf(a.Out, "  blocked while running: %s\n", strings.Join(t.BlockedBy, ", "))
		}
		for _, p := range t.Paths {
			fmt.Fprintf(a.Out, "  path: %s\n", p)
		}
		if t.Guide != "" {
			fmt.Fprintf(a.Out, "  guide: %s\n", strings.TrimSpace(t.Guide))
		}
	}
}
