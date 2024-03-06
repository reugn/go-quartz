package quartz_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/quartz"
)

func TestCronExpression1(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("10/20 15 14 5-10 * ? *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sun Jul 9 14:15:30 2023")
}

func TestCronExpression2(t *testing.T) {
	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("10 5,7,9 14-16 * * ? *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sat Jan 6 15:07:10 2024")
}

func TestCronExpression3(t *testing.T) {
	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 5,7,9 14/2 ? * WED,Sat *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sat Jan 13 16:07:00 2024")
}

func TestCronExpression4(t *testing.T) {
	expression := "0 5,7 14 1 * Sun *"
	_, err := quartz.NewCronTrigger(expression)
	assert.ErrorIs(t, err, quartz.ErrCronParse)
}

func TestCronExpression5(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("* * * * * ? *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sat Apr 22 12:00:50 2023")
}

func TestCronExpression6(t *testing.T) {
	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 14/2 ? * mon/3 *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Thu Feb 1 22:00:00 2024")
}

func TestCronExpression7(t *testing.T) {
	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 5-9 14/2 ? * 3-5 *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Wed Jan 3 22:09:00 2024")
}

func TestCronExpression8(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("*/3 */51 */12 */2 */4 ? *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Mon May 1 12:00:27 2023")
}

func TestCronExpression9(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("*/15 * * ? * 1-7")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sat Apr 22 12:12:30 2023")
}

func TestCronExpression10(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("10,20 10,20 10,20 10,20 6,12 ?")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Tue Dec 10 10:10:20 2024")
}

func TestCronExpression11(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("10,20 10,20 10,20 ? 6,12 3,6")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Fri Jun 23 10:10:20 2023")
}

func TestCronExpression12(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 ? 4,6 SAT,MON")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sat Apr 18 00:00:00 2026")
}

func TestCronExpression13(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 29 2 ?")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 5)
	assert.Equal(t, result, "Wed Feb 29 00:00:00 2040")
}

func TestCronExpression14(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 1 5 ? 2023/2")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 10)
	assert.Equal(t, result, "Wed May 1 00:00:00 2041")
}

func TestCronExpressionMixedRange(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0-2,5,7-9,21-22 * * *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 10)
	assert.Equal(t, result, "Sun Apr 23 21:00:00 2023")
}

func TestCronExpressionMixedStringRange(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 ? * SUN,TUE-WED,Fri-Sat")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 10)
	assert.Equal(t, result, "Sat May 6 00:00:00 2023")
}

func TestCronExpressionStepWithRange(t *testing.T) {
	prev := time.Date(2024, 1, 1, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 5-11/2 * * *")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 10)
	assert.Equal(t, result, "Thu Jan 4 07:00:00 2024")

	cronTrigger, err = quartz.NewCronTrigger("0 0 1,5-11/3 * * *")
	assert.IsNil(t, err)
	result, _ = iterate(prev, cronTrigger, 10)
	assert.Equal(t, result, "Thu Jan 4 05:00:00 2024")
}

func TestCronExpressionExpired(t *testing.T) {
	prev := time.Date(2023, 4, 22, 12, 00, 00, 00, time.UTC).UnixNano()
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 1 1 ? 2023")
	assert.IsNil(t, err)
	_, err = cronTrigger.NextFireTime(prev)
	assert.ErrorIs(t, err, quartz.ErrTriggerExpired)
}

func TestCronExpressionWithLoc(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	assert.IsNil(t, err)
	prev := time.Date(2023, 4, 29, 12, 00, 00, 00, loc).UnixNano()
	cronTrigger, err := quartz.NewCronTriggerWithLoc("0 5 22-23 * * Sun *", loc)
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Mon Oct 16 03:05:00 2023") // Result comparison is in UTC time
}

func TestCronExpressionWithLoc2(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	assert.IsNil(t, err)
	prev := time.Date(2023, 4, 29, 12, 00, 00, 00, loc).UnixNano()
	cronTrigger, err := quartz.NewCronTriggerWithLoc("0 0 10 * * Sun *", loc)
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 50)
	assert.Equal(t, result, "Sun Apr 7 14:00:00 2024")
}

