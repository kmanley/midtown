package midtown

import (
	"bytes"
	_ "container/heap"
	"encoding/gob"
	_ "fmt"
	_ "regexp"
	"time"
)

type Context map[string]string

type JobID string

type JobControl struct {
	MaxConcurrency         int
	StartTime              time.Time
	ContinueJobOnTaskError bool
	RemoteDir              string
	WorkerNameRegex        string
	//CompiledWorkerNameRegex *regexp.Regexp
	// TODO: consider OSRegex as well, to limit to Workers matching a particular OS/version
	//ProcessPriority int
	// TODO: later
	//AssignSingleTaskPerWorker bool
	//TaskWorkerAssignment      map[string][]uint32
	Priority int8          // higher value means higher priority
	Timeout  time.Duration // seconds
	//TaskTimeout          float64 // seconds
	//TaskSeemsHungTimeout uint32
	//AbandonedJobTimeout  uint32
	//MaxTaskReallocations uint8
}

type JobDefinition struct {
	Cmd         string
	Args        []string
	Data        []interface{}
	Description string
	Ctx         *Context
	Ctrl        *JobControl
}

/*
func (this *JobDefinition) String() string {
	return fmt.Sprintf("%s, %v, %s", this.Cmd, this.Data, this.Description) + fmt.Sprintf("%+v", *this.Ctx) +
		fmt.Sprintf("%+v", *this.Ctrl)
}
*/
/*
const (
	JOB_WAITING = iota
	JOB_RUNNING
	JOB_SUSPENDED
	JOB_DONE_OK
	JOB_DONE_ERR // NOTE: cancelled is a subset of JOB_DONE_ERR
)

var JOB_STATES []string = []string{
	"JOB_WAITING",
	"JOB_RUNNING",
	"JOB_SUSPENDED",
	"JOB_DONE_OK",
	"JOB_DONE_ERR"}
*/

type Job struct {
	Id          JobID
	Cmd         string
	Args        []string
	Description string
	Ctrl        *JobControl
	Ctx         *Context
	Created     time.Time
	Started     time.Time
	//Suspended      time.Time
	//Retried        time.Time // note: retried or resumed
	//Cancelled      time.Time
	//CancelReason   string // reason job was cancelled
	Finished time.Time
	//LastClientPoll time.Time
	NumTasks int
	Error    string
	// NOTE: IdleTasks, ActiveTasks, CompletedTasks are never stored in bolt (tasks
	// are stored in separate buckets for performance). These fields are only
	// filled out when returning a Job in the API
	// are only filled by GetJob
	//IdleTasks    TaskList
	//ActiveTasks  TaskList
	//DoneOkTasks  TaskList
	//DoneErrTasks TaskList
	Tasks TaskList
}

type JobList []*Job

type ByPriority JobList

func (a ByPriority) Len() int      { return len(a) }
func (a ByPriority) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPriority) Less(i, j int) bool {
	if a[i].Ctrl.Priority > a[j].Ctrl.Priority {
		return true
	} else if a[i].Ctrl.Priority == a[j].Ctrl.Priority {
		return a[i].Created.Before(a[j].Created)
	}
	return false
}

func (this *Job) ToBytes() ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(this)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func (this *Job) FromBytes(data []byte) error {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	err := dec.Decode(this)
	if err != nil {
		return err
	}
	return nil
}

func NewJob(jobId JobID, cmd string, args []string, description string, data []interface{}, ctx *Context, ctrl *JobControl) (*Job, error) {
	now := time.Now()
	if ctx == nil {
		ctx = &Context{}
	}
	if ctrl == nil {
		ctrl = &JobControl{}
	}

	// TODO: handle AssignSingleTaskPerWorker
	/*
		if ctrl.WorkerNameRegex != "" {
			r, err := regexp.Compile(ctrl.WorkerNameRegex)
			if r == nil {
				ctrl.CompiledWorkerNameRegex = r
			} else {
				return nil, err // TODO: handle error properly
			}
		}
	*/

	newJob := &Job{Id: jobId, Cmd: cmd, Args: args, Description: description, Ctx: ctx,
		Ctrl: ctrl, Created: now}

	newJob.NumTasks = len(data) // TODO: if <= 0, return error and don't push job on heap
	//newJob.RunningTasks = make(TaskMap)
	//newJob.CompletedTasks = make(TaskMap)
	//for i, taskData := range data {
	//	heap.Push(&(newJob.IdleTasks), NewTask(jobID, i, taskData))
	//}
	return newJob, nil
}

