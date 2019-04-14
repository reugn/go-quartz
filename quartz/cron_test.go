package quartz

import (
	"testing"
	"time"
)

func TestCronExpression1(t *testing.T) {
	prev := int64(1554120000)
	result := ""
	cronTrigger, err := NewCronTrigger("10/20 15 14 5-10 * ? *")
	if err != nil {
		t.Fatal(err)
	} else {
		result, _ = iterate(prev, cronTrigger, 1000)
	}
	assertEqual(t, result, "Wed Nov  8 14:15:10 IST 2023")
}

func TestCronExpression2(t *testing.T) {
	prev := int64(1554120000)
	result := ""
	cronTrigger, err := NewCronTrigger("* 5,7,9 14-16 * * ? *")
	if err != nil {
		t.Fatal(err)
	} else {
		result, _ = iterate(prev, cronTrigger, 1000)
	}
	assertEqual(t, result, "Sun Jul 21 15:05:00 IDT 2019")
}

func TestCronExpression3(t *testing.T) {
	prev := int64(1554120000)
	result := ""
	cronTrigger, err := NewCronTrigger("* 5,7,9 14/2 * * Wed,Sat *")
	if err != nil {
		t.Fatal(err)
	} else {
		result, _ = iterate(prev, cronTrigger, 1000)
	}
	assertEqual(t, result, "Wed Nov 20 22:05:00 IST 2019")
}

func TestCronExpression4(t *testing.T) {
	expression := "0 5,7 14 1 * Sun *"
	_, err := NewCronTrigger(expression)
	if err == nil {
		t.Fatalf("%s should fail", expression)
	}
}

func TestCronExpression5(t *testing.T) {
	prev := int64(1554120000)
	result := ""
	cronTrigger, err := NewCronTrigger("* * * * * ? *")
	if err != nil {
		t.Fatal(err)
	} else {
		result, _ = iterate(prev, cronTrigger, 1000)
	}
	assertEqual(t, result, "Mon Apr  1 15:16:40 IDT 2019")
}

func TestCronExpression6(t *testing.T) {
	prev := int64(1554120000)
	result := ""
	cronTrigger, err := NewCronTrigger("* * 14/2 * * Mon/3 *")
	if err != nil {
		t.Fatal(err)
	} else {
		result, _ = iterate(prev, cronTrigger, 1000)
	}
	assertEqual(t, result, "Thu Mar  4 20:00:00 IST 2021")
}

func TestCronExpression7(t *testing.T) {
	prev := int64(1554120000)
	result := ""
	cronTrigger, err := NewCronTrigger("* 5-9 14/2 * * 0-2 *")
	if err != nil {
		t.Fatal(err)
	} else {
		result, _ = iterate(prev, cronTrigger, 1000)
	}
	assertEqual(t, result, "Tue Jul  2 14:09:00 IDT 2019")
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func iterate(prev int64, cronTrigger *CronTrigger, iterations int) (string, error) {
	var err error
	for i := 0; i < iterations; i++ {
		prev, err = cronTrigger.NextFireTime(prev)
		if err != nil {
			return "", err
		}
		// fmt.Println(time.Unix(prev, 0).Format(time.UnixDate))
	}
	return time.Unix(prev, 0).Format(time.UnixDate), nil
}
