package quartz

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CronTrigger implements the quartz.Trigger interface.
// Used to fire a Job at given moments in time, defined with Unix 'cron-like' schedule definitions.
//
// Examples:
//
// Expression               Meaning
// "0 0 12 * * ?"           Fire at 12pm (noon) every day
// "0 15 10 ? * *"          Fire at 10:15am every day
// "0 15 10 * * ?"          Fire at 10:15am every day
// "0 15 10 * * ? *"        Fire at 10:15am every day
// "0 * 14 * * ?"           Fire every minute starting at 2pm and ending at 2:59pm, every day
// "0 0/5 14 * * ?"         Fire every 5 minutes starting at 2pm and ending at 2:55pm, every day
// "0 0/5 14,18 * * ?"      Fire every 5 minutes starting at 2pm and ending at 2:55pm,
// AND fire every 5 minutes starting at 6pm and ending at 6:55pm, every day
// "0 0-5 14 * * ?"         Fire every minute starting at 2pm and ending at 2:05pm, every day
// "0 10,44 14 ? 3 WED"     Fire at 2:10pm and at 2:44pm every Wednesday in the month of March.
// "0 15 10 ? * MON-FRI"    Fire at 10:15am every Monday, Tuesday, Wednesday, Thursday and Friday
// "0 15 10 15 * ?"         Fire at 10:15am on the 15th day of every month
type CronTrigger struct {
	expression  string
	fields      []*cronField
	lastDefined int
	location    *time.Location
}

// Verify CronTrigger satisfies the Trigger interface.
var _ Trigger = (*CronTrigger)(nil)

// NewCronTrigger returns a new CronTrigger using the UTC location.
func NewCronTrigger(expr string) (*CronTrigger, error) {
	return NewCronTriggerWithLoc(expr, time.UTC)
}

// NewCronTriggerWithLoc returns a new CronTrigger with the given time.Location.
func NewCronTriggerWithLoc(expr string, location *time.Location) (*CronTrigger, error) {
	fields, err := validateCronExpression(expr)
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
		fields[0].values, _ = fillRange(0, 59)
	}

	return &CronTrigger{
		expression:  expr,
		fields:      fields,
		lastDefined: lastDefined,
		location:    location,
	}, nil
}

// NextFireTime returns the next time at which the CronTrigger is scheduled to fire.
func (ct *CronTrigger) NextFireTime(prev int64) (int64, error) {
	parser := newCronExpressionParser(ct.lastDefined)
	prevTime := time.Unix(prev/int64(time.Second), 0).In(ct.location)
	return parser.nextTime(prevTime, ct.fields)
}

// Description returns the description of the trigger.
func (ct *CronTrigger) Description() string {
	return fmt.Sprintf("CronTrigger %s", ct.expression)
}

// cronExpressionParser parses cron expressions.
type cronExpressionParser struct {
	minuteBump bool
	hourBump   bool
	dayBump    bool
	monthBump  bool
	yearBump   bool
	done       bool

	lastDefined int
	maxDays     int
}

// newCronExpressionParser returns a new cronExpressionParser.
func newCronExpressionParser(lastDefined int) *cronExpressionParser {
	return &cronExpressionParser{false, false, false, false, false, false,
		lastDefined, 0}
}

// cronField represents a parsed cron expression as an array.
type cronField struct {
	values []int
}

// isEmpty checks if the cronField values array is empty.
func (cf *cronField) isEmpty() bool {
	return len(cf.values) == 0
}

// incr increments each element of the underlying array by the given value.
func (cf *cronField) incr(a int) {
	if !cf.isEmpty() {
		mapped := make([]int, len(cf.values))
		for i, v := range cf.values {
			mapped[i] = v + a
		}
		cf.values = mapped
	}
}

// String is the cronField fmt.Stringer implementation.
func (cf *cronField) String() string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cf.values)), ","), "[]")
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

// <second> <minute> <hour> <day-of-month> <month> <day-of-week> <year>
// <year> field is optional

