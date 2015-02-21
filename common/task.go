// TODO: at some point try to lowercase (not export) as much as possible; don't want to do it yet
// bc not sure of implications for rpc/json serialization
package common

import (
	"bytes"
	"encoding/gob"
	_ "fmt"
	//"github.com/kmanley/midtown/common"
	"time"
)

/*
const (
	TASK_WAITING = iota
	TASK_RUNNING
	TASK_DONE_OK
	TASK_DONE_ERR
)

var TASK_STATES []string = []string{
	"TASK_WAITING",
	"TASK_RUNNING",
	"TASK_DONE_OK",
	"TASK_DONE_ERR"}
*/

type Task struct {
	Job      JobID
	Seq      int
	Indata   interface{}
	Outdata  interface{}
	Started  time.Time
	Finished time.Time
	Worker   string
	Error    string
	// TODO: later
	//ExcludedWorkers map[string]bool
	//Stdout string
	Stderr string
}

type TaskList []*Task

type BySequence TaskList

func (a BySequence) Len() int           { return len(a) }
func (a BySequence) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySequence) Less(i, j int) bool { return a[i].Seq < a[j].Seq }

//type TaskMap map[int]*Task

func NewTask(jobID JobID, seq int, data interface{}) *Task {
	// placeholder in case we need more initialization logic later
	return &Task{Job: jobID, Seq: seq, Indata: data}
}

func (this *Task) ToBytes() ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(this)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func (this *Task) FromBytes(data []byte) error {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	err := dec.Decode(this)
	if err != nil {
		return err
	}
	return nil
}

func (this *Task) Start(worker *Worker) {
	now := time.Now()
	this.Outdata = nil
	this.Started = now
	this.Finished = *new(time.Time)
	this.Worker = worker.Name
	this.Error = ""
	//this.Stdout = ""
	this.Stderr = ""
	worker.SetTask(this)
}

func (this *Task) Reset() {
	this.Outdata = nil
	this.Started = *new(time.Time)
	this.Finished = *new(time.Time)
	this.Worker = ""
	this.Error = ""
	//this.Stdout = ""
	this.Stderr = ""
}

func (this *Task) IsRunning() bool {
	if (!this.Started.IsZero()) && (this.Finished.IsZero()) {
		return true
	}
	return false
}

func (this *Task) Finish(result interface{}, stderr string, err error) {
	now := time.Now()
	this.Outdata = result
	this.Finished = now
	// leave this.Worker alone so there's a record of which worker did the task
	//this.Stdout = stdout
	this.Stderr = stderr
	if err != nil {
		this.Error = err.Error()
	} else {
		this.Error = ""
	}
}

//func (this *Task) hasError() bool {
//	return this.Error != nil
//}

/*
func (this *Task) State() int {
	started, finished := this.Started, this.Finished
	if started.IsZero() {
		return TASK_WAITING
	} else {
		// task has started
		if finished.IsZero() {
			return TASK_RUNNING
		} else {
			if this.hasError() {
				return TASK_DONE_ERR
			} else {
				return TASK_DONE_OK
			}
		}
	}
}

func (this *Task) StateString() string {
	return TASK_STATES[this.State()]
}
*/

/*
// Returns current elapsed time for a running task. To get elapsed time
// for a task that may have completed, use elapsed
func (this *Task) elapsedRunning(now time.Time) time.Duration {
	if !(this.State() == TASK_RUNNING) {
		return 0
	}
	return now.Sub(this.Started)
}

// Returns current elapsed time for a completed task
// Returns 0 for an unstarted or currently running task
func (this *Task) elapsed() time.Duration {
	if this.Finished.IsZero() {
		return 0
	}
	return this.Finished.Sub(this.Started)
}
*/

func (this *Task) Elapsed() time.Duration {
	if this.Started.IsZero() {
		return 0
	}
	if this.Finished.IsZero() {
		return time.Now().Sub(this.Started)
	}
	return this.Finished.Sub(this.Started)
}

type WorkerTask struct {
	Job  JobID
	Seq  int
	Cmd  string
	Args []string
	Dir  string
	Data interface{}
	Ctx  *Context
}

func NewWorkerTask(jobId JobID, seq int, cmd string, args []string, dir string,
	data interface{}, ctx *Context) *WorkerTask {
	// placeholder in case we need more initialization logic later
	return &WorkerTask{jobId, seq, cmd, args, dir, data, ctx}
}

type TaskResult struct {
	WorkerName string
	Job        JobID
	Seq        int
	Result     interface{}
	//Stdout     string
	Stderr string
	Error  error
}

func NewTaskResult(workerName string, jobId JobID, seq int, res interface{}, //stdout string,
	stderr string, err error) *TaskResult {
	// placeholder in case we need more initialization logic later
	return &TaskResult{workerName, jobId, seq, res, //stdout,
		stderr, err}
}
