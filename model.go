/* TODO:

boltdb storage

bucket:
	CompletedJobs
		key=jobid, value=job
	ActiveJobs
		key=jobid, value=job
	<jobid>
		Idle (nested bucket)
			key=seq, value=Task
		Active (nested bucket)
			key=seq, value=Task
		Completed (nested bucket)
			key=seq, value=Task
	Workers
	    key=name, value=Worker

	jobid is date based so we can easily retrieve jobs by date/time

worker stuff is persisted to db only so it can participate in transactions; it's
cleared on startup

need another set of apis for pulling bulk data from db for gui

check group about how to serialize error over json, or just serialize as string
make our own error type that can serialize itself to json?


job errors - do we want to store this in the job or note it by setting error on a task?
  e.g. job timeout

TODO: background goroutine which
 - checks for job timeout (cancel or suspend in this case?)
 - checks for task timeout - in this case set error for the task; the job could continue if continueontaskerror is true
 - check for client abandoned
 - check for hung workers and reallocate tasks

*/

package midtown

import (
	"github.com/boltdb/bolt"
	"github.com/golang/glog"
	"os"
	//"container/heap"
	"fmt"
	//"math"
	"math/rand"
	"strings"
	//"sync"
	"sort"
	"time"
)

const (
	IDLE           = "Idle"
	ACTIVE         = "Active"
	COMPLETED      = "Completed"
	WORKERS        = "Workers"
	MAX_TASK_COUNT = 99999
)

var TASK_KEY_LEN = len(fmt.Sprint(MAX_TASK_COUNT))

type Model struct {
	Db           *bolt.DB
	Fifo         bool
	Workers      WorkerMap
	Prng         *rand.Rand
	LastJobID    JobID
	ShutdownFlag chan bool
}

func (this *Model) Init(dbname string, mode os.FileMode) error {
	db, err := bolt.Open(dbname, mode, nil)
	if err != nil {
		glog.Fatalf("failed to open db %s: %s", dbname, err)
	}
	this.Db = db
	err = this.Db.Update(func(tx *bolt.Tx) error {
		locs := []string{ACTIVE, COMPLETED}
		for idx := range locs {
			_, err := tx.CreateBucketIfNotExists([]byte(locs[idx] + "Jobs"))
			if err != nil {
				return err
			}
		}

		// TODO: think about this; maybe instead of deleting all workers we should just
		// clear their last contact time, then if distrib goes down then comes up state
		// is mostly intact and workers can continue to make progress even while distrib
		// is down.
		// reset Workers bucket, it will be built up again as Workers contact midtownd
		_ = tx.DeleteBucket([]byte(WORKERS))
		_, err = tx.CreateBucket([]byte(WORKERS))
		if err != nil {
			return err
		}

		return nil
	})
	return err

}

func (this *Model) Close() {
	this.Db.Close()
}

// Note: the job ID is purely time-based; this is an essential ordering
// property for our keys in boltdb. When using multiple grids in a single
// environment there is a small chance of duplicate IDs across grids
// TODO: consider grid id prefix?
func (this *Model) NewJobID() (newID JobID) {
	const JOBID_FORMAT = "060102150405.999999"
	for {
		now := time.Now()
		newID = JobID(strings.Replace(now.UTC().Format(JOBID_FORMAT), ".", "", 1))
		for len(newID) < len(JOBID_FORMAT)-1 {
			newID = newID + "0"
		}
		// this is a safeguard when creating jobs quickly in a loop, to ensure IDs are
		// still unique even if clock resolution is too low to always provide a unique
		// value for the format string
		if newID != this.LastJobID {
			break
		} else {
			glog.Warning("paused creating new job ID due to low clock resolution")
		}
	}
	this.LastJobID = newID
	return newID
}

