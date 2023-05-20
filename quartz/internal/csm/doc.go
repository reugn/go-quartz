// Package csm is and internal packaged focused on solving a single task.
// Given an arbitrary date and cron expression, what is the first following
// instant that fits the expression.
//
// The method CronStateMachine.NextTriggerTime() (in cron_state_machine.go)
// calculates the answer. The solution is not mathematically proven to be correct.
// However, it has been thoroughly tested with multiple complex scenarios.
//
// A date can be though of as a mixed-radix number (https://en.wikipedia.org/wiki/Mixed_radix).
// First, (1) we must check from most significant (year) to least significant (second) for
// any field that is invalid according to the cron expression, and move it forward to the
// next valid value. This resets less significant fields and can overflow, advancing more
// significant fields.
// This process is implemented in CronStateMachine.findForward() (fn_find_forward.go).
//
// Second, if no fields are changed by this process, we must do (2) the smallest step forward
// to find the next valid value. This involves moving forward the least significant field (second),
// taking care to advance the next significant field when the previous overflows.
// This process is implemented in CronStateMachine.next() (fn_next.go).
//
// NOTE: Some precautions must be taken as the "day" value does not have a constant radix. It depends
// on the month and the year. January has 30 days and February 2024 has 29. This is taken into account
// by the DayNode struct and CronStateMachine.next() (fn_next.go).
package csm
