package quartz_test

import (
	"errors"
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

func TestCronInvalidMonthDay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, expression, previous, next string
	}{
		{"stepped months after year rollover", "0 0 2 31 4/6 ?", "2026-11-03", "2027-10-31"},
		{"stepped months in current year", "0 0 2 31 4/6 ?", "2026-03-03", "2026-10-31"},
		{"any month", "0 0 2 31 * ?", "2026-04-01", "2026-05-31"},
		{"month list", "0 0 2 31 2,4,10 ?", "2026-01-01", "2026-10-31"},
		{"leap year", "0 0 2 29 2 ?", "2025-01-01", "2028-02-29"},
		{"century leap year", "0 0 2 29 2 ?", "2097-01-01", "2104-02-29"},
		{"selected leap year", "0 0 2 29 2 ? 2027,2028", "2026-01-01", "2028-02-29"},
		{"same month valid day", "0 0 2 30,31 4/6 ?", "2026-03-03", "2026-04-30"},
		{"valid leap day", "0 0 2 29 2 ?", "2028-01-01", "2028-02-29"},
		{"same day later hour", "0 0 2 31 10 ?", "2026-10-31", "2026-10-31"},
		{"impossible day", "0 0 2 31 4 ?", "2026-01-01", ""},
		{"impossible selected year", "0 0 2 29 2 ? 2027", "2026-01-01", ""},
	}
	locations := []*time.Location{time.UTC, time.FixedZone("UTC+0530", 5*3600+30*60)}
	for _, loc := range locations {
		for _, tt := range tests {
			t.Run(loc.String()+"/"+tt.name, func(t *testing.T) {
				trigger, err := quartz.NewCronTriggerWithLoc(tt.expression, loc)
				if err != nil {
					t.Fatal(err)
				}
				previous, err := time.ParseInLocation("2006-01-02", tt.previous, loc)
				if err != nil {
					t.Fatal(err)
				}
				next, err := trigger.NextFireTime(previous.UnixNano())
				if tt.next == "" {
					if !errors.Is(err, quartz.ErrTriggerExpired) {
						t.Fatalf("NextFireTime = (%v, %v), want expired", time.Unix(0, next).In(loc), err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				want, err := time.ParseInLocation("2006-01-02 15:04:05", tt.next+" 02:00:00", loc)
				if err != nil {
					t.Fatal(err)
				}
				if got := time.Unix(0, next).In(loc); !got.Equal(want) {
					t.Fatalf("NextFireTime = %v, want %v", got, want)
				}
			})
		}
	}
}
