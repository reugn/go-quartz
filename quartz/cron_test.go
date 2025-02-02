package quartz_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/quartz"
)

func TestCronExpression(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
	}{
		{
			expression: "10/20 15 14 5-10 * ? *",
			expected:   "Sat Mar 9 14:15:30 2024",
		},
		{
			expression: "10 5,7,9 14-16 * * ? *",
			expected:   "Sat Jan 6 15:07:10 2024",
		},
		{
			expression: "0 5,7,9 14/2 ? * WED,Sat *",
			expected:   "Sat Jan 13 16:07:00 2024",
		},
		{
			expression: "* * * * * ? *",
			expected:   "Mon Jan 1 12:00:50 2024",
		},
		{
			expression: "0 0 14/2 ? * mon/3 *",
			expected:   "Thu Feb 1 22:00:00 2024",
		},
		{
			expression: "0 5-9 14/2 ? * 3-5 *",
			expected:   "Wed Jan 3 22:09:00 2024",
		},
		{
			expression: "*/3 */51 */12 */2 */4 ? *",
			expected:   "Wed Jan 3 00:00:30 2024",
		},
		{
			expression: "*/15 * * ? * 1-7",
			expected:   "Mon Jan 1 12:12:30 2024",
		},
		{
			expression: "10,20 10,20 10,20 10,20 6,12 ?",
			expected:   "Wed Dec 10 10:10:20 2025",
		},
		{
			expression: "10,20 10,20 10,20 ? 6,12 3,6",
			expected:   "Tue Jun 25 10:10:20 2024",
		},
		{
			expression: "0 0 0 ? 4,6 SAT,MON",
			expected:   "Mon Jun 22 00:00:00 2026",
		},
		{
			expression: "0 0 0 29 2 ?",
			expected:   "Fri Feb 29 00:00:00 2228",
		},
		{
			expression: "0 0 0 1 5 ? 2023/2",
			expected:   "Sat May 1 00:00:00 2123",
		},
		{
			expression: "0 0 0-2,5,7-9,21-22 * * *", // mixed range
			expected:   "Sun Jan 7 02:00:00 2024",
		},
		{
			expression: "0 0 0 ? * SUN,TUE-WED,Fri-Sat", // mixed range
			expected:   "Sun Mar 10 00:00:00 2024",
		},
		{
			expression: "0 0 5-11/2 * * *", // step with range
			expected:   "Sun Jan 14 07:00:00 2024",
		},
		{
			expression: "0 0 1,5-11/3 * * *", // step with range
			expected:   "Sun Jan 14 05:00:00 2024",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	for _, tt := range tests {
		test := tt
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, 50)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionExpired(t *testing.T) {
	t.Parallel()
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 1 1 ? 2023")
	assert.IsNil(t, err)
	_, err = cronTrigger.NextFireTime(prev)
	assert.ErrorIs(t, err, quartz.ErrTriggerExpired)
}

func TestCronExpressionWithLoc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
	}{
		{
			expression: "0 5 22-23 * * Sun *",
			expected:   "Mon Oct 16 03:05:00 2023",
		},
		{
			expression: "0 0 10 * * Sun *",
			expected:   "Sun Apr 7 14:00:00 2024",
		},
	}

	loc, err := time.LoadLocation("America/New_York")
	assert.IsNil(t, err)
	prev := time.Date(2023, 4, 29, 12, 00, 00, 00, loc).UnixNano()
	for _, tt := range tests {
		test := tt
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTriggerWithLoc(test.expression, loc)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, 50)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionDaysOfWeek(t *testing.T) {
	t.Parallel()
	tests := []struct {
		dayOfWeek string
		expected  string
	}{
		{
			dayOfWeek: "Sun",
			expected:  "Sun Apr 21 00:00:00 2019",
		},
		{
			dayOfWeek: "Mon",
			expected:  "Mon Apr 22 00:00:00 2019",
		},
		{
			dayOfWeek: "Tue",
			expected:  "Tue Apr 23 00:00:00 2019",
		},
		{
			dayOfWeek: "Wed",
			expected:  "Wed Apr 24 00:00:00 2019",
		},
		{
			dayOfWeek: "Thu",
			expected:  "Thu Apr 18 00:00:00 2019",
		},
		{
			dayOfWeek: "Fri",
			expected:  "Fri Apr 19 00:00:00 2019",
		},
		{
			dayOfWeek: "Sat",
			expected:  "Sat Apr 20 00:00:00 2019",
		},
	}

	for i, tt := range tests {
		n, test := i, tt
		t.Run(test.dayOfWeek, func(t *testing.T) {
			t.Parallel()
			assertDayOfWeek(t, test.dayOfWeek, test.expected)
			assertDayOfWeek(t, strconv.Itoa(n+1), test.expected)
		})
	}
}

func assertDayOfWeek(t *testing.T, dayOfWeek, expected string) {
	t.Helper()
	const prev = int64(1555524000000000000) // Wed Apr 17 18:00:00 2019
	expression := fmt.Sprintf("0 0 0 * * %s", dayOfWeek)
	cronTrigger, err := quartz.NewCronTrigger(expression)
	assert.IsNil(t, err)
	nextFireTime, err := cronTrigger.NextFireTime(prev)
	assert.IsNil(t, err)
	actual := time.Unix(nextFireTime/int64(time.Second), 0).UTC().Format(readDateLayout)
	assert.Equal(t, actual, expected)
}

func TestCronExpressionSpecial(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
	}{
		{
			expression: "@yearly",
			expected:   "Sat Jan 1 00:00:00 2124",
		},
		{
			expression: "@monthly",
			expected:   "Sat May 1 00:00:00 2032",
		},
		{
			expression: "@weekly",
			expected:   "Sun Nov 30 00:00:00 2025",
		},
		{
			expression: "@daily",
			expected:   "Wed Apr 10 00:00:00 2024",
		},
		{
			expression: "@hourly",
			expected:   "Fri Jan 5 16:00:00 2024",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	for _, tt := range tests {
		test := tt
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, 100)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionDayOfMonth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
	}{
		{
			expression: "0 15 10 L * ?",
			expected:   "Mon Mar 31 10:15:00 2025",
		},
		{
			expression: "0 15 10 L-5 * ?",
			expected:   "Wed Mar 26 10:15:00 2025",
		},
		{
			expression: "0 15 10 15W * ?",
			expected:   "Fri Mar 14 10:15:00 2025",
		},
		{
			expression: "0 15 10 1W 1/2 ?",
			expected:   "Wed Jul 1 10:15:00 2026",
		},
		{
			expression: "0 15 10 31W * ?",
			expected:   "Mon Mar 31 10:15:00 2025",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	for _, tt := range tests {
		test := tt
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, 15)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionDayOfWeek(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		expected   string
	}{
		{
			expression: "0 15 10 ? * L",
			expected:   "Sat Oct 26 10:15:00 2024",
		},
		{
			expression: "0 15 10 ? * 5L",
			expected:   "Thu Oct 31 10:15:00 2024",
		},
		{
			expression: "0 15 10 ? * 2#1",
			expected:   "Mon Nov 4 10:15:00 2024",
		},
		{
			expression: "0 15 10 ? * 3#5",
			expected:   "Tue Mar 31 10:15:00 2026",
		},
		{
			expression: "0 15 10 ? * 7#5",
			expected:   "Sat May 30 10:15:00 2026",
		},
	}

	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	for _, tt := range tests {
		test := tt
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()
			cronTrigger, err := quartz.NewCronTrigger(test.expression)
			assert.IsNil(t, err)
			result, _ := iterate(prev, cronTrigger, 10)
			assert.Equal(t, result, test.expected)
		})
	}
}

func TestCronExpressionInvalidLength(t *testing.T) {
	t.Parallel()
	_, err := quartz.NewCronTrigger("0 0 0 * *")
	assert.ErrorContains(t, err, "invalid expression length")
}

func TestCronTriggerNilLocationError(t *testing.T) {
	t.Parallel()
	_, err := quartz.NewCronTriggerWithLoc("@daily", nil)
	assert.ErrorContains(t, err, "location is nil")
}

func TestCronExpressionDescription(t *testing.T) {
	t.Parallel()
	expression := "0 0 0 29 2 ?"
	cronTrigger, err := quartz.NewCronTrigger(expression)
	assert.IsNil(t, err)
	assert.Equal(t, cronTrigger.Description(), fmt.Sprintf("CronTrigger::%s::UTC", expression))
}

func TestCronExpressionValidate(t *testing.T) {
	t.Parallel()
	assert.IsNil(t, quartz.ValidateCronExpression("@monthly"))
	assert.NotEqual(t, quartz.ValidateCronExpression(""), nil)
}

func TestCronExpressionTrim(t *testing.T) {
	t.Parallel()
	expression := "  0 0  10 * *  Sun * "
	assert.IsNil(t, quartz.ValidateCronExpression(expression))
	trigger, err := quartz.NewCronTrigger(expression)
	assert.IsNil(t, err)
	assert.Equal(t, trigger.Description(), "CronTrigger::0 0 10 * * Sun *::UTC")

	expression = " \t\n 0  0 10 *    *  Sun   \n* \r\n  "
	assert.IsNil(t, quartz.ValidateCronExpression(expression))
	trigger, err = quartz.NewCronTrigger(expression)
	assert.IsNil(t, err)
	assert.Equal(t, trigger.Description(), "CronTrigger::0 0 10 * * Sun *::UTC")
}

const readDateLayout = "Mon Jan 2 15:04:05 2006"

func iterate(prev int64, cronTrigger *quartz.CronTrigger, iterations int) (string, error) {
	var err error
	for i := 0; i < iterations; i++ {
		prev, err = cronTrigger.NextFireTime(prev)
		// log.Print(time.Unix(prev/int64(time.Second), 0).UTC().Format(readDateLayout))
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	return time.Unix(prev/int64(time.Second), 0).UTC().Format(readDateLayout), nil
}

func TestCronExpressionParseError(t *testing.T) {
	t.Parallel()
	tests := []string{
		"-1 * * * * *",
		"X * * * * *",
		"* X * * * *",
		"* * X * * *",
		"* * * X * *",
		"* * * * X *",
		"* * * * * X",
		"* * * * * * X",
		"1,X/1 * * * * *",
		"1,X-1 * * * * *",
		"1-2-3 * * * * *",
		"X-2 * * * * *",
		"1-X * * * * *",
		"100-200 * * * * *",
		"1/2/3 * * * * *",
		"1-2-3/4 * * * * *",
		"X-2/4 * * * * *",
		"1-X/4 * * * * *",
		"X/4 * * * * *",
		"*/X * * * * *",
		"200/100 * * * * *",
		"0 5,7 14 1 * Sun *", // day field set twice
		"0 5,7 14 ? * 2#6 *",
		"0 5,7 14 ? * 2#4,4L *",
		"0 0 0 * * -1#1",
		"0 0 0 ? * 1#-1",
		"0 0 0 ? * #1",
		"0 0 0 ? * 1#",
		"0 0 0 * * a#2",
		"0 0 0 * * 50#2",
		"0 5,7 14 ? * 8L *",
		"0 5,7 14 ? * -1L *",
		"0 5,7 14 ? * 0L *",
		"0 15 10 W * ?",
		"0 15 10 0W * ?",
		"0 15 10 32W * ?",
		"0 15 10 W15 * ?",
		"0 15 10 L- * ?",
		"0 15 10 L-a * ?",
		"0 15 10 L-32 * ?",
	}

	for _, tt := range tests {
		test := tt
		t.Run(test, func(t *testing.T) {
			t.Parallel()
			_, err := quartz.NewCronTrigger(test)
			assert.ErrorIs(t, err, quartz.ErrCronParse)
		})
	}
}
