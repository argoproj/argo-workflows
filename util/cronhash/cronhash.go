// Package cronhash expands `H` (hash) tokens in cron expressions into plain cron expressions. `H`
// resolves to a value which is stable for a given CronWorkflow, so that many CronWorkflows are
// spread out over time instead of all firing at once.
package cronhash

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

// fieldRange is an inclusive range of values for a cron field
type fieldRange struct {
	min, max int
}

func (r fieldRange) size() int {
	return r.max - r.min + 1
}

// hashRanges are the ranges a bare `H` resolves within. The day of month is capped at 28 so that a
// schedule runs in every month. An explicit `H(<min>-<max>)` may go beyond these, up to fieldLimits.
var hashRanges = []fieldRange{
	{0, 59}, // minute
	{0, 23}, // hour
	{1, 28}, // day of month
	{1, 12}, // month
	{0, 6},  // day of week
}

// fieldLimits are the ranges of cron itself, which an explicit `H(<min>-<max>)` has to stay within
var fieldLimits = []fieldRange{
	{0, 59}, // minute
	{0, 23}, // hour
	{1, 31}, // day of month
	{1, 12}, // month
	{0, 6},  // day of week
}

var fieldNames = []string{"minute", "hour", "day of month", "month", "day of week"}

// Expand resolves every `H` token using a hash of namespace and name, so the result is always the
// same for the same CronWorkflow. Schedules without a `H`, and expressions this package does not
// understand, are returned unchanged so that the cron parser reports the error.
func Expand(schedule, namespace, name string) (string, error) {
	prefix, expr := splitPrefix(schedule)
	fields := strings.Fields(expr)
	// descriptors (`@daily`) and malformed expressions cannot contain a `H`
	if len(fields) != len(hashRanges) {
		return schedule, nil
	}
	expanded := make([]string, len(fields))
	hashed := false
	for i, field := range fields {
		expandedField, changed, err := expandField(field, i, namespace, name)
		if err != nil {
			return "", err
		}
		expanded[i] = expandedField
		hashed = hashed || changed
	}
	// joining normalizes whitespace, so only rebuild if a `H` was actually resolved
	if !hashed {
		return schedule, nil
	}
	return prefix + strings.Join(expanded, " "), nil
}

// ExpandAll expands every schedule, see Expand
func ExpandAll(schedules []string, namespace, name string) ([]string, error) {
	expanded := make([]string, len(schedules))
	for i, schedule := range schedules {
		s, err := Expand(schedule, namespace, name)
		if err != nil {
			return nil, err
		}
		expanded[i] = s
	}
	return expanded, nil
}

// splitPrefix splits off a timezone prefix such as `CRON_TZ=Asia/Tokyo`, so it is not taken for a field
func splitPrefix(schedule string) (string, string) {
	trimmed := strings.TrimLeft(schedule, " \t")
	for _, prefix := range []string{"CRON_TZ=", "TZ="} {
		if strings.HasPrefix(trimmed, prefix) {
			if i := strings.IndexAny(trimmed, " \t"); i >= 0 {
				return schedule[:len(schedule)-len(trimmed)] + trimmed[:i+1], trimmed[i+1:]
			}
		}
	}
	return "", schedule
}

// expandField expands each comma separated item of a field, reporting whether it held a `H`
func expandField(field string, fieldIndex int, namespace, name string) (string, bool, error) {
	items := strings.Split(field, ",")
	hashed := false
	for i, item := range items {
		if !strings.HasPrefix(item, "H") {
			continue
		}
		expanded, err := expandItem(item, fieldIndex, hash(namespace, name, fieldIndex, i))
		if err != nil {
			return "", false, err
		}
		items[i] = expanded
		hashed = true
	}
	if !hashed {
		return field, false, nil
	}
	return strings.Join(items, ","), true, nil
}

// expandItem expands a single `H`, `H/<step>`, `H(<min>-<max>)` or `H(<min>-<max>)/<step>` item
func expandItem(item string, fieldIndex int, h uint32) (string, error) {
	r := hashRanges[fieldIndex]
	rest := strings.TrimPrefix(item, "H")

	if strings.HasPrefix(rest, "(") {
		end := strings.Index(rest, ")")
		if end < 0 {
			return "", fmt.Errorf("%q is malformed: missing `)`", item)
		}
		var err error
		if r, err = parseRange(rest[1:end], fieldIndex); err != nil {
			return "", fmt.Errorf("%q is malformed: %w", item, err)
		}
		rest = rest[end+1:]
	}

	if rest == "" {
		return strconv.Itoa(r.min + int(h%uint32(r.size()))), nil
	}

	step, err := strconv.Atoi(strings.TrimPrefix(rest, "/"))
	if !strings.HasPrefix(rest, "/") || err != nil {
		return "", fmt.Errorf("%q is malformed: expected `H`, `H/<step>`, `H(<min>-<max>)` or `H(<min>-<max>)/<step>`", item)
	}
	if step <= 0 {
		return "", fmt.Errorf("%q is malformed: step must be greater than 0", item)
	}
	// offset the first run within the step, so schedules with the same step do not overlap
	offset := int(h % uint32(min(step, r.size())))
	return fmt.Sprintf("%d-%d/%d", r.min+offset, r.max, step), nil
}

func parseRange(s string, fieldIndex int) (fieldRange, error) {
	minStr, maxStr, found := strings.Cut(s, "-")
	if !found {
		return fieldRange{}, fmt.Errorf("expected a range like `1-5`, got %q", s)
	}
	rmin, err := strconv.Atoi(minStr)
	if err != nil {
		return fieldRange{}, fmt.Errorf("expected a range like `1-5`, got %q", s)
	}
	rmax, err := strconv.Atoi(maxStr)
	if err != nil {
		return fieldRange{}, fmt.Errorf("expected a range like `1-5`, got %q", s)
	}
	if rmin > rmax {
		return fieldRange{}, fmt.Errorf("range %q is inverted", s)
	}
	if limit := fieldLimits[fieldIndex]; rmin < limit.min || rmax > limit.max {
		return fieldRange{}, fmt.Errorf("range %q is outside of %d-%d, the range of the %s field", s, limit.min, limit.max, fieldNames[fieldIndex])
	}
	return fieldRange{rmin, rmax}, nil
}

// hash depends only on the CronWorkflow and the position of the `H`, so it never changes
func hash(namespace, name string, fieldIndex, itemIndex int) uint32 {
	h := fnv.New32a()
	// the position comes first so neighbouring positions do not hash to neighbouring values
	_, _ = fmt.Fprintf(h, "%d.%d#%s/%s", fieldIndex, itemIndex, namespace, name)
	return h.Sum32()
}
