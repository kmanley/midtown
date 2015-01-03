package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/kmanley/midtown"
	"net/rpc"
	"os"
	"time"
)

type Worker struct {
	name string
	conn *rpc.Client
}

func (this *Worker) GetNextTask() *midtown.WorkerTask {
	var task midtown.WorkerTask
	err := this.conn.Call("Distributor.GetWorkerTask", this.name, &task)
	if err != nil {
		// TODO:
		return nil
	}
	return &task
}

func (this *Worker) RunTask(task *midtown.WorkerTask) {

}

func main() {
	flag.Parse()
	dest := "127.0.0.1" // TODO: cmdline
	port := "9999"      // TODO: cmdline
	distributor := fmt.Sprintf("%s:%s", dest, port)
	conn, err := rpc.Dial("tcp", distributor)
	if err != nil {
		// TODO:
		glog.Errorf("can't connect to distributor: %s", err)
		os.Exit(1)
	}

	// TODO: allow overriding name on cmdline?
	hostname, err := os.Hostname()
	if err != nil {
		glog.Error("can't get hostname: %s", err)
		os.Exit(1)
	}
	name := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	worker := &Worker{name, conn}
	glog.Infof("worker %s starting", name)

	for {
		task := worker.GetNextTask()
		if task != nil {
			worker.RunTask(task)
		} else {
			// TODO: orderly shutdown, signal handler etc.
			// TODO: exponential backoff
			time.Sleep(1 * time.Second)
		}
	}

}
