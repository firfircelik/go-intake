package discover

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// dateLayouts are the layouts tried in order when classifying a
// string as a date. The list matches the v0.1 spec; adding new
// layouts widens the inference but also widens the chance of a
// false positive.
var dateLayouts = []string{
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"2006/01/02",
}

// datetimeLayouts are the layouts tried in order when classifying
// a string as a datetime. The list matches the v0.1 spec.
var datetimeLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

// boolStrings is the set of canonical boolean strings, lowercased
// and trimmed. Both the JSON bool type and these strings classify
// as TypeBool. The numeric forms "1" and "0" are intentionally not
// included: a column of "1"/"0" strings is more likely a count or
// identifier than a boolean flag, and ParseInt succeeds for them
// before ParseBool would.
var boolStrings = map[string]struct{}{
	"true":  {},
	"false": {},
	"yes":   {},
	"no":    {},
	"y":     {},
	"n":     {},
	"t":     {},
	"f":     {},
}

// classifyValue returns the primary type of v. The empty string is
// returned for nil so callers can ignore empty values.
func classifyValue(v any) FieldType {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case bool:
		return TypeBool
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return TypeInt
	case float32, float64:
		return TypeFloat
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		if _, ok := boolStrings[strings.ToLower(s)]; ok {
			return TypeBool
		}
		if isIntString(s) {
			return TypeInt
		}
		if isFloatString(s) {
			return TypeFloat
		}
		if isDateTimeString(s) {
			return TypeDatetime
		}
		if isDateString(s) {
			return TypeDate
		}
		return TypeString
	}
	return TypeString
}

func isIntString(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func isFloatString(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isDateString(s string) bool {
	for _, layout := range dateLayouts {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

func isDateTimeString(s string) bool {
	for _, layout := range datetimeLayouts {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

// toFloat coerces a numeric value to float64. It only accepts
// concrete numeric types; for strings it returns false. Callers
// that have classified a value as int or float should use
// numericValue, which also handles numeric strings.
func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	}
	return 0, false
}

// numericValue returns the numeric value of v, including strings
// that have already been classified as int or float. classifyValue
// must be consulted first; numericValue is only meaningful for
// values whose classification is int or float.
func numericValue(v any) (float64, bool) {
	if f, ok := toFloat(v); ok {
		return f, true
	}
	if s, ok := v.(string); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// toString returns a stable string representation of v. It is used
// only for building Examples and the duplicate-row hash.
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	}
	return ""
}

// fieldAcc is the per-field accumulator used during a single
// pass of InspectSource. All counters are zero-valued on
// construction.
type fieldAcc struct {
	name        string
	maxExamples int
	maxUnique   int

	nullCount    int
	nonNullCount int

	intCount      int
	floatCount    int
	boolCount     int
	dateCount     int
	datetimeCount int
	stringCount   int

	minFloat float64
	maxFloat float64
	hasRange bool

	minLen int
	maxLen int
	hasLen bool

	examples       []string
	examplesSet    map[string]struct{}
	distinct       map[string]struct{}
	distinctCapped bool
}

func newFieldAcc(name string, maxExamples, maxUnique int) *fieldAcc {
	return &fieldAcc{
		name:        name,
		maxExamples: maxExamples,
		maxUnique:   maxUnique,
		minFloat:    math.Inf(1),
		maxFloat:    math.Inf(-1),
		examplesSet: make(map[string]struct{}, maxExamples),
		distinct:    make(map[string]struct{}, 16),
	}
}

func (a *fieldAcc) observe(v any) {
	t := classifyValue(v)
	if t == "" {
		// nil or empty/whitespace string.
		if v == nil {
			a.nullCount++
			return
		}
		// Empty string is not "missing" by the strict definition,
		// but it carries no information. We treat it as null for
		// counting purposes so that a column of "" is flagged the
		// same as a column of nil.
		a.nullCount++
		return
	}
	a.nonNullCount++

	switch t {
	case TypeInt:
		a.intCount++
		if f, ok := numericValue(v); ok {
			a.updateRange(f)
		}
	case TypeFloat:
		a.floatCount++
		if f, ok := numericValue(v); ok {
			a.updateRange(f)
		}
	case TypeBool:
		a.boolCount++
	case TypeDate:
		a.dateCount++
	case TypeDatetime:
		a.datetimeCount++
	case TypeString:
		a.stringCount++
		if s, ok := v.(string); ok {
			l := len([]rune(strings.TrimSpace(s)))
			if !a.hasLen {
				a.minLen, a.maxLen = l, l
				a.hasLen = true
			} else {
				if l < a.minLen {
					a.minLen = l
				}
				if l > a.maxLen {
					a.maxLen = l
				}
			}
		}
	}

	sv := toString(v)
	if _, seen := a.examplesSet[sv]; !seen {
		if len(a.examples) < a.maxExamples {
			a.examples = append(a.examples, sv)
			a.examplesSet[sv] = struct{}{}
		}
	}
	if !a.distinctCapped {
		if _, ok := a.distinct[sv]; !ok {
			if len(a.distinct) < a.maxUnique {
				a.distinct[sv] = struct{}{}
			} else {
				a.distinctCapped = true
			}
		}
	}
}

func (a *fieldAcc) updateRange(f float64) {
	if !a.hasRange {
		a.minFloat, a.maxFloat = f, f
		a.hasRange = true
		return
	}
	if f < a.minFloat {
		a.minFloat = f
	}
	if f > a.maxFloat {
		a.maxFloat = f
	}
}

func (a *fieldAcc) finalize() FieldProfile {
	p := FieldProfile{
		Name:         a.name,
		NullCount:    a.nullCount,
		NonNullCount: a.nonNullCount,
		UniqueCount:  len(a.distinct),
		Examples:     a.examples,
	}
	if total := a.nullCount + a.nonNullCount; total > 0 {
		p.NullRatio = float64(a.nullCount) / float64(total)
	}
	if a.nonNullCount > 0 {
		p.UniqueRatio = float64(p.UniqueCount) / float64(a.nonNullCount)
	}
	if a.hasRange {
		p.MinFloat = a.minFloat
		p.MaxFloat = a.maxFloat
	}
	if a.hasLen {
		p.MinLength = a.minLen
		p.MaxLength = a.maxLen
	}
	p.Type, p.TypeConfidence, p.ParseErrors = inferType(a)
	return p
}

// inferType picks the FieldType for a finalized accumulator. The
// second return value is TypeConfidence (fraction of non-null
// values matching Type). The third is the count of non-null values
// whose primary type did not match Type.
func inferType(a *fieldAcc) (FieldType, float64, int) {
	n := a.nonNullCount
	if n == 0 {
		return TypeUnknown, 0, 0
	}
	if a.boolCount == n {
		return TypeBool, 1.0, 0
	}
	if a.intCount == n {
		return TypeInt, 1.0, 0
	}
	if a.intCount+a.floatCount == n {
		return TypeFloat, 1.0, 0
	}
	if a.dateCount == n {
		return TypeDate, 1.0, 0
	}
	if a.datetimeCount == n {
		return TypeDatetime, 1.0, 0
	}
	if a.stringCount == n {
		return TypeString, 1.0, 0
	}
	// Mixed: fall back to string and report how many values were
	// not strings.
	nonString := n - a.stringCount
	confidence := float64(a.stringCount) / float64(n)
	return TypeString, confidence, nonString
}
