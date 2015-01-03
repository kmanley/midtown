package midtown

import (
	"fmt"
	"github.com/golang/glog"
	"net"
	"net/rpc"
	"sync"
)

type WorkerApi struct {
	model *Model
}

func NewWorkerApi(model *Model) *WorkerApi {
	return &WorkerApi{model}
}

func (this *WorkerApi) GetWorkerTask(workerName string, task **WorkerTask) error {
	t, err := this.model.GetWorkerTask(workerName)
	*task = t
	return err
}

type TaskResult struct {
	WorkerName string
	Job        JobID
	Task       int
	Result     interface{}
	Stdout     string
	Stderr     string
	Error      error
}

func (this *WorkerApi) SetTaskDone(result *TaskResult, reserved *int) error {
	err := this.model.SetTaskDone(result.WorkerName, result.Job, result.Task,
		result.Result, result.Stdout, result.Stderr,
		result.Error)
	*reserved = 0 // TODO: is there a better way when there's nothing to return?
	return err
}

func StartWorkerApi(model *Model, port int, wg *sync.WaitGroup) {
	rpc.Register(NewWorkerApi(model))
	glog.Infof("serving worker API on %d", port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // TODO: allow specifying iface to bind to
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(c)
	}
	wg.Done() // TODO: make sure we do orderly shutdown and call this in all cases
}
