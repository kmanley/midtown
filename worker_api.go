package midtown

import (
	"fmt"
	"github.com/golang/glog"
	"net"
	"net/rpc"
)

type WorkerApi struct {
	model    *Model
	listener net.Listener
	stopping bool
}

var workerApi *WorkerApi

func (this *WorkerApi) GetWorkerTask(workerName string, task **WorkerTask) error {
	t, err := this.model.GetWorkerTask(workerName)
	if t == nil && err == nil {
		// gob can't transmit a top level nil, so return an empty task
		// instead; the worker will have to interpret that as 'no task'
		*task = &WorkerTask{}
	} else {
		*task = t
	}
	return err
}

func (this *WorkerApi) SetTaskDone(result *TaskResult, ok *bool) error {
	err := this.model.SetTaskDone(result.WorkerName, result.Job, result.Seq,
		result.Result, result.Stderr,
		result.Error)
	*ok = (err == nil)
	return err
}

func StartWorkerApi(model *Model, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // TODO: allow specifying iface to bind to
	if err != nil {
		glog.Errorf("worker API listen error: %s", err)
		return
	}
	workerApi = &WorkerApi{model, listener, false}
	rpc.Register(workerApi)
	glog.Infof("serving worker API on %d", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !workerApi.stopping {
				glog.Errorf("accept error: %s", err)
			}
			break
		}
		go rpc.ServeConn(conn)
	}
	glog.Info("worker API server stopped")
}

func StopWorkerApi() {
	glog.Infof("stopping worker API server...")
	workerApi.stopping = true
	workerApi.listener.Close()
}
