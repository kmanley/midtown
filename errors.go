package midtown

import (
	"errors"
	"fmt"
)

/*
type Err struct {
	code int
	msg  string
}

func (this *Err) Error() string {
	return msg
}
*/

type ErrInvalidJob struct {
	jobId JobID
}

func (this ErrInvalidJob) Error() string {
	return fmt.Sprintf("invalid job %s", this.jobId)
}

type ErrJobNotFinished struct {
	jobId JobID
}

func (this ErrJobNotFinished) Error() string {
	return fmt.Sprintf("job %s isn't finished", this.jobId)
}

type ErrInvalidTask struct {
	jobId JobID
	seq   int
}

func (this ErrInvalidTask) Error() string {
	return fmt.Sprintf("invalid task %s:%d", this.jobId, this.seq)
}

type ErrInvalidWorker struct {
	name string
}

func (this ErrInvalidWorker) Error() string {
	return fmt.Sprintf("invalid worker %s", this.name)
}

type ErrInternal struct {
	msg string
}

func (this ErrInternal) Error() string {
	return fmt.Sprintf("internal error: %s", this.msg)
}

var (
	ErrOneOrMoreTasksFailed = errors.New("One or more tasks failed")
	ErrTaskCanceled         = errors.New("Task canceled")
)

/*
// TODO: create custom error struct to include extra data
var ERR_INVALID_JOB_ID = errors.New("invalid job id")
var ERR_WORKER_OUT_OF_SYNC = errors.New("worker is out of sync")
var ERR_TASK_NOT_RUNNING = errors.New("task is not running")
var ERR_WRONG_JOB_STATE = errors.New("job is in wrong state for requested action")
var ERR_TASK_PREEMPTED_PRIORITY = errors.New("task preempted by a higher priority task")
var ERR_TASK_PREEMPTED_CONCURRENCY = errors.New("task preempted to maintain max concurrency")

type ErrorWrongJobState struct {
	State int
}

func (this *ErrorWrongJobState) Error() string {
	return "wrong job state for requested action: " + JOB_STATES[this.State]
}

type JobNotFinished struct {
	State       int
	PctComplete float32
}

func (this *JobNotFinished) Error() string {
	return fmt.Sprintf("job is not finished (state %s, %.2f%% complete)",
		JOB_STATES[this.State], this.PctComplete)
}

type ErrorJobFailed struct {
	State  int
	Reason string
}

func (this *ErrorJobFailed) Error() string {
	return fmt.Sprintf("job failed (state %s): %s", JOB_STATES[this.State], this.Reason)
}

type ErrorJobTimedOut struct {
	Timeout float64
	Elapsed float64
}

func (this *ErrorJobTimedOut) Error() string {
	return fmt.Sprintf("job timed out after %.1f secs (actual=%.1f secs)", this.Timeout, this.Elapsed)
}

type ErrorTaskTimedOut struct {
	Timeout float64
	Elapsed float64
}

func (this *ErrorTaskTimedOut) Error() string {
	return fmt.Sprintf("task timed out after %.1f secs (actual=%.1f secs)", this.Timeout, this.Elapsed)
}
*/