// the ? wildcard is only used in the day of month and day of week fields
func validateCronExpression(expression string) ([]*cronField, error) {
	var tokens []string

	if value, ok := special[expression]; ok {
		tokens = strings.Split(value, " ")
	} else {
		tokens = strings.Split(expression, " ")
	}
	length := len(tokens)
	if length < 6 || length > 7 {
		return nil, cronError("Invalid expression length")
	}
	if length == 6 {
		tokens = append(tokens, "*")
	}
	if (tokens[3] != "?" && tokens[3] != "*") && (tokens[5] != "?" && tokens[5] != "*") {
		return nil, cronError("Day field was set twice")
	}

	return buildCronField(tokens)
}

func buildCronField(tokens []string) ([]*cronField, error) {
	var err error
	fields := make([]*cronField, 7)
	fields[0], err = parseField(tokens[0], 0, 59)
	if err != nil {
		return nil, err
	}

	fields[1], err = parseField(tokens[1], 0, 59)
	if err != nil {
		return nil, err
	}

	fields[2], err = parseField(tokens[2], 0, 23)
	if err != nil {
		return nil, err
	}

	fields[3], err = parseField(tokens[3], 1, 31)
	if err != nil {
		return nil, err
	}

	fields[4], err = parseField(tokens[4], 1, 12, months)
	if err != nil {
		return nil, err
	}

	fields[5], err = parseField(tokens[5], 1, 7, days)
	if err != nil {
		return nil, err
	}
	fields[5].incr(-1)

	fields[6], err = parseField(tokens[6], 1970, 1970*2)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func parseField(field string, min int, max int, translate ...[]string) (*cronField, error) {
	var dict []string
	if len(translate) > 0 {
		dict = translate[0]
	}

	// any value
	if field == "*" || field == "?" {
		return &cronField{[]int{}}, nil
	}

	// single value
	i, err := strconv.Atoi(field)
	if err == nil {
		if inScope(i, min, max) {
			return &cronField{[]int{i}}, nil
		}
		return nil, cronError("Single min/max validation error")
	}

	// list values
	if strings.Contains(field, ",") {
		return parseListField(field, dict)
	}

	// range values
	if strings.Contains(field, "-") {
		return parseRangeField(field, min, max, dict)
	}

	// step values
	if strings.Contains(field, "/") {
		return parseStepField(field, min, max, dict)
	}

	// literal single value
	if dict != nil {
		i := intVal(dict, field)
		if i >= 0 {
			if inScope(i, min, max) {
				return &cronField{[]int{i}}, nil
			}
			return nil, cronError("Cron literal min/max validation error")
		}
	}

	return nil, cronError("Cron parse error")
}

func parseListField(field string, translate []string) (*cronField, error) {
	t := strings.Split(field, ",")
	si, err := sliceAtoi(t)
	if err != nil {
		si, err = indexes(t, translate)
		if err != nil {
			return nil, err
		}
	}

	sort.Ints(si)
	return &cronField{si}, nil
}

func parseRangeField(field string, min int, max int, translate []string) (*cronField, error) {
	var _range []int
	t := strings.Split(field, "-")
	if len(t) != 2 {
		return nil, cronError("Parse cron range error")
	}

	from := normalize(t[0], translate)
	to := normalize(t[1], translate)
	if !inScope(from, min, max) || !inScope(to, min, max) {
		return nil, cronError("Cron range min/max validation error")
	}

	_range, err := fillRange(from, to)
	if err != nil {
		return nil, err
	}

	return &cronField{_range}, nil
}

func parseStepField(field string, min int, max int, translate []string) (*cronField, error) {
	var _step []int
	t := strings.Split(field, "/")
	if len(t) != 2 {
		return nil, cronError("Parse cron step error")
	}

	if t[0] == "*" {
		t[0] = strconv.Itoa(min)
	}

	from := normalize(t[0], translate)
	step := atoi(t[1])
	if !inScope(from, min, max) {
		return nil, cronError("Cron step min/max validation error")
	}

	_step, err := fillStep(from, step, max)
	if err != nil {
		return nil, err
	}

	return &cronField{_step}, nil
}

func (parser *cronExpressionParser) nextTime(prev time.Time, fields []*cronField) (nextTime int64, err error) {
	// Build CronStateMachine and run once
	csm := makeCSMFromFields(prev, fields)
	nextDateTime := csm.NextTriggerTime(prev.Location())
	return nextDateTime.UnixNano(), nil
}
