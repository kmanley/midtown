// TODO: at some point try to lowercase (not export) as much as possible; don't want to do it yet
// bc not sure of implications for rpc/json serialization
package midtown

import (
	"bytes"
	"encoding/gob"
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
	CurrTask    *Task
	Stats       *WorkerStats
	LastContact time.Time
}

type WorkerList []*Worker

func NewWorker(name string) *Worker {
	return &Worker{Name: name}
}

func (this *Worker) ToBytes() ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(this)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func (this *Worker) FromBytes(data []byte) error {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	err := dec.Decode(this)
	if err != nil {
		return err
	}
	return nil
}

//func (this *Worker) String() string {
//	return this.Name
//}

func (this *Worker) setTask(task *Task) {
	this.CurrTask = task
}

func (this *Worker) isWorking() bool {
	return this.CurrTask != nil
}

func (this *Worker) updateLastContact() {
	this.LastContact = time.Now()
}

func (this *Worker) elapsedSinceLastContact() time.Duration {
	return time.Since(this.LastContact)
}