func (this *Model) getOrCreateWorker(tx *bolt.Tx, name string) (*Worker, error) {
	bucket := tx.Bucket([]byte(WORKERS))
	if bucket == nil {
		return nil, fmt.Errorf("failed to open Workers bucket")
	}
	var worker *Worker
	workerBytes := bucket.Get([]byte(name))
	if workerBytes == nil {
		worker = NewWorker(name)
	} else {
		worker := &Worker{}
		err := worker.FromBytes(workerBytes)
		if err != nil {
			return nil, &ErrInternal{"failed to deserialize worker " + string(name)}
		}
	}
	return worker, nil
}

func (this *Model) saveWorker(tx *bolt.Tx, worker *Worker) error {
	bucket := tx.Bucket([]byte(WORKERS))
	if bucket == nil {
		return fmt.Errorf("failed to open Workers bucket")
	}

	// NOTE: saveWorker is only called in functions that
	// are called by workers, so we update the worker's
	// last contact time in this central place
	worker.updateLastContact()

	data, err := worker.ToBytes()
	if err != nil {
		glog.Error("failed to serialize worker", worker.Name, err)
		return err
	}

	err = bucket.Put([]byte(worker.Name), data)
	if err != nil {
		glog.Error("failed to update worker", worker.Name, err)
		return err
	}

	return nil
}

func (this *Model) getJobsBucket(tx *bolt.Tx, which string) (*bolt.Bucket, error) {
	bucket := tx.Bucket([]byte(which + "Jobs"))
	if bucket == nil {
		return nil, fmt.Errorf("failed to open %sJobs bucket", which)
	}
	return bucket, nil
}

func (this *Model) getActiveJobsBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	return this.getJobsBucket(tx, ACTIVE)
}

func (this *Model) getCompletedJobsBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	return this.getJobsBucket(tx, COMPLETED)
}

func (this *Model) getTasksBucket(tx *bolt.Tx, jobId JobID, which string) (*bolt.Bucket, error) {
	bucket := tx.Bucket([]byte(jobId))
	if bucket == nil {
		return nil, fmt.Errorf("failed to open task bucket %s", jobId)
	}
	subBucket := bucket.Bucket([]byte(which))
	if subBucket == nil {
		return nil, fmt.Errorf("failed to open task sub-bucket %s %s", jobId, which)
	}
	return subBucket, nil
}

func (this *Model) getNumTasksInBucket(tx *bolt.Tx, jobId JobID, which string) (int, error) {
	bucket, err := this.getTasksBucket(tx, jobId, which)
	if err != nil {
		return 0, err
	}
	stats := bucket.Stats()
	return stats.KeyN, nil
}

func (this *Model) getIdleTasksBucket(tx *bolt.Tx, jobId JobID) (*bolt.Bucket, error) {
	return this.getTasksBucket(tx, jobId, IDLE)
}

func (this *Model) getActiveTasksBucket(tx *bolt.Tx, jobId JobID) (*bolt.Bucket, error) {
	return this.getTasksBucket(tx, jobId, ACTIVE)
}

func (this *Model) getCompletedTasksBucket(tx *bolt.Tx, jobId JobID) (*bolt.Bucket, error) {
	return this.getTasksBucket(tx, jobId, COMPLETED)
}

func (this *Model) saveJob(bucket *bolt.Bucket, job *Job) error {
	data, err := job.ToBytes()
	if err != nil {
		glog.Error("failed to serialize job", job.Id, err)
		return err
	}

	err = bucket.Put([]byte(job.Id), data)
	if err != nil {
		glog.Error("failed to update job", job.Id, err)
		return err
	}

	return nil
}

// Returns nil if job doesn't exist
func (this *Model) loadJob(bucket *bolt.Bucket, jobId JobID) (*Job, error) {
	jobBytes := bucket.Get([]byte(jobId))
	if jobBytes == nil {
		return nil, &ErrInvalidJob{jobId}
	}

	job := &Job{}
	err := job.FromBytes(jobBytes)
	if err != nil {
		return nil, &ErrInternal{"failed to deserialize job " + string(jobId)}
	}

	return job, nil
}

