// TODO: at some point try to lowercase (not export) as much as possible; don't want to do it yet
// bc not sure of implications for rpc/json serialization
package midtown

import (
	_ "fmt"
	"time"
)

type WorkerStats struct {
	Version   float32
	OSVersion string
	CurrDisk  uint64
	CurrMem   uint64
	CurrCpu   uint8
}

func (this *WorkerStats) reset() {
	this.Version = 0
	this.OSVersion = ""
	this.CurrDisk = 0
	this.CurrMem = 0
	this.CurrCpu = 0
}

type Worker struct {
	Name        string
	CurrJob     JobID
	CurrTask    int
	Stats       WorkerStats
	LastContact time.Time
}

type WorkerMap map[string]*Worker

func NewWorker(name string) *Worker {
	return &Worker{Name: name}
}

func (this *Worker) String() string {
	return this.Name
}

func (this *Worker) assignTask(task *Task) {
	this.CurrJob = task.Job
	this.CurrTask = task.Seq
	//this.updateLastContact()
}

func (this *Worker) reset() {
	this.CurrJob = ""
	this.CurrTask = 0
	// NOTE: we don't reset stats
	//this.updateLastContact()
}

func (this *Worker) isWorking() bool {
	return len(this.CurrJob) > 0
}

func (this *Worker) updateLastContact() {
	this.LastContact = time.Now()
}

func (this *Worker) elapsedSinceLastContact() time.Duration {
	return time.Since(this.LastContact)
}