/*
type JobMap map[JobID]*Job

func NewJob(jobID JobID, cmd string, description string, data []interface{}, ctx *Context,
	ctrl *JobControl) (*Job, error) {
	now := time.Now()
	if ctx == nil {
		ctx = &Context{}
	}
	if ctrl == nil {
		ctrl = &JobControl{}
	}
	// TODO: handle AssignSingleTaskPerWorker
	if ctrl.WorkerNameRegex != "" {
		r, err := regexp.Compile(ctrl.WorkerNameRegex)
		if r == nil {
			ctrl.CompiledWorkerNameRegex = r
		} else {
			return nil, err // TODO: handle error properly
		}
	}

	newJob := &Job{ID: jobID, Cmd: cmd, Description: description, Ctx: *ctx,
		Ctrl: *ctrl, Created: now}

	newJob.NumTasks = len(data) // TODO: if <= 0, return error and don't push job on heap
	newJob.RunningTasks = make(TaskMap)
	newJob.CompletedTasks = make(TaskMap)
	for i, taskData := range data {
		heap.Push(&(newJob.IdleTasks), NewTask(jobID, i, taskData))
	}
	return newJob, nil
}

func (this *Job) allocateTask(worker *Worker) *Task {
	if this.IdleTasks.Len() < 1 {
		return nil
	}
	now := time.Now()
	task := heap.Pop(&(this.IdleTasks)).(*Task)
	// TODO: replace all calls to task.Seq to a getter func
	this.RunningTasks[task.Seq] = task
	task.start(worker)
	if this.Started.IsZero() {
		this.Started = now
	}
	worker.assignTask(task)
	return task
}

func (this *Job) numIdleTasks() int {
	return this.IdleTasks.Len()
}

func (this *Job) numRunningTasks() int {
	return len(this.RunningTasks)
}

func (this *Job) getRunningTask(seq int) *Task {
	task, found := this.RunningTasks[seq]
	if !found {
		return nil
	}
	return task
}

func (this *Job) setTaskDone(worker *Worker, task *Task, result interface{},
	stdout string, stderr string, err error) {
	//if _, found := this.RunningTasks[task.Seq]; !found {
	//	// model must check task is running before calling job.setTaskDone
	//	panic(fmt.Sprintf("task %d not running in job %s", task.Seq, this.ID))
	//}
	now := time.Now()
	task.finish(result, stdout, stderr, err)
	this.CompletedTasks[task.Seq] = task
	delete(this.RunningTasks, task.Seq)
	if err != nil {
		this.NumErrors += 1
	}
	// if all tasks are completed, or we have an error and the job is setup to not continue on error
	// then mark the job as finished now
	if (len(this.CompletedTasks) == this.NumTasks) || (this.NumErrors > 0 && !this.Ctrl.ContinueJobOnTaskError) {
		this.Finished = now
	}
	worker.reset()
}

func (this *Job) suspend(graceful bool) {
	this.Suspended = time.Now()
	// in a graceful suspend, any running tasks are allowed to continue to run
	// in a graceless suspend, any running tasks will be terminated
	if !graceful {
		this._stopRunningTasks()
	}
}

func (this *Job) getMaxConcurrency() uint32 {
	return this.Ctrl.MaxConcurrency
}

func (this *Job) setMaxConcurrency(maxcon uint32) {
	this.Ctrl.MaxConcurrency = maxcon
}

func (this *Job) cancel(reason string) {
	this.Cancelled = time.Now()
	this.CancelReason = reason
	this._stopRunningTasks()
}

func (this *Job) _stopRunningTasks() {
	for _, task := range this.RunningTasks {
		this.reallocateRunningTask(task)
	}
}

func (this *Job) reallocateRunningTask(task *Task) {
	task.reset()
	heap.Push(&(this.IdleTasks), task)
	delete(this.RunningTasks, task.Seq)
}

func (this *Job) reallocateCompletedTask(task *Task) {
	task.reset()
	heap.Push(&(this.IdleTasks), task)
	delete(this.CompletedTasks, task.Seq)
}

func (this *Job) reallocateWorkerTask(worker *Worker) {
	task, found := this.RunningTasks[worker.CurrTask]
	if found {
		this.reallocateRunningTask(task)
	}
	worker.reset()
}

// Retries a failed job, or resumes a suspended job. Any tasks that had previously
// failed are retried.
func (this *Job) retry() error {
	// NOTE: resume also retries any tasks that failed prior to the suspend
	state := this.State()
	if !(state == JOB_SUSPENDED || state == JOB_DONE_ERR) {
		return ERR_WRONG_JOB_STATE
	}

	now := time.Now()
	this.Retried = now

	// re-enqueue any failed tasks
	for _, task := range this.CompletedTasks {
		if task.hasError() {
			this.reallocateCompletedTask(task)
		}
	}

	this.NumErrors = 0
	this.Finished = *new(time.Time)
	return nil
}

func (this *Job) State() int {
	if !this.Cancelled.IsZero() {
		return JOB_CANCELLED
	}
	if this.Suspended.After(this.Retried) {
		return JOB_SUSPENDED
	}
	if !this.Finished.IsZero() {
		if this.NumErrors > 0 {
			return JOB_DONE_ERR
		} else {
			return JOB_DONE_OK
		}
	}
	if len(this.RunningTasks) > 0 {
		return JOB_RUNNING
	}
	// NOTE: JOB_WAITING can be returned even
	// if some tasks have been completed, if there are
	// currently no running tasks, e.g. right after a
	// graceful suspend/resume
	return JOB_WAITING
}

func (this *Job) StateString() string {
	return JOB_STATES[this.State()]
}

func (this *Job) isWorking() bool {
	state := this.State()
	if state == JOB_WAITING || state == JOB_RUNNING {
		return true
	}
	return false
}

func (this *Job) isFinalState() bool {
	return !this.isWorking()
}

func (this *Job) getResult() []interface{} {
	state := this.State()
	if state == JOB_DONE_OK {
		res := make([]interface{}, this.NumTasks)
		for seq, task := range this.CompletedTasks {
			res[seq] = task.Outdata
		}
		return res
	} else {
		// TODO:
		return nil
	}
}

//func (this *Job) getErrors() []interface

func (this *Job) getShortestRunningTask() (minTask *Task) {
	now := time.Now()
	minDuration := time.Duration(1<<63 - 1) // start with max duration
	for _, task := range this.RunningTasks {
		elapsed := task.elapsedRunning(now)
		if elapsed < minDuration {
			minTask = task
			minDuration = elapsed
		}
	}
	return
}

func (this *Job) getFailureReason() string {
	switch this.State() {
	case JOB_CANCELLED:
		return this.CancelReason
	case JOB_DONE_ERR:
		return fmt.Sprintf("%d task error(s)", this.NumErrors)
	default:
		return ""
	}
}

func (this *Job) timedOut() error {
	timeout := this.Ctrl.JobTimeout
	if timeout == 0 || !this.isWorking() {
		return nil
	}
	// NOTE: this could be improved for the case where a job is suspended/resumed.
	// Currently the time spent suspended counts toward the timeout.
	elapsed := time.Since(this.Started).Seconds()
	if elapsed > timeout {
		return &ErrorJobTimedOut{Timeout: timeout, Elapsed: elapsed}
	}
	return nil
}

func (this *Job) taskTimedOut(task *Task) error {
	timeout := this.Ctrl.TaskTimeout
	if timeout == 0 {
		return nil
	}
	elapsed := task.elapsedRunning(time.Now()).Seconds()
	if elapsed > timeout {
		return &ErrorTaskTimedOut{Timeout: timeout, Elapsed: elapsed}
	}
	return nil
}

func (this *Job) percentComplete() float32 {
	if this.isWorking() {
		return float32(len(this.CompletedTasks)) / float32(this.NumTasks) * 100.0
	} else {
		return 100.0
	}
}
*/
