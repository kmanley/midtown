package common

import (
	"time"
)

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

// high-level job info for web UI
type JobSummary struct {
	Id              JobID
	Description     string
	Ctrl            *JobControl
	Created         time.Time
	Started         time.Time
	Suspended       time.Time
	Finished        time.Time
	Error           string
	NumTasks        int
	NumIdleTasks    int
	NumActiveTasks  int
	NumDoneOkTasks  int
	NumDoneErrTasks int
	PctComplete     int
}

const RESOLUTION = 10 * time.Millisecond

func (this *JobSummary) Waittime() time.Duration {
	if this.Started.IsZero() {
		return time.Now().Truncate(RESOLUTION).Sub(this.Created.Truncate(RESOLUTION))
	} else {
		return this.Started.Truncate(RESOLUTION).Sub(this.Created.Truncate(RESOLUTION))
	}
}

func (this *JobSummary) Runtime() time.Duration {
	if this.Finished.IsZero() {
		return time.Now().Truncate(RESOLUTION).Sub(this.Started.Truncate(RESOLUTION))
	} else {
		return this.Finished.Truncate(RESOLUTION).Sub(this.Started.Truncate(RESOLUTION))
	}
}

type JobSummaryList []*JobSummary
