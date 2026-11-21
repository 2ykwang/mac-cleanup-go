package cli

import (
	"errors"
	"flag"
	"math"
	"strconv"
	"strings"
	"time"
)

const day = 24 * time.Hour

// A negative duration means the caller omitted --older-than. Zero is a valid age.
func registerOlderThan(fs *flag.FlagSet) *time.Duration {
	var age time.Duration = -1
	fs.Func("older-than", "", func(s string) error {
		digits, ok := strings.CutSuffix(s, "d")
		days, err := strconv.ParseUint(digits, 10, 64)
		if !ok || err != nil || days > uint64(math.MaxInt64/day) {
			return errors.New("expected a whole number of days such as 7d")
		}
		age = time.Duration(days) * day
		return nil
	})
	return &age
}

func ageCutoff(age time.Duration) time.Time {
	if age < 0 {
		return time.Time{}
	}
	return time.Now().Add(-age)
}
