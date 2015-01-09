package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/kmanley/midtown"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Worker struct {
	name string
	conn *rpc.Client
}

func (this *Worker) GetNextTask() *midtown.WorkerTask {
	var task midtown.WorkerTask
	err := this.conn.Call("WorkerApi.GetWorkerTask", this.name, &task)
	if err != nil {
		glog.Errorf("failed to get task: %s", err)
		return nil
	}
	glog.V(1).Infof("got task %s:%d", task.Job, task.Seq)
	spew.Dump(task)
	return &task
}

func (this *Worker) SetTaskDone(taskResult *midtown.TaskResult) error {
	var ok bool
	err := this.conn.Call("WorkerApi.SetTaskDone", taskResult, &ok)
	if err != nil {
		// TODO:
		return err
	}
	glog.V(1).Infof("set task %s:%d done", taskResult.Job, taskResult.Seq)
	return nil
}

func (this *Worker) RunTask(task *midtown.WorkerTask) *midtown.TaskResult {
	taskResult := &midtown.TaskResult{WorkerName: this.name, Job: task.Job, Seq: task.Seq}
	cmd := exec.Command(task.Cmd, task.Args...)
	cmd.Dir = task.Dir
	input := []interface{}{task.Job, task.Seq, task.Data, task.Ctx}
	var indata bytes.Buffer
	enc := json.NewEncoder(&indata)
	err := enc.Encode(input)
	if err != nil {
		glog.Errorf("failed to encode input data %v", input)
		taskResult.Error = err // TODO: wrap in a better error?
		return taskResult
	}
	cmd.Stdin = strings.NewReader(indata.String())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	taskResult.Stderr = stderr.String()
	if err != nil {
		glog.Errorf("cmd.Run failed: %v", err)
		taskResult.Error = err // TODO: wrap in a better error?
		return taskResult
	}
	var outdata interface{}
	dec := json.NewDecoder(strings.NewReader(stdout.String()))
	err = dec.Decode(&outdata)
	if err != nil {
		glog.Errorf("failed to decode stdout: %v", err)
		glog.Error("stdout: %v", stdout.String())
		taskResult.Error = err // TODO: wrap in a better error?
		return taskResult
	}

	taskResult.Result = outdata
	return taskResult
}

func main() {
	flag.Parse()
	dest := "127.0.0.1" // TODO: cmdline
	port := 9998        // TODO: cmdline
	distributor := fmt.Sprintf("%s:%d", dest, port)
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
			// TODO: distinguish between failure to run task and an error returned by
			// the task subproc?
			taskResult := worker.RunTask(task)
			worker.SetTaskDone(taskResult) // TODO: err handling

		} else {
			// TODO: orderly shutdown, signal handler etc.
			// TODO: exponential backoff
			time.Sleep(1 * time.Second)
		}
	}

}
