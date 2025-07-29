package quartz

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CronTrigger implements the [Trigger] interface.
// Used to fire a Job at given moments in time, defined with Unix 'cron-like' schedule definitions.
//
// Examples:
//
//	Expression               Meaning
//	"0 0 12 * * ?"           Fire at 12pm (noon) every day
//	"0 15 10 ? * *"          Fire at 10:15am every day
//	"0 15 10 * * ?"          Fire at 10:15am every day
//	"0 15 10 * * ? *"        Fire at 10:15am every day
//	"0 * 14 * * ?"           Fire every minute starting at 2pm and ending at 2:59pm, every day
//	"0 0/5 14 * * ?"         Fire every 5 minutes starting at 2pm and ending at 2:55pm, every day
//	"0 0/5 14,18 * * ?"      Fire every 5 minutes starting at 2pm and ending at 2:55pm,
//	                         AND fire every 5 minutes starting at 6pm and ending at 6:55pm, every day
//	"0 0-5 14 * * ?"         Fire every minute starting at 2pm and ending at 2:05pm, every day
//	"0 10,44 14 ? 3 WED"     Fire at 2:10pm and at 2:44pm every Wednesday in the month of March.
//	"0 15 10 ? * MON-FRI"    Fire at 10:15am every Monday, Tuesday, Wednesday, Thursday and Friday
//	"0 15 10 15 * ?"         Fire at 10:15am on the 15th day of every month
//	"0 15 10 ? * 6L"         Fire at 10:15am on the last Friday of every month
//	"0 15 10 ? * 6#3"        Fire at 10:15am on the third Friday of every month
//	"0 15 10 L * ?"          Fire at 10:15am on the last day of every month
//	"0 15 10 L-2 * ?"        Fire at 10:15am on the 2nd-to-last last day of every month
type CronTrigger struct {
	expression  string
	fields      []*cronField
	location    *time.Location
	lastDefined int
	tzPasses    int
}

// Verify CronTrigger satisfies the Trigger interface.
var _ Trigger = (*CronTrigger)(nil)

// NewCronTrigger returns a new [CronTrigger] using the UTC location.
func NewCronTrigger(expression string) (*CronTrigger, error) {
	return NewCronTriggerWithLoc(expression, time.UTC)
}

// NewCronTriggerWithLoc returns a new [CronTrigger] with the given [time.Location].
func NewCronTriggerWithLoc(expression string, location *time.Location) (*CronTrigger, error) {
	if location == nil {
		return nil, newIllegalArgumentError("location is nil")
	}
	expression = trimCronExpression(expression)
	fields, err := parseCronExpression(expression)
	if err != nil {
		return nil, err
	}
	lastDefined := -1
	for i, field := range fields {
		if len(field.values) > 0 {
			lastDefined = i
		}
	}
	// full wildcard expression
	if lastDefined == -1 {
		fields[0].values, _ = fillRangeValues(0, 59)
	}

	// tzPasses determines the number of iterations required when calculating the next
	// fire time. Two iterations are used for time zones with Daylight Saving Time
	// (DST) to resolve ambiguities caused by clock adjustments. UTC requires only
	// one iteration as it's unaffected by DST.
	tzPasses := 2
	if location == time.UTC {
		tzPasses = 1
	}

	return &CronTrigger{
		expression:  expression,
		fields:      fields,
		location:    location,
		lastDefined: lastDefined,
		tzPasses:    tzPasses,
	}, nil
}

// NextFireTime returns the next time at which the CronTrigger is scheduled to fire.
func (ct *CronTrigger) NextFireTime(prev int64) (int64, error) {
	prevTime := time.Unix(prev/int64(time.Second), 0).In(ct.location)
	// Initialize a CronStateMachine from the previous fire time and cron fields.
	csm := newCSMFromFields(prevTime, ct.fields)

	// Iterate ct.tzPasses times to determine the correct next scheduled fire time.
	// This accounts for complexities like Daylight Saving Time (DST) transitions.
	for i := 0; i < ct.tzPasses; i++ {
		nextDateTime := csm.NextTriggerTime(ct.location)
		if nextDateTime.After(prevTime) {
			return nextDateTime.UnixNano(), nil
		}
	}
	return 0, ErrTriggerExpired
}

// Description returns the description of the cron trigger.
func (ct *CronTrigger) Description() string {
	return fmt.Sprintf("CronTrigger%s%s%s%s", Sep, ct.expression, Sep, ct.location)
}

// cronField represents a parsed cron expression field.
type cronField struct {
	// stores the parsed and sorted numeric values for the field
	values []int
	// n is used to store special values for the day-of-month
	// and day-of-week fields
	n int
}

// newCronField returns a new cronField.
func newCronField(values []int) *cronField {
	return &cronField{values: values}
}

// newCronFieldN returns a new cronField with the provided n.
func newCronFieldN(values []int, n int) *cronField {
	return &cronField{values: values, n: n}
}