func (this *Model) saveTask(bucket *bolt.Bucket, task *Task) error {
	data, err := task.ToBytes()
	if err != nil {
		glog.Error("failed to serialize task", task.Job, task.Seq, err)
		return err
	}

	key := fmt.Sprintf("%0*d", TASK_KEY_LEN, task.Seq)
	err = bucket.Put([]byte(key), data)
	if err != nil {
		glog.Errorf("failed to put job %s task %s", task.Job, task.Seq)
		return err
	}

	return nil
}

func (this *Model) loadTask(bucket *bolt.Bucket, jobId JobID, seq int) (*Task, error) {
	key := fmt.Sprintf("%0*d", TASK_KEY_LEN, seq)
	taskBytes := bucket.Get([]byte(key))
	if taskBytes == nil {
		return nil, &ErrInvalidTask{jobId, seq}
	}

	task := &Task{}
	err := task.FromBytes(taskBytes)
	if err != nil {
		return nil, &ErrInternal{fmt.Sprintf("failed to deserialize task %s:%s", jobId, seq)}
	}

	return task, nil
}

func (this *Model) loadTasks(tx *bolt.Tx, jobId JobID, which string, tasks *TaskList) error {
	bucket, err := this.getTasksBucket(tx, jobId, which)
	if err != nil {
		return err
	}

	err = bucket.ForEach(func(key, taskBytes []byte) error {
		task := &Task{}
		err = task.FromBytes(taskBytes)
		if err != nil {
			return err
		}
		*tasks = append(*tasks, task)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// loads currently active jobs and populates the passed in list
// the list is sorted such that highest priority jobs come first
func (this *Model) loadActiveJobs(tx *bolt.Tx, jobs *JobList) error {
	bucket, err := this.getActiveJobsBucket(tx)
	if err != nil {
		return err
	}

	err = bucket.ForEach(func(key, jobBytes []byte) error {
		job := &Job{}
		err = job.FromBytes(jobBytes)
		if err != nil {
			return err
		}
		*jobs = append(*jobs, job)
		return nil
	})
	if err != nil {
		return err
	}

	sort.Sort(ByPriority(*jobs))

	return nil
}

func (this *Model) CreateJob(jobDef *JobDefinition) (JobID, error) {

	job, err := NewJob(this.NewJobID(), jobDef.Cmd, jobDef.Description, jobDef.Data, jobDef.Ctx, jobDef.Ctrl)
	if err != nil {
		return "", err
	}

	err = this.Db.Update(func(tx *bolt.Tx) error {

		// Create the task buckets
		taskBucket, err := tx.CreateBucket([]byte(job.Id))
		if err != nil {
			return err
		}
		locs := []string{IDLE, ACTIVE, COMPLETED}
		for idx := range locs {
			_, err := taskBucket.CreateBucket([]byte(locs[idx]))
			if err != nil {
				return err
			}
		}

		activeJobsBucket, err := this.getActiveJobsBucket(tx)
		if err != nil {
			return err
		}

		if err = this.saveJob(activeJobsBucket, job); err != nil {
			return err
		}

		idleTasksBucket, err := this.getIdleTasksBucket(tx, job.Id)
		if err != nil {
			return err
		}

		for i, taskData := range jobDef.Data {
			task := NewTask(job.Id, i, taskData)
			err = this.saveTask(idleTasksBucket, task)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return job.Id, nil
}

//func (this *Model) GetWorkerTask(workerName string) (*WorkerTask, error) {

//	err := this.Db.Update(func(tx *bolt.Tx) error {

//		worker, err := this.getOrCreateWorker(tx, workerName)
//		if err != nil {
//			return err
//		}

//		/* TODO:
//		if worker.isWorking() {
//			// Out of sync; we think this worker already has a task but it is requesting
//			// a new one. We need to mark the task we thought that worker was working on
//			// as idle so it gets picked up by a different worker.
//			reallocateWorkerTask(worker)
//		}
//		*/

//		job := getJobForWorker(workerName)
//		if job == nil {
//			// no work available right now
//			return nil
//		}
//		task := job.allocateTask(worker)
//		ret := &WorkerTask{job.ID, task.Seq, job.Cmd, task.Indata, &job.Ctx}
//		return ret

// TODO: update task, job, worker in bolt

//	})

//}

func (this *Model) getJobForWorker(tx *bolt.Tx, workerName string) (*Job, error) {
	now := time.Now()
	activeJobs := make(JobList, 0, 50)
	candidates := make(JobList, 0, 50)
	err := this.loadActiveJobs(tx, &activeJobs)
	if err != nil {
		return nil, err
	}

	for _, job := range activeJobs {

		// TODO: consider best order of the following clauses for performance
		// TODO: add support for hidden workers?

		// TODO: handle suspended later
		//if !job.isWorking() {
		//	// job is suspended, cancelled, or done
		//	continue
		//}

		if job.Ctrl.StartTime.After(now) {
			// job not scheduled to start yet
			continue
		}

		numIdleTasks, err := this.getNumTasksInBucket(tx, job.Id, IDLE)
		if err != nil {
			return nil, err
		}
		if numIdleTasks < 1 {
			// all this job's tasks are currently being executed
			continue
		}

		if job.Ctrl.MaxConcurrency > 0 {
			numActiveTasks, err := this.getNumTasksInBucket(tx, job.Id, ACTIVE)
			if err != nil {
				return nil, err
			}
			if numActiveTasks >= job.Ctrl.MaxConcurrency {
				// job is configured with a max concurrency and that has been reached
				continue
			}
		}

		// TODO: later // TODO: don't access Job.Ctrl directly, use accessor functions on Job
		//if (job.Ctrl.CompiledWorkerNameRegex != nil) && (!job.Ctrl.CompiledWorkerNameRegex.MatchString(workerName)) {
		//	// job is configured for specific workers and the requesting worker is not a match
		//	continue
		//}

		if len(candidates) > 0 && (job.Ctrl.Priority != candidates[0].Ctrl.Priority) {
			// there are already higher priority candidates so no point in considering this
			// job or any others that will follow
			break
		}
		// if we got this far, then this job is a candidate for selection
		candidates = append(candidates, job)
	}
	if len(candidates) < 1 {
		return nil, nil
	}
	if this.Fifo {
		return candidates[0], nil
	} else {
		return candidates[this.Prng.Intn(len(candidates))], nil
	}
}

func (this *Model) modifyJob(jobId JobID, fn func(*Job) error) error {

	err := this.Db.Update(func(tx *bolt.Tx) error {
		activeJobsBucket, err := this.getActiveJobsBucket(tx)
		if err != nil {
			return err
		}

		job, err := this.loadJob(activeJobsBucket, jobId)
		if err != nil {
			return err
		}

		if err = fn(job); err != nil {
			return err
		}

		if err = this.saveJob(activeJobsBucket, job); err != nil {
			return err
		}

		return nil
	})

	return err
}

/*
func (this *Model) SetJobPriority(jobId JobID, priority int8) error {

	err := this.Db.Update(func(tx *bolt.Tx) error {
		activeJobsBucket, err := tx.CreateBucketIfNotExists([]byte("ActiveJobs"))
		if err != nil {
			glog.Error("failed to create ActiveJobs bucket", err)
			return err
		}

		job, err := this.loadJob(activeJobsBucket, jobId)
		if err != nil {
			return err
		}

		job.Ctrl.Priority = priority

		if err = this.saveJob(activeJobsBucket, job); err != nil {
			return err
		}

		return nil
	})

	return err
}
*/

func (this *Model) SetJobMaxConcurrency(jobId JobID, maxcon int) error {
	return this.modifyJob(jobId, func(job *Job) error { job.Ctrl.MaxConcurrency = maxcon; return nil })
}

func (this *Model) SetJobPriority(jobId JobID, priority int8) error {
	return this.modifyJob(jobId, func(job *Job) error { job.Ctrl.Priority = priority; return nil })
}

func (this *Model) SetJobTimeout(jobId JobID, timeout time.Duration) error {
	return this.modifyJob(jobId, func(job *Job) error { job.Ctrl.Timeout = timeout; return nil })
}

func (this *Model) getJob(jobId JobID, which string) (*Job, error) {
	var job *Job
	var activeTasks, completedTasks TaskList
	err := this.Db.View(func(tx *bolt.Tx) error {
		bucket, err := this.getJobsBucket(tx, which)
		if err != nil {
			return err
		}

		job, err = this.loadJob(bucket, jobId)
		if err != nil {
			return err
		}

		taskLists := []*TaskList{&(job.IdleTasks), &(job.ActiveTasks), &(job.CompletedTasks)}
		locs := []string{IDLE, ACTIVE, COMPLETED}
		for idx := range locs {
			err = this.loadTasks(tx, jobId, locs[idx], taskLists[idx])
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	job.ActiveTasks = activeTasks
	job.CompletedTasks = completedTasks
	return job, nil
}

func (this *Model) GetActiveJob(jobId JobID) (*Job, error) {
	return this.getJob(jobId, ACTIVE)
}

func (this *Model) GetCompletedJob(jobId JobID) (*Job, error) {
	return this.getJob(jobId, COMPLETED)
}

/*

func SuspendJob(jobID JobID, graceful bool) error {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	job, err := getJob(jobID, true)
	if err != nil {
		return err
	}
	if !job.isWorking() {
		return ERR_WRONG_JOB_STATE
	}

	job.suspend(graceful)
	return nil
}

func CancelJob(jobID JobID, reason string) error {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	job, err := getJob(jobID, false) // allow pulling from DB, e.g. could be suspended
	if err != nil {
		return err
	}
	// we allow canceling a suspended job
	if !(job.isWorking() || job.State() == JOB_SUSPENDED) {
		return ERR_WRONG_JOB_STATE
	}
	job.cancel(reason)
	return nil
}

// TODO: confirm cancelled job can't be retried but suspended job can
func RetryJob(jobID JobID) error {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	job, err := getJob(jobID, false)
	if err != nil {
		return err
	}
	if job.isWorking() {
		return ERR_WRONG_JOB_STATE
	}
	return job.retry()
}

// NOTE: caller must hold mutex
func getJobForWorker(workerName string) (job *Job) {
	now := time.Now()
	candidates := make([]*Job, 0, Model.Jobs.Len())
	jobs := Model.Jobs.Copy()
	for jobs.Len() > 0 {
		// note: this pops in priority + createdtime order
		job = (heap.Pop(jobs)).(*Job)

		// TODO: consider best order of the following clauses for performance
		// TODO: add support for hidden workers?
		if !job.isWorking() {
			// job is suspended, cancelled, or done
			continue
		}

		if job.Ctrl.StartTime.After(now) {
			// job not scheduled to start yet
			continue
		}
		if job.IdleTasks.Len() < 1 {
			// all tasks are currently being executed
			continue
		}
		maxcon := job.getMaxConcurrency()
		if (maxcon > 0) && (uint32(len(job.RunningTasks)) >= maxcon) {
			// job is configured with a max concurrency and that has been reached
			continue
		}
		// TODO: don't access Job.Ctrl directly, use accessor functions on Job
		if (job.Ctrl.CompiledWorkerNameRegex != nil) && (!job.Ctrl.CompiledWorkerNameRegex.MatchString(workerName)) {
			// job is configured for specific workers and the requesting worker is not a match
			continue
		}
		if len(candidates) > 0 && (job.Ctrl.JobPriority != candidates[0].Ctrl.JobPriority) {
			// there are already higher priority candidates so no point in considering this
			// job or any others that will follow
			break
		}
		// if we got this far, then this job is a candidate for selection
		candidates = append(candidates, job)
	}
	if len(candidates) < 1 {
		return nil
	}
	if Model.Fifo {
		return candidates[0]
	} else {
		return candidates[Model.prng.Intn(len(candidates))]
	}
}



func forgetWorker(workerName string) {
	delete(Model.Workers, workerName)
}

func reallocateWorkerTask(worker *Worker) {
	jobID := worker.CurrJob
	job, err := getJob(jobID, true)
	if err != nil {
		return // TODO: log
	}
	fmt.Println(fmt.Sprintf("reallocating %s task %s %d", worker, worker.CurrJob, worker.CurrTask)) // TODO: proper logging
	job.reallocateWorkerTask(worker)
}

func GetWorkerTask(workerName string) *WorkerTask {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	worker := getOrCreateWorker(workerName)
	if worker.isWorking() {
		// Out of sync; we think this worker already has a task but it is requesting
		// a new one. Since our view is the truth, we need to reallocate the task we
		// thought this worker was working on. Note we don't return an error in this
		// case, we go ahead and allocate a new task
		reallocateWorkerTask(worker)
	}
	job := getJobForWorker(workerName)
	if job == nil {
		// no work available right now
		return nil
	}
	task := job.allocateTask(worker)
	ret := &WorkerTask{job.ID, task.Seq, job.Cmd, task.Indata, &job.Ctx}
	return ret
}

func SetTaskDone(workerName string, jobID JobID, taskSeq int, result interface{},
	stdout string, stderr string, taskError error) error {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()

	worker := getOrCreateWorker(workerName)
	if !(worker.CurrJob == jobID && worker.CurrTask == taskSeq) {
		// Out of sync; the worker is reporting a result for a different job/task to
		// what we think it's working on. Our view is the truth so we reallocate the
		// job we thought the worker was working on and reject its result.
		// TODO: logging
		reallocateWorkerTask(worker)
		// Ignore the error; the worker will request a new task and get back in sync
		return nil
	}

	job, err := getJob(jobID, true)
	if err != nil {
		return err
	}

	task := job.getRunningTask(taskSeq)
	if task == nil {
		// TODO: logging
		return ERR_TASK_NOT_RUNNING
	}

	job.setTaskDone(worker, task, result, stdout, stderr, taskError)

	return nil
}

func CheckJobStatus(workerName string, jobID JobID, taskSeq int) error {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()

	// TODO: much of this function is copied from SetTaskDone, refactor
	worker := getOrCreateWorker(workerName)
	if !(worker.CurrJob == jobID && worker.CurrTask == taskSeq) {
		// Out of sync; the worker is asking about a different job/task to
		// what we think it's working on. Our view is the truth so we reallocate the
		// job we thought the worker was working on
		// TODO: what about jobID, taskSeq - do we need to reallocate that too??
		// TODO: do we really need to store worker.Job, worker.Task? I guess so since we display
		//          workers and what they're working on
		// TODO: logging
		reallocateWorkerTask(worker)
		return ERR_WORKER_OUT_OF_SYNC
	}

	job, err := getJob(jobID, true)
	if err != nil {
		return err
	}

	// Note we don't have to check the job state; only that the Task is
	// still in the jobs runmap. For example job could be suspended; if it were suspended
	// gracefully, the task will still be in the runmap, if it were graceless, the Task
	// would have already been moved to the idlemap
	task := job.getRunningTask(taskSeq)
	if task == nil {
		// TODO: logging
		return ERR_TASK_NOT_RUNNING
	}

	err = job.timedOut()
	if err != nil {
		// TODO: worker should call setTaskDone in response to this, with this same error.
		// That's better than us calling setTaskDone here, since it gives us a chance to
		// capture stdout/stderr and possibly diagnose why the task timed out. We rely on
		// the worker to honor this return code.
		return err
	}

	err = job.taskTimedOut(task)
	if err != nil {
		// TODO: worker should call setTaskDone in response to this, with this same error.
		// That's better than us calling setTaskDone here, since it gives us a chance to
		// capture stdout/stderr and possibly diagnose why the task timed out. We rely on
		// the worker to honor this return code.
		return err
	}

	//if job.clientTimedOut()

	// If there is a higher priority job with idle tasks, then we might need to kill this
	// lower priority task to allow the higher priority task to make progress
	job2 := getJobForWorker(workerName)
	if (job2 != nil) && (job2.Ctrl.JobPriority > job.Ctrl.JobPriority) && (job2.numIdleTasks() > 0) {
		// there is a higher priority task available with idle tasks, so this task
		// should be preempted. However, if there's another running task for this
		// job that has been running for less time, then spare this task and preempt
		// the other (it will be preempted when *its* worker calls checkJobStatus)
		task2 := job.getShortestRunningTask()
		if task2 == task {
			job.reallocateRunningTask(task)
			return ERR_TASK_PREEMPTED_PRIORITY
		}
	}

	// Someone may have modified this job's max concurrency via GUI or API such that
	// it is currently running with too high a concurrency. If so, throttle it back
	// by reallocating this task. Here we don't discern between tasks that have been running
	// a long vs. short time. That would be difficult to do and probably not worth the effort.
	if (job.getMaxConcurrency() > 0) && (uint32(job.numRunningTasks()) > job.getMaxConcurrency()) {
		job.reallocateRunningTask(task)
		return ERR_TASK_PREEMPTED_CONCURRENCY
	}

	// No error - worker should continue working on this task
	return nil
}

func GetJobResult(jobID JobID) ([]interface{}, error) {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()

	job, err := getJob(jobID, false)
	if err != nil {
		return nil, err
	}

	state := job.State()
	switch state {
	case JOB_WAITING, JOB_RUNNING, JOB_SUSPENDED:
		return nil, &JobNotFinished{State: state, PctComplete: job.percentComplete()}
	case JOB_CANCELLED, JOB_DONE_ERR:
		// TODO: support GetJobErrors() for the error case
		return nil, &ErrorJobFailed{State: state, Reason: job.getFailureReason()}
	default:
		res := job.getResult()
		return res, nil
	}
}

// TODO: caller must hold mutex
func getHungWorkers() []*Worker {
	ret := make([]*Worker, 0, int(math.Max(1.0, float64(len(Model.Workers)/10))))
	for _, worker := range Model.Workers {
		if worker.elapsedSinceLastContact() > Model.HUNG_WORKER_TIMEOUT {
			ret = append(ret, worker)
		}
	}
	return ret
}

func ReallocateHungTasks() {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	hungWorkers := getHungWorkers()
	for i := range hungWorkers {
		hungWorker := hungWorkers[i]
		fmt.Println("hung worker", hungWorker) // TODO: proper logging
		reallocateWorkerTask(hungWorker)
		forgetWorker(hungWorker.Name)
	}
}

func PrintStats() {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	fmt.Println(len(Model.jobMap), "job(s)")
	jobs := Model.Jobs.Copy()
	for jobs.Len() > 0 {
		// note: this pops in priority + createdtime order
		job := (heap.Pop(jobs)).(*Job)
		jobID := job.ID
		started := job.Started.Format(time.RFC822)
		if job.Started.IsZero() {
			started = "N/A"
		}
		fmt.Println("job", jobID, job.Description, job.State(), "started:", started, "idle:", job.IdleTasks.Len(),
			"running:", len(job.RunningTasks), "done:", len(job.CompletedTasks))
	}
}
*/

/* TODO:
func sanityCheck() {
	Model.Mutex.Lock()
	defer Model.Mutex.Unlock()
	if len(Model.JobMap) != Model.Jobs.Len() {
		panic("# of entries in JobMap and Jobs don't match")
	}
	jobs := Model.Jobs.Copy()
	for jobs.Len() > 0 {
		// note: this pops in priority + createdtime order
		heapJob := (heap.Pop(jobs)).(*Job)
		mapJob, found := Model.JobMap[heapJob.ID]
		if !found {
			panic("job found in heap but not map")
		}
		if mapJob != heapJob {
			panic("heap job != map job")
		}
		// TODO: lots of other checks to do
	}
}
*/