func TestCronDaysOfWeek(t *testing.T) {
	daysOfWeek := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	expected := []string{
		"Sun Apr 21 00:00:00 2019",
		"Mon Apr 22 00:00:00 2019",
		"Tue Apr 23 00:00:00 2019",
		"Wed Apr 24 00:00:00 2019",
		"Thu Apr 18 00:00:00 2019",
		"Fri Apr 19 00:00:00 2019",
		"Sat Apr 20 00:00:00 2019",
	}

	for i := 0; i < len(daysOfWeek); i++ {
		cronDayOfWeek(t, daysOfWeek[i], expected[i])
		cronDayOfWeek(t, strconv.Itoa(i+1), expected[i])
	}
}

func cronDayOfWeek(t *testing.T, dayOfWeek, expected string) {
	prev := int64(1555524000000000000) // Wed Apr 17 18:00:00 2019
	expression := fmt.Sprintf("0 0 0 * * %s", dayOfWeek)
	cronTrigger, err := quartz.NewCronTrigger(expression)
	assert.IsNil(t, err)
	nextFireTime, err := cronTrigger.NextFireTime(prev)
	assert.IsNil(t, err)
	assert.Equal(t, time.Unix(nextFireTime/int64(time.Second), 0).UTC().Format(readDateLayout),
		expected)
}

func TestCronYearly(t *testing.T) {
	prev := int64(1555351200000000000)
	cronTrigger, err := quartz.NewCronTrigger("@yearly")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 100)
	assert.Equal(t, result, "Sun Jan 1 00:00:00 2119")
}

func TestCronMonthly(t *testing.T) {
	prev := int64(1555351200000000000)
	cronTrigger, err := quartz.NewCronTrigger("@monthly")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 100)
	assert.Equal(t, result, "Sun Aug 1 00:00:00 2027")
}

func TestCronWeekly(t *testing.T) {
	prev := int64(1555351200000000000)
	cronTrigger, err := quartz.NewCronTrigger("@weekly")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 100)
	assert.Equal(t, result, "Sun Mar 14 00:00:00 2021")
}

func TestCronDaily(t *testing.T) {
	prev := int64(1555351200000000000)
	cronTrigger, err := quartz.NewCronTrigger("@daily")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 1000)
	assert.Equal(t, result, "Sun Jan 9 00:00:00 2022")
}

func TestCronHourly(t *testing.T) {
	prev := int64(1555351200000000000)
	cronTrigger, err := quartz.NewCronTrigger("@hourly")
	assert.IsNil(t, err)
	result, _ := iterate(prev, cronTrigger, 1000)
	assert.Equal(t, result, "Mon May 27 10:00:00 2019")
}

func TestCronExpressionInvalidLength(t *testing.T) {
	_, err := quartz.NewCronTrigger("0 0 0 * *")
	assert.ErrorContains(t, err, "invalid expression length")
}

func TestCronTriggerNilLocationError(t *testing.T) {
	_, err := quartz.NewCronTriggerWithLoc("@daily", nil)
	assert.ErrorContains(t, err, "location is nil")
}

func TestCronExpressionDescription(t *testing.T) {
	expression := "0 0 0 29 2 ?"
	cronTrigger, err := quartz.NewCronTrigger(expression)
	assert.IsNil(t, err)
	assert.Equal(t, cronTrigger.Description(), fmt.Sprintf("CronTrigger::%s::UTC", expression))
}

func TestCronValidateExpression(t *testing.T) {
	assert.IsNil(t, quartz.ValidateCronExpression("@monthly"))
	assert.NotEqual(t, quartz.ValidateCronExpression(""), nil)
}

func TestCronTrimExpression(t *testing.T) {
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

var readDateLayout = "Mon Jan 2 15:04:05 2006"

func iterate(prev int64, cronTrigger *quartz.CronTrigger, iterations int) (string, error) {
	var err error
	for i := 0; i < iterations; i++ {
		prev, err = cronTrigger.NextFireTime(prev)
		// fmt.Println(time.Unix(prev/int64(time.Second), 0).UTC().Format(readDateLayout))
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	return time.Unix(prev/int64(time.Second), 0).UTC().Format(readDateLayout), nil
}

func TestCronExpressionError(t *testing.T) {
	tests := []string{
		"*/X * * * * *",
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := quartz.NewCronTrigger(test)
			assert.ErrorIs(t, err, quartz.ErrCronParse)
		})
	}
}
