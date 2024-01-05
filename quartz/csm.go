package quartz

import (
	"time"

	CSM "github.com/reugn/go-quartz/internal/csm"
)

func makeCSMFromFields(prev time.Time, fields []*cronField) *CSM.CronStateMachine {
	year := CSM.NewCommonNode(prev.Year(), 0, 999999, fields[6].values)
	month := CSM.NewCommonNode(int(prev.Month()), 1, 12, fields[4].values)
	var day *CSM.DayNode
	if len(fields[5].values) != 0 {
		day = CSM.NewWeekdayNode(prev.Day(), 1, 31, fields[5].values, month, year)
	} else {
		day = CSM.NewMonthdayNode(prev.Day(), 1, 31, fields[3].values, month, year)
	}
	hour := CSM.NewCommonNode(prev.Hour(), 0, 59, fields[2].values)
	minute := CSM.NewCommonNode(prev.Minute(), 0, 59, fields[1].values)
	second := CSM.NewCommonNode(prev.Second(), 0, 59, fields[0].values)

	csm := CSM.NewCronStateMachine(second, minute, hour, day, month, year)
	return csm
}
