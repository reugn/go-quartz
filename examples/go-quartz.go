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

func (pj PrintJob) Key() int {
	return quartz.HashCode(pj.Description())
}

func (pj PrintJob) Execute() {
	fmt.Println("Executing " + pj.Description())
}

//demo main
func main() {
	sched := quartz.NewStdScheduler()
	cronTrigger, _ := quartz.NewCronTrigger("1/3 * * * * *")
	cronJob := PrintJob{"Cron job"}
	sched.Start()
	sched.ScheduleJob(&PrintJob{"Ad hoc Job"}, quartz.NewRunOnceTrigger(time.Second*5))
	sched.ScheduleJob(&PrintJob{"First job"}, quartz.NewSimpleTrigger(time.Second*12))
	sched.ScheduleJob(&PrintJob{"Second job"}, quartz.NewSimpleTrigger(time.Second*6))
	sched.ScheduleJob(&PrintJob{"Third job"}, quartz.NewSimpleTrigger(time.Second*3))
	sched.ScheduleJob(&cronJob, cronTrigger)

	time.Sleep(time.Second * 10)

	j, _ := sched.GetScheduledJob(cronJob.Key())
	fmt.Println(j.TriggerDescription)
	fmt.Println(sched.GetJobKeys())
	sched.DeleteJob(cronJob.Key())
	fmt.Println(sched.GetJobKeys())

	time.Sleep(time.Second * 10)
	sched.Stop()
}
