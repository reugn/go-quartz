// "csm" is an internal package focused on solving a single task.
// Given an arbitrary date and cron expression, what is the first following
// instant that fits the expression?
//
// The method CronStateMachine.NextTriggerTime() (cron_state_machine.go)
// computes the answer. The current solution is not proven to be mathematically correct.
// However, it has been thoroughly validated with multiple complex test cases.
//
// A date can be though of as a mixed-radix number (https://en.wikipedia.org/wiki/Mixed_radix).
// First, we must check from most significant (year) to least significant (second) for
// any field that is invalid according to the cron expression, and move it forward to the
// next valid value. This resets less significant fields and can overflow; advancing more
// significant fields. This process is implemented in CronStateMachine.findForward() (fn_find_forward.go).
//
// Second, if no fields are changed by this process, we must perform the smallest possible step forward
// to find the next valid value. This involves moving forward the least significant field (second),
// taking care to advance the next significant field when the previous one overflows. This process is
// implemented in CronStateMachine.next() (fn_next.go).
//
// NOTE: Some precautions must be taken as the "day" value does not have a constant radix. It depends
// on the month and the year. January always has 30 days, while February 2024 has 29. This is taken into account
// by the DayNode struct (day_node.go) and CronStateMachine.next() (fn_next.go).
package csm
