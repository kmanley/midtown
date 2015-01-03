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

// TODO: port e.g. ":9999"
func StartWorkerAPI(model *Model, port string, wg *sync.WaitGroup) {
	rpc.Register(NewWorkerApi(model))
	glog.Infof("serving worker API on %s", port)
	ln, err := net.Listen("tcp", port)
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
