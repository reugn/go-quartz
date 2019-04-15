package quartz

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CronTrigger struct {
	fields      []*CronField
	lastDefined int
}

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
	return &CronTrigger{fields, lastDefined}, nil
}

func (ct *CronTrigger) NextFireTime(prev int64) (int64, error) {
	parser := NewCronExpressionParser(ct.lastDefined)
	return parser.nextTime(prev, ct.fields)
}

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

func NewCronExpressionParser(lastDefined int) *CronExpressionParser {
	return &CronExpressionParser{false, false, false, false, false, false,
		lastDefined, 0}
}

type CronField struct {
	values []int
}

func (cf *CronField) isEmpty() bool {
	return len(cf.values) == 0
}

func (cf *CronField) toString() string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cf.values)), ","), "[]")
}

var (
	months      = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	days        = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	daysInMonth = []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// pre-defined cron expressions
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

//<second> <minute> <hour> <day-of-month> <month> <day-of-week> <year>
//<year> field is optional
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
	tfmt := time.Unix(prev, 0).UTC().Format(readDateLayout)
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
	nextTime = ntime.Unix()
	return
}

//the ? wildcard is only used in the day of month and day of week fields
func validateCronExpression(expression string) ([]*CronField, error) {
	var tokens []string
	if value, ok := special[expression]; ok {
		tokens = strings.Split(value, " ")
	} else {
		tokens = strings.Split(expression, " ")
	}
	length := len(tokens)
	if length < 6 || length > 7 {
		return nil, cronError("length")
	}
	if length == 6 {
		tokens = append(tokens, "*")
	}
	if (tokens[3] != "?" && tokens[3] != "*") && (tokens[5] != "?" && tokens[5] != "*") {
		return nil, cronError("day field set twice")
	}
	if tokens[6] != "*" {
		return nil, cronError("year field not supported, use asterisk")
	}
	var err error
	fields := make([]*CronField, 7)
	fields[0], err = parseField(tokens[0], 0, 59)
	fields[1], err = parseField(tokens[1], 0, 59)
	fields[2], err = parseField(tokens[2], 0, 23)
	fields[3], err = parseField(tokens[3], 1, 31)
	fields[4], err = parseField(tokens[4], 1, 12, months)
	fields[5], err = parseField(tokens[5], 0, 6, days)
	fields[6], err = parseField(tokens[6], 1970, 1970*2)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func parseField(field string, min int, max int, translate ...[]string) (*CronField, error) {
	var tr []string
	if len(translate) > 0 {
		tr = translate[0]
	}
	//any value
	if field == "*" || field == "?" {
		return &CronField{[]int{}}, nil
	}
	//single value
	i, err := strconv.Atoi(field)
	if err == nil {
		if inScope(i, min, max) {
			return &CronField{[]int{i}}, nil
		}
		return nil, cronError("single min/max validation")
	}
	//list of values
	if strings.Contains(field, ",") {
		t := strings.Split(field, ",")
		si, err := sliceAtoi(t)
		if err != nil {
			//TODO: translation can fail
			si = indexes(t, tr)
		}
		sort.Ints(si)
		return &CronField{si}, nil
	}
	//range of values
	if strings.Contains(field, "-") {
		var _range []int
		t := strings.Split(field, "-")
		if len(t) != 2 {
			return nil, cronError("parse range")
		}
		from := normalize(t[0], tr)
		to := normalize(t[1], tr)
		if !inScope(from, min, max) || !inScope(to, min, max) {
			return nil, cronError("range min/max validation")
		}
		_range, err = fillRange(from, to)
		if err != nil {
			return nil, err
		}
		return &CronField{_range}, nil
	}
	//step values
	if strings.Contains(field, "/") {
		var _step []int
		t := strings.Split(field, "/")
		if len(t) != 2 {
			return nil, cronError("parse step")
		}
		from := normalize(t[0], tr)
		step := atoi(t[1])
		if !inScope(from, min, max) {
			return nil, cronError("step min/max validation")
		}
		_step, err = fillStep(from, step, max)
		if err != nil {
			return nil, err
		}
		return &CronField{_step}, nil
	}
	//literal single value
	if tr != nil {
		i := intVal(tr, field)
		if i >= 0 {
			if inScope(i, min, max) {
				return &CronField{[]int{i}}, nil
			}
			return nil, cronError("literal min/max validation")
		}
	}
	return nil, cronError("parse")
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
			next, parser.yearBump = bumpLiteral(intVal(months, prev), 11, 1)
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
		panic("out of expression range")
	}
	return strconv.Itoa(next)
}

func bumpLiteral(iprev int, max int, step int) (int, bool) {
	bumped := iprev + step
	if bumped > max {
		if bumped%max == 0 {
			return iprev, true
		}
		return (bumped % max) - 1, true
	}
	return bumped, false
}

// return: bumped value, bump next
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

// return: next value, bump next
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
