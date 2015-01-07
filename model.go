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
		Done-OK (nested bucket)
			key=seq, value=Task
		Done-Error (nested bucket)
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
	"errors"
	"sort"
	"time"
)

const (
	IDLE           = "Idle"
	ACTIVE         = "Active"
	COMPLETED      = "Completed"
	DONE_OK        = "Done_OK"
	DONE_ERR       = "Done_Err"
	WORKERS        = "Workers"
	MAX_TASK_COUNT = 99999
)

var TASK_KEY_LEN = len(fmt.Sprint(MAX_TASK_COUNT))

type Model struct {
	Db           *bolt.DB
	Fifo         bool // TODO: consider persisting this setting in bolt
	Prng         *rand.Rand
	LastJobID    JobID
	ShutdownFlag chan bool
}

func (this *Model) Init(dbname string, mode os.FileMode) error {

	this.Prng = rand.New(rand.NewSource(time.Now().Unix()))

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
		// _ = tx.DeleteBucket([]byte(WORKERS))
		_, err = tx.CreateBucketIfNotExists([]byte(WORKERS))
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
		worker = &Worker{}
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

// TODO: add offset, count later to support paging when there could be hundres/thousands of workers
func (this *Model) loadWorkers(tx *bolt.Tx, workers *WorkerList) error {
	bucket := tx.Bucket([]byte(WORKERS))
	if bucket == nil {
		return fmt.Errorf("failed to open Workers bucket")
	}
	err := bucket.ForEach(func(key, workerBytes []byte) error {
		worker := &Worker{}
		err := worker.FromBytes(workerBytes)
		if err != nil {
			return err
		}
		*workers = append(*workers, worker)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// TODO: add offset, count later
func (this *Model) GetWorkers() (WorkerList, error) {
	workers := make(WorkerList, 0, 50)
	err := this.Db.View(func(tx *bolt.Tx) error {
		err := this.loadWorkers(tx, &workers)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return workers, nil
}

func (this *Model) getJobsBucket(tx *bolt.Tx, which string) (*bolt.Bucket, error) {
	bucket := tx.Bucket([]byte(which + "Jobs"))
	if bucket == nil {
		return nil, fmt.Errorf("failed to open %sJobs bucket", which)
	}
	return bucket, nil
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
	ctr := 0
	err = bucket.ForEach(func(_, _ []byte) error {
		ctr += 1
		return nil
	})
	return ctr, nil
	// TODO: use Stats again after bug is fixed
	// stats := bucket.Stats()
	// return stats.KeyN, nil
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

func (this *Model) deleteJob(bucket *bolt.Bucket, job *Job) error {
	err := bucket.Delete([]byte(job.Id))
	return err
}

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
		glog.Error("failed to serialize task ", task.Job, task.Seq, err)
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

func (this *Model) deleteTask(bucket *bolt.Bucket, task *Task) error {
	key := fmt.Sprintf("%0*d", TASK_KEY_LEN, task.Seq)
	err := bucket.Delete([]byte(key))
	return err
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
	bucket, err := this.getJobsBucket(tx, ACTIVE)
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
		locs := []string{IDLE, ACTIVE, DONE_OK, DONE_ERR}
		for idx := range locs {
			_, err := taskBucket.CreateBucket([]byte(locs[idx]))
			if err != nil {
				return err
			}
		}

		activeJobsBucket, err := this.getJobsBucket(tx, ACTIVE)
		if err != nil {
			return err
		}

		if err = this.saveJob(activeJobsBucket, job); err != nil {
			return err
		}

		idleTasksBucket, err := this.getTasksBucket(tx, job.Id, IDLE)
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

func (this *Model) GetWorkerTask(workerName string) (*WorkerTask, error) {

	var job *Job
	var task *Task
	err := this.Db.Update(func(tx *bolt.Tx) error {

		worker, err := this.getOrCreateWorker(tx, workerName)
		if err != nil {
			return err
		}

		/* TODO:
		if worker.isWorking() {
			// Out of sync; we think this worker already has a task but it is requesting
			// a new one. We need to mark the task we thought that worker was working on
			// as idle so it gets picked up by a different worker.
			reallocateWorkerTask(worker)
		}
		*/

		job, err = this.getJobForWorker(tx, worker)
		if job == nil {
			// no work available right now
			return nil
		}

		idleTasksBucket, err := this.getTasksBucket(tx, job.Id, IDLE)
		if err != nil {
			return err
		}

		activeTasksBucket, err := this.getTasksBucket(tx, job.Id, ACTIVE)
		if err != nil {
			return err
		}

		cursor := idleTasksBucket.Cursor()
		_, taskBytes := cursor.First()
		task = &Task{}
		err = task.FromBytes(taskBytes)
		if err != nil {
			return err
		}

		task.start(worker)

		// save task in the active bucket
		err = this.saveTask(activeTasksBucket, task)
		if err != nil {
			return err
		}

		// remove it from the idle bucket
		err = this.deleteTask(idleTasksBucket, task)
		if err != nil {
			return err
		}

		err = this.saveWorker(tx, worker)
		if err != nil {
			return err
		}

		// if this is the first task started for a job, need to update the job too
		if job.Started.IsZero() {
			job.Started = time.Now()
			activeJobsBucket, err := this.getJobsBucket(tx, ACTIVE)
			if err != nil {
				return err
			}
			err = this.saveJob(activeJobsBucket, job)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if job == nil || task == nil {
		return nil, nil
	}

	ret := NewWorkerTask(job.Id, task.Seq, job.Cmd, task.Indata, job.Ctx)
	return ret, nil
}

func (this *Model) getJobForWorker(tx *bolt.Tx, worker *Worker) (*Job, error) {
	now := time.Now()
	activeJobs := make(JobList, 0, 50)
	candidates := make(JobList, 0, 50)
	err := this.loadActiveJobs(tx, &activeJobs)
	if err != nil {
		return nil, err
	}

	for _, job := range activeJobs {

		if len(candidates) > 0 && (job.Ctrl.Priority < candidates[0].Ctrl.Priority) {
			// there are already higher priority candidates so no point in considering this
			// job or any others that will follow
			break
		}

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
		activeJobsBucket, err := this.getJobsBucket(tx, ACTIVE)
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

func (this *Model) SetJobMaxConcurrency(jobId JobID, maxcon int) error {
	return this.modifyJob(jobId, func(job *Job) error { job.Ctrl.MaxConcurrency = maxcon; return nil })
}

func (this *Model) SetJobPriority(jobId JobID, priority int8) error {
	return this.modifyJob(jobId, func(job *Job) error { job.Ctrl.Priority = priority; return nil })
}

func (this *Model) SetJobTimeout(jobId JobID, timeout time.Duration) error {
	return this.modifyJob(jobId, func(job *Job) error { job.Ctrl.Timeout = timeout; return nil })
}

func (this *Model) GetJob(jobId JobID, which string) (*Job, error) {
	var job *Job
	err := this.Db.View(func(tx *bolt.Tx) error {
		bucket, err := this.getJobsBucket(tx, which)
		if err != nil {
			return err
		}

		job, err = this.loadJob(bucket, jobId)
		if err != nil {
			return err
		}

		/*
			taskLists := []*TaskList{&(job.IdleTasks), &(job.ActiveTasks),
				&(job.DoneOkTasks), &(job.DoneErrTasks)}
			locs := []string{IDLE, ACTIVE, DONE_OK, DONE_ERR}
			for idx := range locs {
				err = this.loadTasks(tx, jobId, locs[idx], taskLists[idx])
			}
		*/
		locs := []string{IDLE, ACTIVE, DONE_OK, DONE_ERR}
		for idx := range locs {
			err = this.loadTasks(tx, jobId, locs[idx], &(job.Tasks))
		}
		sort.Sort(BySequence(job.Tasks))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (this *Model) cancelPendingTasks(tx *bolt.Tx, jobId JobID) error {
	taskSaveBucket, err := this.getTasksBucket(tx, jobId, DONE_ERR)
	if err != nil {
		return err
	}
	bucketNames := []string{IDLE, ACTIVE}
	for _, bucketName := range bucketNames {
		bucket, err := this.getTasksBucket(tx, jobId, bucketName)
		if err != nil {
			return err
		}
		cursor := bucket.Cursor()
		for taskSeq, taskBytes := cursor.First(); taskSeq != nil; taskSeq, taskBytes = cursor.Next() {
			pendingTask := &Task{}
			err = pendingTask.FromBytes(taskBytes)
			if err != nil {
				return &ErrInternal{fmt.Sprintf("failed to deserialize pending task %s:%s", jobId, taskSeq)}
			}
			pendingTask.finish(nil, "", "", ErrTaskCanceled)
			err = this.saveTask(taskSaveBucket, pendingTask)
			if err != nil {
				return err
			}
			cursor.Delete() // remove from pending task bucket
		}
	}
	return nil
}

// TODO: when testing, make sure job state is correct whether 1st task or nth task is in error
// and whether ContinueJobOnTaskError is set or not
func (this *Model) SetTaskDone(workerName string, jobId JobID, taskSeq int, result interface{},
	stdout string, stderr string, taskError error) error {

	err := this.Db.Update(func(tx *bolt.Tx) error {

		worker, err := this.getOrCreateWorker(tx, workerName)
		if err != nil {
			return err
		}

		activeJobsBucket, err := this.getJobsBucket(tx, ACTIVE)
		if err != nil {
			return err
		}

		activeTasksBucket, err := this.getTasksBucket(tx, jobId, ACTIVE)
		if err != nil {
			return err
		}

		job, err := this.loadJob(activeJobsBucket, jobId)
		if err != nil {
			return err
		}

		task, err := this.loadTask(activeTasksBucket, jobId, taskSeq)
		if err != nil {
			return err
		}

		// TODO: deal with this
		//if !(worker.CurrJob == jobID && worker.CurrTask == taskSeq) {
		//	// Out of sync; the worker is reporting a result for a different job/task to
		//	// what we think it's working on. Our view is the truth so we reallocate the
		//	// job we thought the worker was working on and reject its result.
		//	// TODO: logging
		//	reallocateWorkerTask(worker)
		//	// Ignore the error; the worker will request a new task and get back in sync
		//	return nil
		//}

		//job.setTaskDone(worker, task, result, stdout, stderr, taskError)
		task.finish(result, stdout, stderr, taskError)

		taskSaveBucketName := DONE_OK
		if taskError != nil {
			taskSaveBucketName = DONE_ERR
		}

		taskSaveBucket, err := this.getTasksBucket(tx, jobId, taskSaveBucketName)
		if err != nil {
			return err
		}

		// save the task in the new bucket...
		err = this.saveTask(taskSaveBucket, task)
		if err != nil {
			return err
		}

		// ...and remove the task from active bucket
		err = this.deleteTask(activeTasksBucket, task)
		if err != nil {
			return err
		}

		// update the worker which now has no task assigned
		worker.setTask(nil)
		err = this.saveWorker(tx, worker)
		if err != nil {
			return err
		}

		// if this task failed and the job isn't configured to continue on
		// task errors then cancel all idle and active tasks
		if taskError != nil && job.Ctrl.ContinueJobOnTaskError == false {
			err = this.cancelPendingTasks(tx, jobId)
			if err != nil {
				return err
			}
		}

		// if job is done, we need to update it too. The job is done if either all of
		// its tasks are done, or a task is marked in error and the job is set to fail
		// on first error
		numDoneOK, err := this.getNumTasksInBucket(tx, jobId, DONE_OK)
		if err != nil {
			return err
		}

		numDoneErr, err := this.getNumTasksInBucket(tx, jobId, DONE_ERR)
		if err != nil {
			return err
		}

		if (numDoneOK + numDoneErr) == job.NumTasks {
			now := time.Now()
			job.Finished = now
			if numDoneErr > 0 {
				job.Error = ErrOneOrMoreTasksFailed.Error()
			}

			completedJobsBucket, err := this.getJobsBucket(tx, COMPLETED)
			if err != nil {
				return err
			}
			err = this.saveJob(completedJobsBucket, job)
			if err != nil {
				return err
			}
			err = this.deleteJob(activeJobsBucket, job)
			if err != nil {
				return err
			}
		}
		return nil

	})

	if err != nil {
		return err
	}

	return nil
}

// Looks for an active job, if not found looks for a completed job.
func (this *Model) findJob(tx *bolt.Tx, jobId JobID) (*Job, error) {
	var job *Job
	// this function can be called on either an active or completed
	// job so we have to check both buckets
	locs := []string{ACTIVE, COMPLETED}
	for _, loc := range locs {
		bucket, err := this.getJobsBucket(tx, loc)
		if err != nil {
			return nil, err
		}

		job, err = this.loadJob(bucket, jobId)
		if err == nil {
			// found it!
			break
		}
	}
	if job == nil {
		return nil, &ErrInvalidJob{jobId}
	} else {
		return job, nil
	}
}

// Gets job result data; this only works on jobs that are completed without errors.
// To get partial results for a running job, call GetPartialJobResult
// To get full info for jobs including which tasks failed, call GetJob
func (this *Model) GetJobResult(jobId JobID) ([]interface{}, error) {
	var res []interface{}
	err := this.Db.View(func(tx *bolt.Tx) error {
		job, err := this.findJob(tx, jobId)
		if err != nil {
			return err
		}
		// job is either done ok, done with error(s), or still running
		if job.Finished.IsZero() {
			return &ErrJobNotFinished{jobId}
		} else if job.Error != "" {
			return errors.New(job.Error)
		} else {
			// all tasks must be in DONE_OK bucket; TODO: assert this
			// they should also already be sorted by seq since bolt sorts by key
			tasks := make(TaskList, 0, job.NumTasks)
			err = this.loadTasks(tx, jobId, DONE_OK, &tasks)
			if err != nil {
				return err
			}
			res = make([]interface{}, job.NumTasks)
			for i, task := range tasks {
				res[i] = task.Outdata
			}
			return nil
		}
	})
	if err != nil {
		return nil, err
	}

	return res, nil

}

// TODO: func (this *Model) GetPartialJobResult(jobID JobID, taskSeqs []int) ([]interface{}, error)
// this can be used by client to do async result fetching, by passing in ids of tasks not yet
// seen.

// TODO: func (this *Model) GetJobInfo(jobID JobID)

/*
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
*/
//}

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


func reallocateWorkerTask(worker *Worker) {
	jobID := worker.CurrJob
	job, err := getJob(jobID, true)
	if err != nil {
		return // TODO: log
	}
	fmt.Println(fmt.Sprintf("reallocating %s task %s %d", worker, worker.CurrJob, worker.CurrTask)) // TODO: proper logging
	job.reallocateWorkerTask(worker)
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
