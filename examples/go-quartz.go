package main

import (
	"fmt"
	"time"

	"go-quartz/quartz"
)

//implements quartz.Job interface
type PrintJob struct {
	desc string
}

func (pj PrintJob) Description() string {
	return pj.desc
}

func (pj PrintJob) Execute() {
	fmt.Println("Executing " + pj.Description())
}

//demo main
func main() {
	sched := quartz.NewStdScheduler()
	cronTrigger, _ := quartz.NewCronTrigger("1/3 * * * * *")
	sched.Start()
	sched.ScheduleJob(&PrintJob{"Ad hoc Job"}, quartz.NewRunOnceTrigger(time.Second*5))
	sched.ScheduleJob(&PrintJob{"First job"}, quartz.NewSimpleTrigger(time.Second*12))
	sched.ScheduleJob(&PrintJob{"Second job"}, quartz.NewSimpleTrigger(time.Second*6))
	sched.ScheduleJob(&PrintJob{"Third job"}, quartz.NewSimpleTrigger(time.Second*3))
	sched.ScheduleJob(&PrintJob{"Cron job"}, cronTrigger)
	time.Sleep(time.Second * 15)
	sched.Stop()
}
