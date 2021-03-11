package quartz

import (
	"errors"
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
//                          AND fire every 5 minutes starting at 6pm and ending at 6:55pm, every day
// "0 0-5 14 * * ?"         Fire every minute starting at 2pm and ending at 2:05pm, every day
// "0 10,44 14 ? 3 WED"     Fire at 2:10pm and at 2:44pm every Wednesday in the month of March.
// "0 15 10 ? * MON-FRI"    Fire at 10:15am every Monday, Tuesday, Wednesday, Thursday and Friday
// "0 15 10 15 * ?"         Fire at 10:15am on the 15th day of every month
type CronTrigger struct {
	expression  string
	fields      []*CronField
	lastDefined int
}

// NewCronTrigger returns a new CronTrigger.
func NewCronTrigger(expr string) (*CronTrigger, error) {
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

	return &CronTrigger{expr, fields, lastDefined}, nil
}

// NextFireTime returns the next time at which the CronTrigger is scheduled to fire.
func (ct *CronTrigger) NextFireTime(prev int64) (int64, error) {
	parser := NewCronExpressionParser(ct.lastDefined)
	return parser.nextTime(prev, ct.fields)
}

// Description returns a CronTrigger description.
func (ct *CronTrigger) Description() string {
	return fmt.Sprintf("CronTrigger %s", ct.expression)
}

// CronExpressionParser parses cron expressions.
type CronExpressionParser struct {
	minuteBump bool
	hourBump   bool
	dayBump    bool
	monthBump  bool
	yearBump   bool
	done       bool

	lastDefined int
	maxDays     int
}

// NewCronExpressionParser returns a new CronExpressionParser.
func NewCronExpressionParser(lastDefined int) *CronExpressionParser {
	return &CronExpressionParser{false, false, false, false, false, false,
		lastDefined, 0}
}

// CronField represents a parsed cron expression as an array.
type CronField struct {
	values []int
}

// isEmpty checks if the CronField values array is empty.
func (cf *CronField) isEmpty() bool {
	return len(cf.values) == 0
}

// String is the CronField fmt.Stringer implementation.
func (cf *CronField) String() string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cf.values)), ","), "[]")
}

var (
	months      = []string{"0", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	days        = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	daysInMonth = []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// the pre-defined cron expressions
	special = map[string]string{
		"@yearly":  "0 0 0 1 1 *",
		"@monthly": "0 0 0 1 * *",
		"@weekly":  "0 0 0 * * 0",
		"@daily":   "0 0 0 * * *",
		"@hourly":  "0 0 * * * *",
	}

	readDateLayout  = "Mon Jan 2 15:04:05 2006"
	writeDateLayout = "Jan 2 15:04:05 2006"
)

// <second> <minute> <hour> <day-of-month> <month> <day-of-week> <year>
// <year> field is optional
const (
	secondIndex = iota
	minuteIndex
	hourIndex
	dayOfMonthIndex
	monthIndex
	dayOfWeekIndex
	yearIndex
)

func (parser *CronExpressionParser) nextTime(prev int64, fields []*CronField) (nextTime int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown cron expression error")
			}
		}
	}()

	tfmt := time.Unix(prev/int64(time.Second), 0).UTC().Format(readDateLayout)
	ttok := strings.Split(strings.Replace(tfmt, "  ", " ", 1), " ")
	hms := strings.Split(ttok[3], ":")
	parser.maxDays = maxDays(intVal(months, ttok[1]), atoi(ttok[4]))

	second := parser.nextSeconds(atoi(hms[2]), fields[0])
	minute := parser.nextMinutes(atoi(hms[1]), fields[1])
	hour := parser.nextHours(atoi(hms[0]), fields[2])
	dayOfMonth := parser.nextDay(intVal(days, ttok[0]), fields[5], atoi(ttok[2]), fields[3])
	month := parser.nextMonth(ttok[1], fields[4])
	year := parser.nextYear(ttok[4], fields[6])

	nstr := fmt.Sprintf("%s %s %s:%s:%s %s", month, strconv.Itoa(dayOfMonth),
		hour, minute, second, year)
	ntime, err := time.Parse(writeDateLayout, nstr)
	nextTime = ntime.UnixNano()
	return
}

// the ? wildcard is only used in the day of month and day of week fields
func validateCronExpression(expression string) ([]*CronField, error) {
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
	if tokens[6] != "*" {
		return nil, cronError("Year field is not supported, use asterisk") // TODO: support year field
	}

	return buildCronField(tokens)
}

func buildCronField(tokens []string) ([]*CronField, error) {
	var err error
	fields := make([]*CronField, 7)
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

	fields[5], err = parseField(tokens[5], 0, 6, days)
	if err != nil {
		return nil, err
	}

	fields[6], err = parseField(tokens[6], 1970, 1970*2)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func parseField(field string, min int, max int, translate ...[]string) (*CronField, error) {
	var dict []string
	if len(translate) > 0 {
		dict = translate[0]
	}

	// any value
	if field == "*" || field == "?" {
		return &CronField{[]int{}}, nil
	}

	// single value
	i, err := strconv.Atoi(field)
	if err == nil {
		if inScope(i, min, max) {
			return &CronField{[]int{i}}, nil
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
				return &CronField{[]int{i}}, nil
			}
			return nil, cronError("Cron literal min/max validation error")
		}
	}

	return nil, cronError("Cron parse error")
}

func parseListField(field string, translate []string) (*CronField, error) {
	t := strings.Split(field, ",")
	si, err := sliceAtoi(t)
	if err != nil {
		si, err = indexes(t, translate)
		if err != nil {
			return nil, err
		}
	}

	sort.Ints(si)
	return &CronField{si}, nil
}

