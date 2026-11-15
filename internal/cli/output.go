package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
)

// cliError is a failure with a stable machine-readable code and a next step.
type cliError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Exit    int    `json:"-"`
}

func (e *cliError) Error() string { return e.Message }

func usageErrorf(code, hint, format string, args ...any) *cliError {
	return &cliError{Code: code, Message: fmt.Sprintf(format, args...), Hint: hint, Exit: ExitUsage}
}

func runtimeErrorf(code, hint, format string, args ...any) *cliError {
	return &cliError{Code: code, Message: fmt.Sprintf(format, args...), Hint: hint, Exit: ExitError}
}

// partialError marks a run whose output was already written but where some
// items failed. It maps to ExitPartial without printing anything more.
type partialError struct{ failed int }

func (e *partialError) Error() string { return fmt.Sprintf("%d items failed", e.failed) }

func writeJSONError(w io.Writer, e *cliError) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]*cliError{"error": e}); err != nil {
		fmt.Fprintf(w, "error: %s\n", e.Message)
	}
}

// writeJSON indents only for terminals; piped output stays compact for tokens.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // keep <id> and paths readable
	if isTerminal(w) {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--json" || arg == "-json" {
			return true
		}
	}
	return false
}

var isTerminal = func(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

// colorEnabled follows the NO_COLOR and TERM=dumb conventions.
func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func noArgs(name string, args []string) error {
	positional, err := parseArgs(newFlagSet(name), args)
	if err != nil {
		return err
	}
	if len(positional) > 0 {
		return fmt.Errorf("unexpected argument %q", positional[0])
	}
	return nil
}

// parseArgs lets flags and positional arguments mix, so
// `clean docker --dry-run` and `clean --dry-run docker` mean the same thing.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		// Parse drops a "--" terminator; everything after it is positional.
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(positional, rest...), nil
		}
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// flagError turns flag parsing failures into usage errors, and -h into help.
func (a *App) flagError(name string, err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return a.runHelp([]string{name})
	}
	return usageErrorf("invalid_flag", fmt.Sprintf("run `mac-cleanup help %s`", name), "%s", err.Error())
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// splitList accepts both repeated flags and comma-separated values.
func splitList(values []string) []string {
	var out []string
	for _, v := range values {
		for part := range strings.SplitSeq(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