// add increments each element of the underlying values slice by the given delta.
func (cf *cronField) add(delta int) {
	for i := range cf.values {
		cf.values[i] += delta
	}
}

// String is the cronField fmt.Stringer implementation.
func (cf *cronField) String() string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cf.values)), ","), "[]")
}

// boundary represents inclusive range boundaries for cron field values.
type boundary struct {
	lower int
	upper int
}

var (
	months = []string{"0", "JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}
	days   = []string{"0", "SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}

	// the pre-defined cron expressions
	special = map[string]string{
		"@yearly":  "0 0 0 1 1 *",
		"@monthly": "0 0 0 1 * *",
		"@weekly":  "0 0 0 * * 1",
		"@daily":   "0 0 0 * * *",
		"@hourly":  "0 0 * * * *",
	}
)

// ValidateCronExpression validates a cron expression string.
// A valid expression consists of the following fields:
//
//	<second> <minute> <hour> <day-of-month> <month> <day-of-week> <year>
//
// where the <year> field is optional.
// See the cron expression format table in the readme file for supported special characters.
func ValidateCronExpression(expression string) error {
	_, err := parseCronExpression(trimCronExpression(expression))
	return err
}

// parseCronExpression parses a cron expression string.
func parseCronExpression(expression string) ([]*cronField, error) {
	var tokens []string
	if value, ok := special[expression]; ok {
		tokens = strings.Split(value, " ")
	} else {
		tokens = strings.Split(expression, " ")
	}
	length := len(tokens)
	if length < 6 || length > 7 {
		return nil, newCronParseError("invalid expression length")
	}
	if length == 6 {
		tokens = append(tokens, "*")
	}
	if (tokens[3] != "?" && tokens[3] != "*") && (tokens[5] != "?" && tokens[5] != "*") {
		return nil, newCronParseError("day field set twice")
	}

	return buildCronField(tokens)
}

var whitespacePattern = regexp.MustCompile(`\s+`)

func trimCronExpression(expression string) string {
	return strings.TrimSpace(whitespacePattern.ReplaceAllString(expression, " "))
}

func buildCronField(tokens []string) ([]*cronField, error) {
	var err error
	fields := make([]*cronField, 7)
	// second field
	fields[0], err = parseField(tokens[0], boundary{0, 59}, nil)
	if err != nil {
		return nil, err
	}
	// minute field
	fields[1], err = parseField(tokens[1], boundary{0, 59}, nil)
	if err != nil {
		return nil, err
	}
	// hour field
	fields[2], err = parseField(tokens[2], boundary{0, 23}, nil)
	if err != nil {
		return nil, err
	}
	// day-of-month field
	fields[3], err = parseDayOfMonthField(tokens[3], boundary{1, 31}, nil)
	if err != nil {
		return nil, err
	}
	// month field
	fields[4], err = parseField(tokens[4], boundary{1, 12}, months)
	if err != nil {
		return nil, err
	}
	// day-of-week field
	fields[5], err = parseDayOfWeekField(tokens[5], boundary{1, 7}, days)
	if err != nil {
		return nil, err
	}
	fields[5].add(-1)
	// year field
	fields[6], err = parseField(tokens[6], boundary{1970, 1970 * 2}, nil)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func parseField(field string, bound boundary, names []string) (*cronField, error) {
	// any value
	if field == "*" || field == "?" {
		return newCronField([]int{}), nil
	}
	// list values
	if strings.ContainsRune(field, listRune) {
		return parseListField(field, bound, names)
	}
	// step values
	if strings.ContainsRune(field, stepRune) {
		return parseStepField(field, bound, names)
	}
	// range values
	if strings.ContainsRune(field, rangeRune) {
		return parseRangeField(field, bound, names)
	}
	// simple value
	numeric, err := normalize(field, names)
	if err != nil {
		return nil, err
	}
	if inScope(numeric, bound.lower, bound.upper) {
		return newCronField([]int{numeric}), nil
	}

	return nil, newInvalidCronFieldError("numeric", field)
}

var (
	cronLastMonthDayRegex = regexp.MustCompile(`^L(-[0-9]+)?$`)
	cronWeekdayRegex      = regexp.MustCompile(`^[0-9]+W$`)

	cronLastWeekdayRegex = regexp.MustCompile(`^[a-zA-Z0-9]*L$`)
	cronHashRegex        = regexp.MustCompile(`^[a-zA-Z0-9]+#[0-9]+$`)
)

func parseDayOfMonthField(field string, bound boundary, names []string) (*cronField, error) {
	if strings.ContainsRune(field, lastRune) && cronLastMonthDayRegex.MatchString(field) {
		if field == string(lastRune) {
			return newCronFieldN([]int{}, cronLastDayOfMonthN), nil
		}
		values := strings.Split(field, string(rangeRune))
		if len(values) != 2 {
			return nil, newInvalidCronFieldError("last", field)
		}
		n, err := strconv.Atoi(values[1])
		if err != nil || !inScope(n, bound.lower, bound.upper) {
			return nil, newInvalidCronFieldError("last", field)
		}
		return newCronFieldN([]int{}, -n), nil
	}

	if strings.ContainsRune(field, weekdayRune) {
		if field == fmt.Sprintf("%c%c", lastRune, weekdayRune) {
			return newCronFieldN([]int{0}, cronLastDayOfMonthN|cronWeekdayN), nil
		}

		if cronWeekdayRegex.MatchString(field) {
			day := strings.TrimSuffix(field, string(weekdayRune))
			if day == "" {
				return nil, newInvalidCronFieldError("weekday", field)
			}
			dayOfMonth, err := strconv.Atoi(day)
			if err != nil || !inScope(dayOfMonth, bound.lower, bound.upper) {
				return nil, newInvalidCronFieldError("weekday", field)
			}
			return newCronFieldN([]int{dayOfMonth}, cronWeekdayN), nil
		}
	}

	return parseField(field, bound, names)
}

func parseDayOfWeekField(field string, bound boundary, names []string) (*cronField, error) {
	if strings.ContainsRune(field, lastRune) && cronLastWeekdayRegex.MatchString(field) {
		day := strings.TrimSuffix(field, string(lastRune))
		if day == "" { // Saturday
			return newCronFieldN([]int{7}, -1), nil
		}
		dayOfWeek, err := normalize(day, names)
		if err != nil || !inScope(dayOfWeek, bound.lower, bound.upper) {
			return nil, newInvalidCronFieldError("last", field)
		}
		return newCronFieldN([]int{dayOfWeek}, -1), nil
	}

	if strings.ContainsRune(field, hashRune) && cronHashRegex.MatchString(field) {
		values := strings.Split(field, string(hashRune))
		if len(values) != 2 {
			return nil, newInvalidCronFieldError("hash", field)
		}
		dayOfWeek, err := normalize(values[0], names)
		if err != nil || !inScope(dayOfWeek, bound.lower, bound.upper) {
			return nil, newInvalidCronFieldError("hash", field)
		}
		n, err := strconv.Atoi(values[1])
		if err != nil || !inScope(n, 1, 5) {
			return nil, newInvalidCronFieldError("hash", field)
		}
		return newCronFieldN([]int{dayOfWeek}, n), nil
	}

	return parseField(field, bound, names)
}

func parseListField(field string, bound boundary, names []string) (*cronField, error) {
	t := strings.Split(field, string(listRune))
	values, stepValues := extractStepValues(t)
	values, rangeValues := extractRangeValues(values)
	listValues, err := translateLiterals(names, values)
	if err != nil {
		return nil, err
	}
	for _, v := range stepValues {
		stepField, err := parseStepField(v, bound, names)
		if err != nil {
			return nil, err
		}
		listValues = append(listValues, stepField.values...)
	}
	for _, v := range rangeValues {
		rangeField, err := parseRangeField(v, bound, names)
		if err != nil {
			return nil, err
		}
		listValues = append(listValues, rangeField.values...)
	}

	sort.Ints(listValues)
	return newCronField(listValues), nil
}

func parseRangeField(field string, bound boundary, names []string) (*cronField, error) {
	t := strings.Split(field, string(rangeRune))
	if len(t) != 2 {
		return nil, newInvalidCronFieldError("range", field)
	}
	from, err := normalize(t[0], names)
	if err != nil {
		return nil, err
	}
	to, err := normalize(t[1], names)
	if err != nil {
		return nil, err
	}
	if !inScope(from, bound.lower, bound.upper) || !inScope(to, bound.lower, bound.upper) {
		return nil, newInvalidCronFieldError("range", field)
	}
	rangeValues, err := fillRangeValues(from, to)
	if err != nil {
		return nil, err
	}

	return newCronField(rangeValues), nil
}

func parseStepField(field string, bound boundary, names []string) (*cronField, error) {
	t := strings.Split(field, string(stepRune))
	if len(t) != 2 {
		return nil, newInvalidCronFieldError("step", field)
	}
	to := bound.upper
	var (
		from int
		err  error
	)
	switch {
	case t[0] == "*":
		from = bound.lower
	case strings.ContainsRune(t[0], rangeRune):
		trange := strings.Split(t[0], string(rangeRune))
		if len(trange) != 2 {
			return nil, newInvalidCronFieldError("step", field)
		}
		from, err = normalize(trange[0], names)
		if err != nil {
			return nil, err
		}
		to, err = normalize(trange[1], names)
		if err != nil {
			return nil, err
		}
	default:
		from, err = normalize(t[0], names)
		if err != nil {
			return nil, err
		}
	}

	step, err := strconv.Atoi(t[1])
	if err != nil {
		return nil, newInvalidCronFieldError("step", field)
	}
	if !inScope(from, bound.lower, bound.upper) || !inScope(step, 1, bound.upper) ||
		!inScope(to, bound.lower, bound.upper) {
		return nil, newInvalidCronFieldError("step", field)
	}

	stepValues, err := fillStepValues(from, step, to)
	if err != nil {
		return nil, err
	}

	return newCronField(stepValues), nil
}