func parseRangeField(field string, min int, max int, translate []string) (*CronField, error) {
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

	return &CronField{_range}, nil
}

func parseStepField(field string, min int, max int, translate []string) (*CronField, error) {
	var _step []int
	t := strings.Split(field, "/")
	if len(t) != 2 {
		return nil, cronError("Parse cron step error")
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

	return &CronField{_step}, nil
}

func (parser *CronExpressionParser) setDone(index int) {
	if parser.lastDefined == index {
		parser.done = true
	}
}

func (parser *CronExpressionParser) lastSet(index int) bool {
	if parser.lastDefined <= index {
		return true
	}
	return false
}

func (parser *CronExpressionParser) nextSeconds(prev int, field *CronField) string {
	var next int
	next, parser.minuteBump = parser.findNextValue(prev, field.values)
	parser.setDone(secondIndex)
	return alignDigit(next, "0")
}

func (parser *CronExpressionParser) nextMinutes(prev int, field *CronField) string {
	var next int
	if field.isEmpty() && parser.lastSet(minuteIndex) {
		if parser.minuteBump {
			next, parser.hourBump = bumpValue(prev, 59, 1)
			return alignDigit(next, "0")
		}
		return alignDigit(prev, "0")
	}

	next, parser.hourBump = parser.findNextValue(prev, field.values)
	parser.setDone(minuteIndex)
	return alignDigit(next, "0")
}

func (parser *CronExpressionParser) nextHours(prev int, field *CronField) string {
	var next int
	if field.isEmpty() && parser.lastSet(hourIndex) {
		if parser.hourBump {
			next, parser.dayBump = bumpValue(prev, 23, 1)
			return alignDigit(next, "0")
		}
		return alignDigit(prev, "0")
	}

	next, parser.dayBump = parser.findNextValue(prev, field.values)
	parser.setDone(hourIndex)
	return alignDigit(next, "0")
}

func (parser *CronExpressionParser) nextDay(prevWeek int, weekField *CronField,
	prevMonth int, monthField *CronField) int {
	var nextMonth int
	if weekField.isEmpty() && monthField.isEmpty() && parser.lastSet(dayOfWeekIndex) {
		if parser.dayBump {
			nextMonth, parser.monthBump = bumpValue(prevMonth, parser.maxDays, 1)
			return nextMonth
		}
		return prevMonth
	}

	if len(monthField.values) > 0 {
		nextMonth, parser.monthBump = parser.findNextValue(prevMonth, monthField.values)
		parser.setDone(dayOfMonthIndex)
		return nextMonth
	} else if len(weekField.values) > 0 {
		nextWeek, bumpDayOfMonth := parser.findNextValue(prevWeek, weekField.values)
		parser.setDone(dayOfWeekIndex)
		var _step int
		if len(weekField.values) == 1 && weekField.values[0] < prevWeek {
			bumpDayOfMonth = false
		}

		if bumpDayOfMonth && len(weekField.values) == 1 {
			_step = 7
		} else {
			_step = step(prevWeek, nextWeek, 7)
		}
		nextMonth, parser.monthBump = bumpValue(prevMonth, parser.maxDays, _step)
		return nextMonth
	}
	return prevMonth
}

func (parser *CronExpressionParser) nextMonth(prev string, field *CronField) string {
	var next int
	if field.isEmpty() && parser.lastSet(dayOfWeekIndex) {
		if parser.monthBump {
			next, parser.yearBump = bumpLiteral(intVal(months, prev), 12, 1)
			return months[next]
		}
		return prev
	}

	next, parser.yearBump = parser.findNextValue(intVal(months, prev), field.values)
	parser.setDone(monthIndex)
	return months[next]
}

func (parser *CronExpressionParser) nextYear(prev string, field *CronField) string {
	var next int
	if field.isEmpty() && parser.lastSet(yearIndex) {
		if parser.yearBump {
			next, _ = bumpValue(prev, int(^uint(0)>>1), 1)
			return strconv.Itoa(next)
		}
		return prev
	}

	next, halt := parser.findNextValue(prev, field.values)
	if halt != false {
		panic("Out of expression range error")
	}

	return strconv.Itoa(next)
}

func bumpLiteral(iprev int, max int, step int) (int, bool) {
	bumped := iprev + step
	if bumped > max {
		if bumped%max == 0 {
			return iprev, true
		}
		return (bumped % max), true
	}

	return bumped, false
}

// returns bumped value, bump next
func bumpValue(prev interface{}, max int, step int) (int, bool) {
	var iprev, bumped int

	switch prev.(type) {
	case string:
		iprev, _ = strconv.Atoi(prev.(string))
	case int:
		iprev = prev.(int)
	default:
		panic("Unknown type at bumpValue")
	}

	bumped = iprev + step
	if bumped > max {
		return bumped % max, true
	}

	return bumped, false
}

// returns next value, bump next
func (parser *CronExpressionParser) findNextValue(prev interface{}, values []int) (int, bool) {
	var iprev int

	switch prev.(type) {
	case string:
		iprev, _ = strconv.Atoi(prev.(string))
	case int:
		iprev = prev.(int)
	default:
		panic("Unknown type at findNextValue")
	}

	if len(values) == 0 {
		return iprev, false
	}

	for _, element := range values {
		if parser.done {
			if element >= iprev {
				return element, false
			}
		} else {
			if element > iprev {
				parser.done = true
				return element, false
			}
		}
	}

	return values[0], true
}
